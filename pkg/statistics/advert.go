package statistics

import (
	"fmt"
	"github.com/morgine/wechat_sdk/pkg"
	"net/url"
	"strconv"
	"time"
)

// 广告数据分析接口说明
// 向所有成为流量主的公众号、小程序、小游戏开发者开放数据接口。通过数据接口，开发者可以获取与公众平台官网统计模块类似但更灵活的数据，还可根据需要进行高级处理。
//
// 请注意：
//
// 接口侧数据库中仅存储了2016年1月1日之后的数据，将无法查询到此前的数据，即使查到，也是不可信的脏数据；
//
// 建议开发者在调用接口获取数据后，将数据保存在自身数据库中，以最大化访问的效率，也降低微信侧接口调用的不必要损耗；
//
// 由于数据量较大, 所有接口采取分页获取的方式, 每页最大获取量为90。（eg：total_num 为100，则当page = 1，page_size = 10，则返回前10条；page = 1，page_size = 20，则返回前20条；page = 2，page_size = 10，则返回第11条到第20条）
//
// 广告位枚举值变更说明
//
// 由于多个接口都使用了广告位参数，为保证体验的一致性和参数的可读性，我们做了一些变更，所有接口均支持以 广告位类型名称（ad_slot） 传递参数，回包时新增这个名称来代表相关含义。此前的参数 slot_id 也可继续使用并回传。

// 广告位类型名称（ad_slot）	广告位类型
const (
	SlotIdBizBottom         AdSlot = "SLOT_ID_BIZ_BOTTOM"         // 公众号底部广告
	SlotIdBizMidContext     AdSlot = "SLOT_ID_BIZ_MID_CONTEXT"    // 公众号文中广告
	SlotIdBizVideoEnd       AdSlot = "SLOT_ID_BIZ_VIDEO_END"      // 公众号视频后贴
	SlotIdBizSponsor        AdSlot = "SLOT_ID_BIZ_SPONSOR"        // 公众号互选广告
	SlotIdBizCps            AdSlot = "SLOT_ID_BIZ_CPS"            // 公众号返佣商品
	SlotIdWeappBanner       AdSlot = "SLOT_ID_WEAPP_BANNER"       // 小程序banner
	SlotIdWeappRewardVideo  AdSlot = "SLOT_ID_WEAPP_REWARD_VIDEO" // 小程序激励视频
	SlotIdWeappInterstitial AdSlot = "SLOT_ID_WEAPP_INTERSTITIAL" // 小程序插屏广告
	SlotIdWeappVideoFeeds   AdSlot = "SLOT_ID_WEAPP_VIDEO_FEEDS"  // 小程序视频广告
	SlotIdWeappVideoBegin   AdSlot = "SLOT_ID_WEAPP_VIDEO_BEGIN"  // 小程序视频前贴
	SlotIdWeappBox          AdSlot = "SLOT_ID_WEAPP_BOX"          // 小程序格子广告
)

const MaxPublisherPageSize = 90

// 广告位类型
type AdSlot string

// 广告统计公共参数
type PublisherCommonOptions struct {
	Page      int       // 返回第几页数据
	PageSize  int       // 当页返回数据条数，最大90条
	StartDate time.Time // 获取数据的开始时间 yyyy-mm-dd
	EndDate   time.Time // 获取数据的结束时间 yyyy-mm-dd
}

func (po PublisherCommonOptions) toUrl(action, accessToken string, slot AdSlot) string {
	if po.PageSize > MaxPublisherPageSize {
		po.PageSize = MaxPublisherPageSize
	}
	vs := url.Values{
		"action":       []string{action},
		"access_token": []string{accessToken},
		"page":         []string{strconv.Itoa(po.Page)},
		"page_size":    []string{strconv.Itoa(po.PageSize)},
		"start_date":   []string{po.StartDate.Format("2006-01-02")},
		"end_date":     []string{po.EndDate.Format("2006-01-02")},
	}
	if slot != "" {
		vs.Set("ad_slot", string(slot))
	}
	return "https://api.weixin.qq.com/publisher/stat?" + vs.Encode()
}

type BaseResp struct {
	ErrMsg string `json:"err_msg"` // 返回错误信息
	Ret    int    `json:"ret"`     // 错误码
}

func (br BaseResp) IsError() (ok bool, err error) {
	if br.Ret == 0 {
		return false, nil
	} else {
		switch br.Ret {
		case 45009:
			return true, fmt.Errorf("ret: %d, err msg: 请求过于频繁, 请稍后尝试", br.Ret)
		case 45010:
			return true, fmt.Errorf("ret: %d, err msg: 无效的接口名", br.Ret)
		case 1701:
			return true, fmt.Errorf("ret: %d, err msg: 参数错误", br.Ret)
		case 2009:
			return true, fmt.Errorf("ret: %d, err msg: 无效的流量主", br.Ret)
		default:
			return true, fmt.Errorf("ret: %d, err msg: 无效的流量主", br.ErrMsg)
		}
	}
}

