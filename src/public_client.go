package src

import (
	"github.com/morgine/wechat_sdk/pkg"
	"github.com/morgine/wechat_sdk/pkg/material"
	"github.com/morgine/wechat_sdk/pkg/message"
	"github.com/morgine/wechat_sdk/pkg/statistics"
	"github.com/morgine/wechat_sdk/pkg/users"
	"log"
	"net/http"
	"os"
	"time"
)

type PublicClient struct {
	configs *PublicClientConfigs
}

type PublicClientConfigs struct {
	AppGetter      AppGetter          // 公众号信息
	Dispatcher     *Dispatcher        // 事件处理器
	MsgVerifyToken string             // 消息验证
	TokenGetter    AccessTokenGetter  // token  提供器
	MsgCrypt       *pkg.WXBizMsgCrypt // 消息加密/解密器
	Logger         *log.Logger        // 错误日志收集器
}

// 获得 appid
func (pc *PublicClient) GetAppInfo() (*AppInfo, error) {
	return pc.configs.AppGetter()
}

func NewPublicClient(opts *PublicClientConfigs) (*PublicClient, error) {
	if opts.Logger == nil {
		app, err := opts.AppGetter()
		if err != nil {
			return nil, err
		}
		opts.Logger = log.New(os.Stderr, app.Appid, log.LstdFlags|log.Llongfile)
	}
	return &PublicClient{
		configs: opts,
	}, nil
}

// 读取用户发送/触发的消息, 如果 decrypter 不为 nil, 则通过 decrypter 解密, 否则按明文方式解析消息.
func (pc *PublicClient) ListenMessage(w http.ResponseWriter, r *http.Request) {
	echoStr, err := message.CheckSignature(r, pc.configs.MsgVerifyToken)
	if err != nil {
		pc.configs.Logger.Println(err)
	}
	if echoStr != "" {
		_, _ = w.Write([]byte(echoStr))
	} else {
		msg, msgData, err := message.ReadServerMessage(r, pc.configs.MsgCrypt)
		if err != nil {
			pc.configs.Logger.Println(err)
		} else {
			writer := &responseWriter{
				msgCrypt: pc.configs.MsgCrypt,
				history:  nil,
				w:        w,
				msg:      msg,
			}
			err = pc.configs.Dispatcher.trigger(
				msg,
				msgData,
				pc.configs.AppGetter,
				func() CustomerMsgResponser {
					sender := newCustomerMsgSender(pc.configs.TokenGetter, pc.configs.Logger)
					return newCustomerMsgResponser(msg.FromUserName, sender)
				},
				writer,
			)
			if err != nil {
				pc.configs.Logger.Println(err)
			}
			if writer.history == nil {
				_, _ = w.Write([]byte(""))
			}
		}
	}
	return
}

// 上传永久素材
func (pc *PublicClient) UploadMaterial(mediaType material.MediaType, data []byte, filename string, videoDesc *material.VideoDescription) (res *material.UploadedMedia, err error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		return material.UploadMaterial(mediaType, data, filename, token, videoDesc)
	}
}

// 上传临时素材，3 天有效
func (pc *PublicClient) UploadTempMaterial(mediaType material.MediaType, data []byte, filename string) (res *material.TempMedia, err error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		return material.UploadTempMaterial(mediaType, data, filename, token)
	}
}

// 创建公众号标签
func (pc *PublicClient) CreateAppUserTag(tagName string) (*users.Tag, error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	}
	return users.CreateTag(tagName, token)
}

// 获得公众号标签
func (pc *PublicClient) GetAppUserTags() ([]*users.Tag, error) {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	}
	return users.GetTags(accessToken)
}

// 更新公众号标签
func (pc *PublicClient) UpdateAppUserTag(tag *users.Tag) error {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return err
	}
	return users.UpdateTag(accessToken, tag)
}

