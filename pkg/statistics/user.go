package statistics

import (
	"github.com/morgine/wechat_sdk/pkg"
	"time"
)

type Summary struct {
	RefDate    string `json:"ref_date"`    // 数据的日期
	UserSource int    `json:"user_source"` // 用户的渠道，数值代表的含义如下： 0代表其他合计 1代表公众号搜索 17代表名片分享 30代表扫描二维码 51代表支付后关注（在支付完成页） 57代表文章内账号名称 100微信广告 161他人转载 176 专辑页内账号名称
	NewUser    int    `json:"new_user"`    // 新增的用户数量
	CancelUser int    `json:"cancel_user"` // 取消关注的用户数量，new_user减去cancel_user即为净增用户数量
}

type Cumulate struct {
	RefDate      string `json:"ref_date"`
	CumulateUser int    `json:"cumulate_user"`
}

// 获取用户增减数据
func GetUserSummary(token string, beginDate, endDate time.Time) ([]*Summary, error) {
	URL := "https://api.weixin.qq.com/datacube/getusersummary?access_token=" + token
	data := map[string]string{
		"begin_date": beginDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}
	res := &struct {
		List []*Summary `json:"list"`
	}{}
	err := pkg.PostSchema(pkg.KindJson, URL, data, res)
	if err != nil {
		return nil, err
	} else {
		return res.List, nil
	}
}

// 获取累计用户数据
func GetUserCumulate(token string, beginDate, endDate time.Time) ([]*Cumulate, error) {
	URL := "https://api.weixin.qq.com/datacube/getusercumulate?access_token=" + token
	data := map[string]string{
		"begin_date": beginDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}
	res := &struct {
		List []*Cumulate `json:"list"`
	}{}
	err := pkg.PostSchema(pkg.KindJson, URL, data, res)
	if err != nil {
		return nil, err
	} else {
		return res.List, nil
	}
}
