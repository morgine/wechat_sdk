package src

import (
	"fmt"
	"github.com/morgine/wechat_sdk/pkg"
	"log"
	"net/http"
	"sync"
	"time"
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
	MsgVerifyToken   string                                          // 消息验证 token
	AesKey           string                                          // 消息加解密 key
	AesToken         string                                          // 消息加解密 token
	AppidGetter      func(r *http.Request) (appid string, err error) // 监听服务器消息时， 通过该方法获得 appid 参数
	ComponentStorage ComponentStorage                                // 开放平台存储器
	AppStorage       AppStorage                                      // 公众号信息存储器
	Logger           *log.Logger                                     // 错误日志收集器
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

// 监听通知消息
func (oc *OpenClient) ListenVerifyTicket(w http.ResponseWriter, r *http.Request) {
	notify, err := pkg.ListenComponentAuthorizationNotify(r, oc.msgCrypt)
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

func (oc *OpenClient) setNotify(notify *pkg.AuthorizationNotify) error {
	switch notify.InfoType {
	case pkg.EvtComponentVerifyTicket:
		return oc.configs.ComponentStorage.SaveVerifyTicket(notify.ComponentVerifyTicket)
	case pkg.EvtAuthorized, pkg.EvtUpdateauthorized:
		_, err := oc.refreshAppInfo(notify.AuthorizerAppid)
		if err != nil {
			return err
		}
	case pkg.EvtUnauthorized:
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
		accessToken, err := pkg.GetComponentAccessToken(oc.configs.Appid, oc.configs.Secret, verifyTicket)
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
	code, err := oc.configs.ComponentStorage.GetPreAuthCode()
	if err != nil {
		return "", err
	}
	now := Now().Unix()
	if code == nil || code.ExpiredAt < now {
		token, err := oc.getComponentAccessToken()
		if err != nil {
			return "", nil
		}
		pac, err := pkg.CreatePreAuthCode(oc.configs.Appid, token)
		if err != nil {
			return "", err
		}
		code = &ExpireData{
			Value:     pac.PreAuthCode,
			ExpiredAt: now + pac.ExpiresIn - (pac.ExpiresIn >> 3), // 过期时间提前 1/8
		}
		err = oc.configs.ComponentStorage.SavePreAuthCode(code)
		if err != nil {
			return "", err
		}
	}
	return code.Value, nil
}

// 获得授权地址
func (oc *OpenClient) ComponentLoginPage(redirect string) (string, error) {
	preAuthCode, err := oc.getPreAuthCode()
	if err != nil {
		return "", err
	}
	return pkg.ComponentLoginPage(&pkg.ComponentLoginPageOptions{
		ComponentAppid: oc.configs.Appid,
		PreAuthCode:    preAuthCode,
		RedirectUri:    redirect,
		AuthType:       "",
		BizAppid:       "",
	}), nil
}

// 获得授权信息, 用户授权/未授权都跳回该地址
func (oc *OpenClient) ListenLoginPage(request *http.Request) error {
	code, _ := pkg.ListenLoginPage(request)
	if code != "" {
		token, err := oc.getComponentAccessToken()
		if err != nil {
			return err
		}
		authInfo, err := pkg.GetAuthorizationInfo(oc.configs.Appid, code, token)
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
			authToken, err := pkg.RefreshAuthorizerToken(oc.configs.Appid, appid, appAccessToken.RefreshToken, componentToken)
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
func (oc *OpenClient) refreshAppInfo(appid string) (*AppInfo, error) {
	token, err := oc.getComponentAccessToken()
	if err != nil {
		return nil, err
	}
	authInfo, err := pkg.GetAuthorizerInfo(oc.configs.Appid, appid, token)
	if err != nil {
		return nil, err
	}
	info := authInfo.AuthorizerInfo
	appInfo := &AppInfo{
		Appid:           appid,
		NickName:        info.NickName,
		HeadImg:         info.HeadImg,
		ServiceTypeInfo: info.ServiceTypeInfo.ID,
		VerifyTypeInfo:  info.VerifyTypeInfo.ID,
		UserName:        info.UserName,
		PrincipalName:   info.PrincipalName,
		Alias:           info.Alias,
		QrcodeUrl:       info.QrcodeUrl,
		Idc:             info.Idc,
		Signature:       info.Signature,
	}
	err = oc.configs.AppStorage.SaveAppInfo(appid, appInfo)
	if err != nil {
		return nil, err
	}
	return appInfo, nil
}

// 获得公众号客户端
func (oc *OpenClient) GetClient(appid string) (*PublicClient, error) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	client := oc.publicClients[appid]
	if client == nil {
		opts := &PublicClientConfigs{
			AppGetter: func() (*AppInfo, error) {
				return oc.configs.AppStorage.GetAppInfo(appid)
			},
			Dispatcher:     oc.Dispatcher,
			MsgVerifyToken: oc.configs.MsgVerifyToken,
			TokenGetter: func() (token string, err error) {
				return oc.getAppAccessToken(appid)
			},
			MsgCrypt: oc.msgCrypt,
			Logger:   oc.configs.Logger,
		}
		var err error
		client, err = NewPublicClient(opts)
		if err != nil {
			return nil, err
		}
		oc.publicClients[appid] = client
	}
	return client, nil
}

// 读取用户发送/触发的消息
func (oc *OpenClient) ListenMessage(w http.ResponseWriter, r *http.Request) {
	appid, err := oc.configs.AppidGetter(r)
	if err != nil {
		oc.configs.Logger.Println(err)
	} else {
		client, err := oc.GetClient(appid)
		if err != nil {
			oc.configs.Logger.Println(err)
		} else {
			client.ListenMessage(w, r)
		}
	}
	return
}

// 获得公众号信息，如果公众号不存在则拉取并保存公众号信息
func (oc *OpenClient) GetAppInfo(appid string) (*AppInfo, error) {
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
		list, err := pkg.GetAuthorizerList(accessToken, oc.configs.Appid, offset, limit)
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
			info, err := oc.refreshAppInfo(information.AuthorizerAppid)
			if err != nil {
				return err
			}
			appids = append(appids, info.Appid)
		}
	}
	return oc.configs.AppStorage.DelAppInfoNotIn(appids)
}

// 公众号用户统计
type AppUserStatistics struct {
	Appid      string          `json:"appid"`
	Nickname   string          `json:"nickname"`
	Err        string          `json:"err"`
	Statistics *UserStatistics `json:"statistics"`
}

// 多公众号统计数据
type MultipleAppUserStatistics struct {
	AppCount          int                  `json:"count"`               // 当前统计的公众号总量
	CumulateUser      int                  `json:"cumulate_user"`       // 用户总量
	NewUser           int                  `json:"new_user"`            // 新增的用户数量
	CancelUser        int                  `json:"cancel_user"`         // 取消关注的用户数量，new_user减去cancel_user即为净增用户数量
	AppUserStatistics []*AppUserStatistics `json:"app_user_statistics"` // 公众号统计列表
}

// 获得多公众号统计数据
func (oc *OpenClient) UserStatistics(appLimit, appOffset int, beginDate, endDate time.Time) (*MultipleAppUserStatistics, error) {
	apps, err := oc.configs.AppStorage.GetAppNicknames(appLimit, appOffset)
	if err != nil {
		return nil, err
	}
	maus := &MultipleAppUserStatistics{
		AppCount: len(apps),
	}
	wg := sync.WaitGroup{}
	for _, app := range apps {
		aus := &AppUserStatistics{
			Appid:    app.Appid,
			Nickname: app.Nickname,
		}
		wg.Add(1)
		go func() {
			client, err := oc.GetClient(aus.Appid)
			if err != nil {
				aus.Err = err.Error()
			} else {
				aus.Statistics, err = client.GetUserStatistics(beginDate, endDate)
				if err != nil {
					aus.Err = err.Error()
				} else {
					maus.CumulateUser += aus.Statistics.CumulateUser
					maus.NewUser += aus.Statistics.NewUser
					maus.CancelUser += aus.Statistics.CancelUser
				}
			}
			wg.Done()
		}()
		maus.AppUserStatistics = append(maus.AppUserStatistics, aus)
	}
	wg.Wait()
	return maus, nil
}