type PublisherAdPosGeneralResponse struct {
	BaseResp BaseResp `json:"base_resp"`
	List     []struct {
		SlotID        int64   `json:"slot_id"`        // 广告位类型id
		AdSlot        string  `json:"ad_slot"`        // 广告位类型名称
		Date          string  `json:"date"`           // 日期
		ReqSuccCount  int     `json:"req_succ_count"` // 拉取量
		ExposureCount int     `json:"exposure_count"` // 曝光量
		ExposureRate  float64 `json:"exposure_rate"`  // 曝光率
		ClickCount    int     `json:"click_count"`    // 点击量
		ClickRate     float64 `json:"click_rate"`     // 点击率
		Income        int     `json:"income"`         // 收入(分)
		Ecpm          float64 `json:"ecpm"`           // 广告千次曝光收益(分)
	} `json:"list"`
	Summary struct {
		ReqSuccCount  int     `json:"req_succ_count"` // 总拉取量
		ExposureCount int     `json:"exposure_count"` // 总曝光量
		ExposureRate  float64 `json:"exposure_rate"`  // 总曝光率
		ClickCount    int     `json:"click_count"`    // 总点击量
		ClickRate     float64 `json:"click_rate"`     // 总点击率
		Income        int     `json:"income"`         // 总收入(分)
		Ecpm          float64 `json:"ecpm"`           // 广告千次曝光收益(分)
	} `json:"summary"`
	TotalNum int `json:"total_num"` // list 返回总条数
}

// 获取公众号分广告位数据, 最大时间跨度: 90天, slot 是广告位类型，为可选参数
func GetPublisherAdPosGeneral(accessToken string, slot AdSlot, opts PublisherCommonOptions) (*PublisherAdPosGeneralResponse, error) {
	uri := opts.toUrl("publisher_adpos_general", accessToken, slot)
	rsp := &PublisherAdPosGeneralResponse{}
	err := pkg.GetJson(uri, rsp)
	if err != nil {
		return nil, err
	} else {
		if ok, err := rsp.BaseResp.IsError(); ok {
			return nil, err
		}
		return rsp, nil
	}
}

type PublisherCpsGeneralResponse struct {
	BaseResp BaseResp `json:"base_resp"`
	List     []struct {
		Date            string  `json:"date"`             // 日期
		ExposureCount   int     `json:"exposure_count"`   // 曝光量
		ClickCount      int     `json:"click_count"`      // 点击量
		ClickRate       float64 `json:"click_rate"`       // 点击率
		OrderCount      int     `json:"order_count"`      // 订单量
		OrderRate       float64 `json:"order_rate"`       // 下单率
		TotalFee        int     `json:"total_fee"`        // 订单金额(分)
		TotalCommission int     `json:"total_commission"` // 预估收入(分)
	} `json:"list"`
	Summary struct {
		ExposureCount   int     `json:"exposure_count"`   // 总曝光量
		ClickCount      int     `json:"click_count"`      // 总点击量
		ClickRate       float64 `json:"click_rate"`       // 总点击率
		OrderCount      int     `json:"order_count"`      // 总下单量
		OrderRate       float64 `json:"order_rate"`       // 总下单率
		TotalFee        int     `json:"total_fee"`        // 订单总金额(分)
		TotalCommission int     `json:"total_commission"` // 总预估收入(分)
	} `json:"summary"`
	TotalNum int `json:"total_num"` // list 返回总条数
}

// 获取公众号返佣商品数据, 最大时间跨度: 60天
func GetPublisherCpsGeneral(accessToken string, opts PublisherCommonOptions) (*PublisherCpsGeneralResponse, error) {
	uri := opts.toUrl("publisher_cps_general", accessToken, "")
	rsp := &PublisherCpsGeneralResponse{}
	err := pkg.GetJson(uri, rsp)
	if err != nil {
		return nil, err
	} else {
		if ok, err := rsp.BaseResp.IsError(); ok {
			return nil, err
		}
		return rsp, nil
	}
}

type PublisherSettlementResponse struct {
	BaseResp          BaseResp `json:"base_resp"`
	Body              string   `json:"body"`                // 主体名称
	RevenueAll        int      `json:"revenue_all"`         // 累计收入
	PenaltyAll        int      `json:"penalty_all"`         // 扣除金额
	SettledRevenueAll int      `json:"settled_revenue_all"` // 已结算金额
	SettlementList    []struct {
		Date           string `json:"date"`            // 数据更新时间
		Zone           string `json:"zone"`            // 日期区间
		Month          string `json:"month"`           // 收入月份
		Order          int    `json:"order"`           // 1 = 上半月，2 = 下半月
		SettStatus     int    `json:"sett_status"`     // 1 = 结算中；2、3 = 已结算；4 = 付款中；5 = 已付款
		SettledRevenue int    `json:"settled_revenue"` // 区间内结算收入
		SettNo         string `json:"sett_no"`         // 结算单编号
		MailSendCnt    int    `json:"mail_send_cnt"`   // 申请补发结算单次数
		SlotRevenue    []struct {
			SlotID             string `json:"slot_id"`      // 产生收入的广告位
			SlotSettledRevenue int    `json:"slot_revenue"` // 该广告位结算金额
		} `json:"slot_revenue"`
	} `json:"settlement_list"`
	TotalNum int `json:"total_num"` // 请求返回总条数
}

// 获取公众号结算收入数据及结算主体信息, 最大时间跨度: 无
func GetPublisherSettlement(accessToken string, opts PublisherCommonOptions) (*PublisherSettlementResponse, error) {
	uri := opts.toUrl("publisher_settlement", accessToken, "")
	rsp := &PublisherSettlementResponse{}
	err := pkg.GetJson(uri, rsp)
	if err != nil {
		return nil, err
	} else {
		if ok, err := rsp.BaseResp.IsError(); ok {
			return nil, err
		}
		return rsp, nil
	}
}