// 删除公众号标签
// 错误码    说明
// -1	   系统繁忙
// 45058   不能修改0/1/2这三个系统默认保留的标签
// 45057   该标签下粉丝数超过10w，不允许直接删除
func (pc *PublicClient) DeleteAppUserTag(tagID int) error {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return err
	}
	return users.DeleteTag(accessToken, tagID)
}

// 获得对应标签下的用户列表
func (pc *PublicClient) GetAppTagUsers(tagID int, nextOpenid string) (*users.Users, error) {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	}
	return users.GetTagUsers(accessToken, tagID, nextOpenid)
}

// 批量为用户打标签
func (pc *PublicClient) BatchTagging(tagID int, openids []string) error {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return err
	}
	return users.BatchTagging(accessToken, tagID, openids)
}

// 批量为用户取消标签
func (pc *PublicClient) BatchUntagging(tagID int, openids []string) error {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return err
	}
	return users.BatchUntagging(accessToken, tagID, openids)
}

// 获取用户身上的标签列表
func (pc *PublicClient) GetUserTags(openid string) (ids []int, err error) {
	accessToken, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	}
	return users.GetUserTags(accessToken, openid)
}

// 获取用户增减数据
func (pc *PublicClient) GetUserSummary(beginDate, endDate time.Time) ([]*statistics.Summary, error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		dateTimeRanges := splitDateTimeRange(beginDate, endDate, oneWeekTime, fiveWeekTime)
		var summaries []*statistics.Summary
		for _, dtr := range dateTimeRanges {
			rangeSummaries, err := statistics.GetUserSummary(token, dtr.BeginDate, dtr.EndDate)
			if err != nil {
				return nil, err
			} else {
				summaries = append(summaries, rangeSummaries...)
			}
		}
		return summaries, nil
	}
}

// 获取累计用户数据
func (pc *PublicClient) GetUserCumulate(beginDate, endDate time.Time) ([]*statistics.Cumulate, error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		dateTimeRanges := splitDateTimeRange(beginDate, endDate, oneWeekTime, fiveWeekTime)
		var cumulates []*statistics.Cumulate
		for _, dtr := range dateTimeRanges {
			rangeCumulates, err := statistics.GetUserCumulate(token, dtr.BeginDate, dtr.EndDate)
			if err != nil {
				return nil, err
			} else {
				cumulates = append(cumulates, rangeCumulates...)
			}
		}
		return cumulates, nil
	}
}

type dateTimeRange struct {
	BeginDate time.Time
	EndDate   time.Time
}

var (
	oneDateTime  = 24 * time.Hour
	oneWeekTime  = oneDateTime * 7
	fiveWeekTime = oneWeekTime * 5
)

// 将一个大的时间区间划分为多个小的时间区间，起止时间不分先后，将被自动判断，默认按一周时间分割
// 结束时间最晚只能是当前时间的前一天，起始时间最晚只能是结束时间的前一天，超出范围将被设置为默认值
// 最大时间跨度不能超过 maxRange，超出将重置 endDate 到最大时间范围内(防止误输入将区间设置得过大)
func splitDateTimeRange(beginDate, endDate time.Time, splitRange, maxRange time.Duration) []*dateTimeRange {
	// 如果起始日期大于结束日期，则将二者日期交换
	if beginDate.After(endDate) {
		beginDate, endDate = endDate, beginDate
	}
	// 结束日期最大值只能是当前日期的前一天
	now := Now()
	y, m, d := now.Date()
	yesterday := time.Date(y, m, d-1, 0, 0, 0, 0, now.Location())
	if endDate.After(yesterday) {
		endDate = yesterday
	}
	// 起始日期和结束日期必须有一天的时间跨度
	if endDate.Sub(beginDate) < oneDateTime {
		beginDate = endDate.Add(-oneDateTime)
	}
	// 如果超出最大范围则将结束时间设置在最大范围之内
	if endDate.Sub(beginDate) > maxRange {
		endDate = beginDate.Add(maxRange)
	}
	var dtrs []*dateTimeRange
	for {
		if r := endDate.Sub(beginDate); r > splitRange {
			dtr := &dateTimeRange{
				BeginDate: beginDate,
				EndDate:   beginDate.Add(splitRange),
			}
			dtrs = append(dtrs, dtr)
			beginDate = dtr.EndDate
		} else if r > 0 {
			dtr := &dateTimeRange{
				BeginDate: beginDate,
				EndDate:   beginDate.Add(r),
			}
			dtrs = append(dtrs, dtr)
			beginDate = dtr.EndDate
		} else {
			break
		}
	}
	return dtrs
}

