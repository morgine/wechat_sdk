package src

type AppInfo struct {
	// 公众号 appid
	Appid string

	// 授权方昵称
	NickName string `json:"nick_name"`

	// 授权方头像
	HeadImg string `json:"head_img"`

	// 授权方公众号类型，0代表订阅号，1代表由历史老帐号升级后的订阅号，2代表服务号
	ServiceTypeInfo int `json:"service_type_info"`

	// 授权方认证类型，-1代表未认证，0代表微信认证，1代表新浪微博认证，2代表腾讯微博认证，
	// 3代表已资质认证通过但还未通过名称认证，4代表已资质认证通过、还未通过名称认证，但通
	// 过了新浪微博认证，5代表已资质认证通过、还未通过名称认证，但通过了腾讯微博认证
	VerifyTypeInfo int `json:"verify_type_info"`

	// 授权方公众号的原始ID
	UserName string `json:"user_name"`

	// 公众号的主体名称
	PrincipalName string `json:"principal_name"`

	// 授权方公众号所设置的微信号，可能为空
	Alias string `json:"alias"`

	// 用以了解以下功能的开通状况（0代表未开通，1代表已开通）： open_store:是否开通微信门店
	// 功能 open_scan:是否开通微信扫商品功能 open_pay:是否开通微信支付功能 open_card:是否
	// 开通微信卡券功能 open_shake:是否开通微信摇一摇功能
	//BusinessInfo *Business `json:"business_info"`

	// 二维码图片的URL，开发者最好自行也进行保存
	QrcodeUrl string `json:"qrcode_url"`

	Idc int `json:"idc"` // 文档无说明

	// APP 简介
	Signature string `json:"signature"`
}

type AppGetter func() (*AppInfo, error)
