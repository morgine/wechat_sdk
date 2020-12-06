package message

import (
	"encoding/xml"
	"errors"
	"github.com/morgine/wechat/pkg"
	"io/ioutil"
	"net/http"
)

var ErrIncorrectSignature = errors.New("签名错误")

type EventType string

const (
	// 直接关注或扫描公众号二维码关注
	EvtUserSubscribe EventType = "subscribe"

	// 已关注用户扫描公众号二维码
	EvtUserScan EventType = "SCAN"

	// 用户取消关注
	EvtUserUnsubscribe EventType = "unsubscribe"

	// 上报地理位置事件
	// 用户同意上报地理位置后，每次进入公众号会话时，都会在进入时上报地理位置，或在进入会话后每5秒上报一次地理位置，
	// 公众号可以在公众平台网站中修改以上设置。上报地理位置时，微信会将上报地理位置事件推送到开发者填写的URL。
	EvtUserLocation EventType = "LOCATION"

	// 自定义菜单事件
	EvtUserClick EventType = "CLICK" // 点击菜单拉取消息时的事件推送
	EvtUserView  EventType = "VIEW"  // 点击菜单跳转链接时的事件推送

	// 模板消息发送之后微信服务器会推送一个事件消息
	EvtTemplateMsgResult EventType = "TEMPLATESENDJOBFINISH"

	// 模板消息发送之后微信服务器会推送一个事件消息
	EvtGroupMsgResult EventType = "MASSSENDJOBFINISH"
)

// 微信服务器发出来的消息
type ServerMsgType string

const (
	// 事件类型消息
	ServerMsgTypeEvent ServerMsgType = "event"
	// 用户发送文本消息
	ServerMsgTypeText ServerMsgType = "text"
	// 用户发送图片
	ServerMsgTypeImage ServerMsgType = "image"
	// 用户发送语音
	ServerMsgTypeVoice ServerMsgType = "voice"
	// 用户发送视频
	ServerMsgTypeVideo ServerMsgType = "video"
	// 用户发送短视频
	ServerMsgTypeShortVideo ServerMsgType = "shortvideo"
	// 用户发送位置信息
	ServerMsgTypeLocation ServerMsgType = "location"
	// 用户发送连接
	ServerMsgTypeLink ServerMsgType = "link"
)

// TODO: 1.完善自定义菜单事件推送, see: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141016
// TODO: 2.完善对用户消息的解析, see: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140453
// 微信服务器推送给本地服务器的消息
type ServerMessage struct {

	// 微信原始ID
	ToUserName string

	// openid
	FromUserName string

	// 消息创建时间（整型）
	CreateTime int64

	// 消息类型，event/text/image/voice/video/shortvideo/location/link
	MsgType ServerMsgType

	// 消息id，64位整型
	MsgId int64
}

type ServerMessageData []byte

type EventMessage struct {

	// 事件类型
	Event EventType

	// 事件KEY值.
	// 如果用户扫描公众号二维码且用户未关注公众号时返回 qrscene_为前缀，后面为二维码的参数值
	// 如果用户扫描公众号二维码且用户已关注公众号时返回二维码scene_id, 是一个32位无符号整数
	// 如果是公众号菜单 CLICK 事件, 则返回自定义菜单所设置的 KEY 值
	EventKey string
}

// 解析事件消息
func (smd ServerMessageData) MarshalEvent() (*EventMessage, error) {
	msg := &EventMessage{}
	err := xml.Unmarshal(smd, msg)
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}

// MsgType: "text"
type TextMessage struct {
	// 文本消息内容
	Content string

	// 消息ID
	MsgId int64

	// 点击的菜单ID
	// see: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140547
	BizMsgMenuID string `xml:"bizmsgmenuid" desc:"点击客服消息菜单中的ID"`
}

// 解析文本消息
func (smd ServerMessageData) MarshalTextMessage() (msg *TextMessage, err error) {
	msg = &TextMessage{}
	err = xml.Unmarshal(smd, msg)
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}

// MsgType: "event", Event: "scan" or "subscribe"
// 用户扫描带参数的公众号二维码
// see: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140454
type ScanQRCode struct {

	// 用户扫描关注二维码的ticket，可用来换取二维码图片
	Ticket string
}

// 解析二维码参数，需要满足事件类型为 scan/subscribe
func (smd ServerMessageData) MarshalScanQRCode() (ticket *ScanQRCode, err error) {
	ticket = &ScanQRCode{}
	err = xml.Unmarshal(smd, ticket)
	if err != nil {
		return nil, err
	} else {
		return ticket, nil
	}
}

// MsgType: "event", Event: "LOCATION"
// 用户同意上报地理位置后，每次进入公众号会话时，都会在进入时上报地理位置，或在进入会话后每
// 5秒上报一次地理位置，公众号可以在公众平台网站中修改以上设置。上报地理位置时，微信会将上
// 报地理位置事件推送到开发者填写的URL。
type UserLocation struct {
	Latitude  float32 `desc:"地理位置纬度"`
	Longitude float32 `desc:"地理位置经度"`
	Precision float32 `desc:"地理位置精度"`
}

func (smd ServerMessageData) MarshalUserLocation() (location *UserLocation, err error) {
	location = &UserLocation{}
	err = xml.Unmarshal(smd, location)
	if err != nil {
		return nil, err
	} else {
		return location, nil
	}
}