type UserStatistics struct {
	CumulateUser int         `json:"cumulate_user"` // 用户总量
	NewUser      int         `json:"new_user"`      // 新增的用户数量
	CancelUser   int         `json:"cancel_user"`   // 取消关注的用户数量，new_user减去cancel_user即为净增用户数量
	Cumulates    []*Cumulate `json:"cumulates"`     // 累计数据(同一天的数据，不分用户渠道)
}

type Cumulate struct {
	RefDate      string                `json:"ref_date"`      // 日期
	CumulateUser int                   `json:"cumulate_user"` // 用户总量
	NewUser      int                   `json:"new_user"`      // 新增的用户数量
	CancelUser   int                   `json:"cancel_user"`   // 取消关注的用户数量，new_user减去cancel_user即为净增用户数量
	Summaries    []*statistics.Summary `json:"summaries"`     // 详细增减数据(该数据分用户渠道，同一天可能有多个不同渠道的数据)
}

// 用户统计，包含用户增减数据及累计用户数据
func (pc *PublicClient) GetUserStatistics(beginDate, endDate time.Time) (*UserStatistics, error) {
	us := &UserStatistics{
		Cumulates: nil,
	}
	summaries, err := pc.GetUserSummary(beginDate, endDate)
	if err != nil {
		return nil, err
	}
	cumulates, err := pc.GetUserCumulate(beginDate, endDate)
	if err != nil {
		return nil, err
	}
	us.Cumulates = make([]*Cumulate, len(cumulates))
	var maxDateCumulates *Cumulate
	for idx, c := range cumulates {
		cumulate := &Cumulate{
			RefDate:      c.RefDate,
			CumulateUser: c.CumulateUser,
			NewUser:      0,
			CancelUser:   0,
		}
		for _, summary := range summaries {
			if summary.RefDate == cumulate.RefDate {
				cumulate.NewUser += summary.NewUser
				cumulate.CancelUser += summary.CancelUser
				cumulate.Summaries = append(cumulate.Summaries, summary)
			}
		}
		us.Cumulates[idx] = cumulate
		us.CancelUser += cumulate.CancelUser
		us.NewUser += cumulate.NewUser
		if maxDateCumulates == nil || maxDateCumulates.RefDate < cumulate.RefDate {
			maxDateCumulates = cumulate
		}
	}
	if maxDateCumulates != nil {
		us.CumulateUser = maxDateCumulates.CumulateUser
	}
	return us, nil
}

// 获取公众号分广告位数据, 最大时间跨度: 90天, slot 是广告位类型，为可选参数
func (pc *PublicClient) GetPublisherAdPosGeneral(slot statistics.AdSlot, opts statistics.PublisherCommonOptions) (*statistics.PublisherAdPosGeneralResponse, error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		return statistics.GetPublisherAdPosGeneral(token, slot, opts)
	}
}

// 获取公众号返佣商品数据, 最大时间跨度: 60天
func (pc *PublicClient) GetPublisherCpsGeneral(opts statistics.PublisherCommonOptions) (*statistics.PublisherCpsGeneralResponse, error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		return statistics.GetPublisherCpsGeneral(token, opts)
	}
}

// 获取公众号结算收入数据及结算主体信息, 最大时间跨度: 无
func (pc *PublicClient) GetPublisherSettlement(opts statistics.PublisherCommonOptions) (*statistics.PublisherSettlementResponse, error) {
	token, err := pc.configs.TokenGetter()
	if err != nil {
		return nil, err
	} else {
		return statistics.GetPublisherSettlement(token, opts)
	}
}
