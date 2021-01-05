package src

import (
	"fmt"
	"github.com/morgine/wechat_sdk/pkg"
	"github.com/morgine/wechat_sdk/pkg/open_platform"
	"log"
	"net/http"
	"sync"
)

type OpenClient struct {
	configs       *OpenClientConfigs
	publicClients map[string]*PublicClient
	msgCrypt      *pkg.WXBizMsgCrypt
	*Dispatcher
	mu sync.Mutex
}

type OpenClientConfigs struct {
	Appid            string
	Secret           string
	MsgVerifyToken   string           // 消息验证 token
	AesKey           string           // 消息加解密 key
	AesToken         string           // 消息加解密 token
	ComponentStorage ComponentStorage // 开放平台存储器
	AppStorage       AppStorage       // 公众号信息存储器
	Logger           *log.Logger      // 错误日志收集器
}

func NewOpenClient(configs *OpenClientConfigs) (*OpenClient, error) {
	msgCrypt, err := pkg.NewWXBizMsgCrypt(configs.AesToken, configs.AesKey, configs.Appid)
	if err != nil {
		return nil, err
	}
	return &OpenClient{
		configs:       configs,
		publicClients: map[string]*PublicClient{},
		msgCrypt:      msgCrypt,
		Dispatcher:    NewDispatcher(),
		mu:            sync.Mutex{},
	}, nil
}

func (oc *OpenClient) Configs() *OpenClientConfigs {
	return oc.configs
}

// 监听通知消息
func (oc *OpenClient) ListenVerifyTicket(w http.ResponseWriter, r *http.Request) {
	notify, err := open_platform.ListenComponentAuthorizationNotify(r, oc.msgCrypt)
	if err != nil {
		oc.configs.Logger.Println(err)
	} else {
		err = oc.setNotify(notify)
		if err != nil {
			oc.configs.Logger.Println(err)
		} else {
			_, _ = w.Write([]byte("success"))
		}
	}
}

func (oc *OpenClient) setNotify(notify *open_platform.AuthorizationNotify) error {
	switch notify.InfoType {
	case open_platform.EvtComponentVerifyTicket:
		return oc.configs.ComponentStorage.SaveVerifyTicket(notify.ComponentVerifyTicket)
	case open_platform.EvtAuthorized, open_platform.EvtUpdateAuthorized:
		_, err := oc.refreshAppInfo(notify.AuthorizerAppid)
		if err != nil {
			return err
		}
	case open_platform.EvtUnauthorized:
		oc.mu.Lock()
		defer oc.mu.Unlock()
		delete(oc.publicClients, notify.AuthorizerAppid)
		return oc.configs.AppStorage.DelAppInfo(notify.AuthorizerAppid)
	default:
		return nil
	}
	return nil
}

// 获得三方平台 access token
func (oc *OpenClient) getComponentAccessToken() (string, error) {
	token, err := oc.configs.ComponentStorage.GetAccessToken()
	if err != nil {
		return "", err
	}
	now := Now().Unix()
	if token == nil || token.ExpiredAt < now {
		verifyTicket, err := oc.configs.ComponentStorage.GetVerifyTicket()
		if err != nil {
			return "", err
		}
		if verifyTicket == "" {
			return "", fmt.Errorf("component %s: verify ticket is empty", oc.configs.Appid)
		}
		accessToken, err := open_platform.GetComponentAccessToken(oc.configs.Appid, oc.configs.Secret, verifyTicket)
		if err != nil {
			return "", err
		}
		token = &ExpireData{
			Value:     accessToken.Token,
			ExpiredAt: now + accessToken.ExpiresIn - (accessToken.ExpiresIn >> 3), // 过期时间提前 1/8
		}
		err = oc.configs.ComponentStorage.SaveAccessToken(token)
		if err != nil {
			return "", err
		}
	}
	return token.Value, nil
}

// 获得 pre auth code
func (oc *OpenClient) getPreAuthCode() (string, error) {
	//code, err := oc.configs.ComponentStorage.GetPreAuthCode()
	//if err != nil {
	//	return "", err
	//}
	//now := Now().Unix()
	//if code == nil || code.ExpiredAt < now {
	token, err := oc.getComponentAccessToken()
	if err != nil {
		return "", nil
	}
	pac, err := open_platform.CreatePreAuthCode(oc.configs.Appid, token)
	if err != nil {
		return "", err
	}
	return pac.PreAuthCode, nil
	//code = &ExpireData{
	//	Value:     pac.PreAuthCode,
	//	ExpiredAt: now + pac.ExpiresIn - (pac.ExpiresIn >> 3), // 过期时间提前 1/8
	//}
	//err = oc.configs.ComponentStorage.SavePreAuthCode(code)
	//if err != nil {
	//	return "", err
	//}
	//}
	//return code.Value, nil
}