// MsgType: "event", Event: "TEMPLATESENDJOBFINISH""
// 模板消息发送结果
type TemplateMsgResult struct {
	// 模板消息ID
	MsgID int64 `desc:"64位整型"`

	// [success]-[发送成功] [failed:user block]-[用户屏蔽消息] [failed: system failed]-[系统错误]
	Status string
}

func (smd ServerMessageData) MarshalTemplateMsgResult() (result *TemplateMsgResult, err error) {
	result = &TemplateMsgResult{}
	err = xml.Unmarshal(smd, result)
	if err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

// MsgType: "event", Event: "MASSSENDJOBFINISH"
// 群发消息结果
type GroupMsgResult struct {
	// 群发消息ID, 64位整型
	MsgID int64

	// 群发结果:
	// 为“send success”或“send fail”或“err(num)”。但send success时，也有可能因用户
	// 拒收公众号的消息、系统错误等原因造成少量用户接收失败。err(num)是审核失败的具体原因，可能的情
	// 况如下：
	// err(10001), 涉嫌广告
	// err(20001), 涉嫌政治
	// err(20004), 涉嫌社会
	// err(20002), 涉嫌色情
	// err(20006), 涉嫌违法犯罪
	// err(20008), 涉嫌欺诈
	// err(20013), 涉嫌版权
	// err(22000), 涉嫌互推(互相宣传)
	// err(21000), 涉嫌其他
	// err(30001), 原创校验出现系统错误且用户选择了被判为转载就不群发
	// err(30002), 原创校验被判定为不能群发
	// err(30003), 原创校验被判定为转载文且用户选择了被判为转载就不群发
	Status string

	// tag_id下粉丝数；或者openid_list中的粉丝数
	TotalCount int

	// 过滤（过滤是指特定地区、性别的过滤、用户设置拒收的过滤，用户接收已超4条的过滤）后，
	// 准备发送的粉丝数，原则上，FilterCount = SentCount + ErrorCount
	FilterCount int

	// 发送成功的粉丝数
	SentCount int

	// 发送失败的粉丝数
	ErrorCount int
	CopyrightCheckResult
}

func (smd ServerMessageData) MarshalGroupMsgResult() (result *GroupMsgResult, err error) {
	result = &GroupMsgResult{}
	err = xml.Unmarshal(smd, result)
	if err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

// Response 用于被动回复消息, 当用户发送文本、图片、视频、图文、地理位置这五种消息时，开发者只能回复1条
// 图文消息；其余场景最多可回复8条图文消息, 多余的消息将被忽略
func Response(serverMsg *ServerMessage, resMsg *ResponseMessage, writer http.ResponseWriter, encrypt *pkg.WXBizMsgCrypt) error {
	resMsg.FromUserName = pkg.Cdata{Value: serverMsg.ToUserName}
	resMsg.ToUserName = pkg.Cdata{Value: serverMsg.FromUserName}
	if resMsg.Articles != nil {
		resMsg.ArticleCount = len(*resMsg.Articles)
	}
	data, err := xml.Marshal(resMsg)
	if err != nil {
		return err
	}
	if encrypt != nil {
		msg, err := encrypt.EncryptMsg(data, resMsg.CreateTime, "")
		if err != nil {
			return err
		} else {
			data, err = xml.Marshal(msg)
			if err != nil {
				return err
			}
		}
	}
	writer.Header().Set("Content-Type", "application/xml;charset=UTF-8")
	_, err = writer.Write(data)
	return err
}

// 验证请求是否是来自微信服务器, 如果 echoStr 不为空, 则响应 echoStr 内容.
// token 为消息校验 token, 从微信后台获取.
func CheckSignature(req *http.Request, token string) (echoStr string, err error) {
	q := req.URL.Query()
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	signature := q.Get("signature")
	if signature != pkg.SignParams(token, timestamp, nonce) {
		return "", ErrIncorrectSignature
	}
	return q.Get("echostr"), nil
}

// 读取用户发送/触发的消息, 如果 decrypter 不为 nil, 则通过 decrypter 解密, 否则按明文方式解析消息.
// 读取消息之前应当使用 CheckSignature 验证签名
func ReadServerMessage(req *http.Request, decrypter *pkg.WXBizMsgCrypt) (smsg *ServerMessage, smd ServerMessageData, err error) {
	var data []byte
	if decrypter != nil {
		data, err = decrypter.DecryptRequest(req)
		if err != nil {
			return nil, nil, err
		}
	} else {
		data, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, nil, err
		}
		defer req.Body.Close()
	}
	smsg = &ServerMessage{}
	err = xml.Unmarshal(data, smsg)
	if err != nil {
		return nil, nil, err
	} else {
		return smsg, data, nil
	}
}

type CopyrightCheckResult struct {
	// ResultList item 个数
	Count int

	ResultList *ResultList `xml:"ResultList>item,omitempty"`

	// 1-未被判为转载，可以群发，2-被判为转载，可以群发，3-被判为转载，不能群发
	CheckState int
}

type ResultList []ArticleResult

type ArticleResult struct {

	// 群发文章的序号，从1开始
	ArticleIdx int

	// 用户声明文章的状态
	UserDeclareState int

	// 系统校验的状态
	AuditState int

	// 	相似原创文的url
	OriginalArticleUrl string

	// 相似原创文的类型
	OriginalArticleType int

	// 是否能转载
	CanReprint int

	// 是否需要替换成原创文内容
	NeedReplaceContent int

	// 是否需要注明转载来源
	NeedShowReprintSource int
}