// 获得授权地址
func (oc *OpenClient) ComponentLoginPage(redirect string) (string, error) {
	preAuthCode, err := oc.getPreAuthCode()
	if err != nil {
		return "", err
	}
	return open_platform.ComponentLoginPage(&open_platform.ComponentLoginPageOptions{
		ComponentAppid: oc.configs.Appid,
		PreAuthCode:    preAuthCode,
		RedirectUri:    redirect,
		AuthType:       "",
		BizAppid:       "",
	}), nil
}

// 获得授权信息, 用户授权/未授权都跳回该地址
func (oc *OpenClient) ListenLoginPage(request *http.Request) error {
	code, _ := open_platform.ListenLoginPage(request)
	if code != "" {
		token, err := oc.getComponentAccessToken()
		if err != nil {
			return err
		}
		authInfo, err := open_platform.GetAuthorizationInfo(oc.configs.Appid, code, token)
		if err != nil {
			return err
		}
		appAccessToken := &AppAccessToken{
			AccessToken:  authInfo.AuthorizerAccessToken,
			ExpireAt:     Now().Unix() + authInfo.ExpiresIn - (authInfo.ExpiresIn >> 3),
			RefreshToken: authInfo.AuthorizerRefreshToken,
		}
		err = oc.configs.ComponentStorage.SaveAppAccessToken(authInfo.AuthorizerAppid, appAccessToken)
		if err != nil {
			return err
		}
	}
	return nil
}

// 获得公众号 access token
func (oc *OpenClient) getAppAccessToken(appid string) (string, error) {
	appAccessToken, err := oc.configs.ComponentStorage.GetAppAccessToken(appid)
	if err != nil {
		return "", err
	}
	if appAccessToken == nil {
		return "", fmt.Errorf("%s unauthorized or access token missing", appid)
	} else {
		now := Now().Unix()
		if appAccessToken.ExpireAt < now {
			componentToken, err := oc.getComponentAccessToken()
			if err != nil {
				return "", err
			}
			authToken, err := open_platform.RefreshAuthorizerToken(oc.configs.Appid, appid, appAccessToken.RefreshToken, componentToken)
			if err != nil {
				return "", err
			}
			appAccessToken = &AppAccessToken{
				AccessToken:  authToken.AuthorizerAccessToken,
				ExpireAt:     Now().Unix() + authToken.ExpiresIn - (authToken.ExpiresIn >> 3),
				RefreshToken: authToken.AuthorizerRefreshToken,
			}
			err = oc.configs.ComponentStorage.SaveAppAccessToken(appid, appAccessToken)
			if err != nil {
				return "", err
			}
		}
		return appAccessToken.AccessToken, nil
	}
}

// 获取并保存公众号信息
func (oc *OpenClient) refreshAppInfo(appid string) (*open_platform.AuthorizerInfo, error) {
	token, err := oc.getComponentAccessToken()
	if err != nil {
		return nil, err
	}
	authInfo, err := open_platform.GetAuthorizerInfo(oc.configs.Appid, appid, token)
	if err != nil {
		return nil, err
	}
	info := &authInfo.AuthorizerInfo
	err = oc.configs.AppStorage.SaveAppInfo(appid, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// 获得公众号客户端, 客户端有可能为 nil(被人为移除)
func (oc *OpenClient) GetClient(appid string) (*PublicClient, error) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	client, ok := oc.publicClients[appid]
	if !ok {
		app, err := oc.configs.AppStorage.GetAppInfo(appid)
		if err != nil {
			return nil, err
		} else {
			if app == nil {
				client = nil
			} else {
				opts := &PublicClientConfigs{
					Appid:          appid,
					Info:           app,
					Dispatcher:     oc.Dispatcher,
					MsgVerifyToken: oc.configs.MsgVerifyToken,
					TokenGetter: func() (token string, err error) {
						return oc.getAppAccessToken(appid)
					},
					MsgCrypt: oc.msgCrypt,
					Logger:   oc.configs.Logger,
				}
				client = NewPublicClient(opts)
			}
			oc.publicClients[appid] = client
		}
	}
	return client, nil
}

// 读取用户发送/触发的消息
func (oc *OpenClient) ListenMessage(appid string, w http.ResponseWriter, r *http.Request) {
	client, err := oc.GetClient(appid)
	if err != nil {
		oc.configs.Logger.Println(err)
	} else if client != nil {
		client.ListenMessage(w, r)
	}
}

// 获得公众号信息，如果公众号不存在则拉取并保存公众号信息
func (oc *OpenClient) GetAppInfo(appid string) (*open_platform.AuthorizerInfo, error) {
	appInfo, err := oc.configs.AppStorage.GetAppInfo(appid)
	if err != nil {
		return nil, err
	}
	if appInfo == nil {
		return oc.refreshAppInfo(appid)
	} else {
		return appInfo, nil
	}
}

// 创建开放平台帐号并绑定公众号/小程序
func (oc *OpenClient) CreateAndBindOpenApp(appid string) (openAppid string, err error) {
	accessToken, err := oc.getComponentAccessToken()
	if err != nil {
		return "", err
	}
	return open_platform.CreateAndBindOpenApp(accessToken, appid)
}

// 将公众号/小程序绑定到开放平台帐号下, 该 API 用于将一个尚未绑定开放平台帐号的公众号或小程序绑定至指定开放平台帐号上。二者须主体相同。
func (oc *OpenClient) BindOpenApp(appid, openAppid string) error {
	accessToken, err := oc.getComponentAccessToken()
	if err != nil {
		return err
	}
	return open_platform.BindOpenApp(accessToken, appid, openAppid)
}

// 获取公众号/小程序所绑定的开放平台帐号
func (oc *OpenClient) GetBindOpenApp(appid string) (openAppid string, err error) {
	accessToken, err := oc.getComponentAccessToken()
	if err != nil {
		return "", err
	}
	return open_platform.GetBindOpenApp(accessToken, appid)
}

// 将公众号/小程序从开放平台帐号下解绑
func (oc *OpenClient) UnbindOpenApp(appid, openAppid string) error {
	accessToken, err := oc.getComponentAccessToken()
	if err != nil {
		return err
	}
	return open_platform.UnbindOpenApp(accessToken, appid, openAppid)
}

// 迁移公众号, 获得已授权公众号信息以及 refresh token, 并删除多余的公众号(可能公众号已解除授权)
func (oc *OpenClient) MigrateApps() error {
	accessToken, err := oc.getComponentAccessToken()
	if err != nil {
		return err
	}
	// 包含所有 app 信息，一次性全部重置
	var appids []string
	var offset, limit = 0, 100

	for {
		list, err := open_platform.GetAuthorizerList(accessToken, oc.configs.Appid, offset, limit)
		if err != nil {
			return err
		}
		if len(list.List) == 0 {
			break
		}
		offset += limit
		for _, information := range list.List {
			token := &AppAccessToken{
				RefreshToken: information.RefreshToken,
			}
			// 保存 access token
			err = oc.configs.ComponentStorage.SaveAppAccessToken(information.AuthorizerAppid, token)
			if err != nil {
				return err
			}
			// 获取并保存授权方信息
			_, err := oc.refreshAppInfo(information.AuthorizerAppid)
			if err != nil {
				return err
			}
			appids = append(appids, information.AuthorizerAppid)
		}
	}
	return oc.configs.AppStorage.DelAppInfoNotIn(appids)
}

//// 公众号用户统计
//type AppUserStatistics struct {
//	Appid      string          `json:"appid"`
//	Nickname   string          `json:"nickname"`
//	Err        string          `json:"err"`
//	Statistics *UserStatistics `json:"statistics"`
//}
//
//// 多公众号统计数据
//type MultipleAppUserStatistics struct {
//	AppCount          int                  `json:"count"`               // 当前统计的公众号总量
//	CumulateUser      int                  `json:"cumulate_user"`       // 用户总量
//	NewUser           int                  `json:"new_user"`            // 新增的用户数量
//	CancelUser        int                  `json:"cancel_user"`         // 取消关注的用户数量，new_user减去cancel_user即为净增用户数量
//	AppUserStatistics []*AppUserStatistics `json:"app_user_statistics"` // 公众号统计列表
//}
//
//// 获得多公众号统计数据
//func (oc *OpenClient) GetUserStatistics(apps []*AppInfo, beginDate, endDate time.Time) *MultipleAppUserStatistics {
//	//apps, err := oc.configs.AppStorage.GetAppNicknames(appLimit, appOffset)
//	//if err != nil {
//	//	return nil, err
//	//}
//	maus := &MultipleAppUserStatistics{
//		AppCount: len(apps),
//	}
//	wg := sync.WaitGroup{}
//	for _, app := range apps {
//		aus := &AppUserStatistics{
//			Appid:    app.Appid,
//			Nickname: app.NickName,
//		}
//		wg.Add(1)
//		go func() {
//			client, err := oc.GetClient(aus.Appid)
//			if err != nil {
//				aus.Err = err.Error()
//			} else {
//				aus.Statistics, err = client.GetUserStatistics(beginDate, endDate)
//				if err != nil {
//					aus.Err = err.Error()
//				} else {
//					maus.CumulateUser += aus.Statistics.CumulateUser
//					maus.NewUser += aus.Statistics.NewUser
//					maus.CancelUser += aus.Statistics.CancelUser
//				}
//			}
//			wg.Done()
//		}()
//		maus.AppUserStatistics = append(maus.AppUserStatistics, aus)
//	}
//	wg.Wait()
//	return maus
//}
