package message

import (
	"bytes"
	"encoding/json"
	"github.com/morgine/wechat/pkg"
	"io/ioutil"
	"net/http"
)

const (
	ErrGroupMsgCodeAlreadySent          = 45065
	ErrGroupMsgCodeSendTooFast          = 45066
	ErrGroupMsgCodeClientMsgIDIsTooLong = 45067
)

var errCodeTexts = map[int]string{
	ErrGroupMsgCodeAlreadySent:          "相同 clientmsgid 已存在群发记录，返回数据中带有已存在的群发任务的 msgid",
	ErrGroupMsgCodeSendTooFast:          "相同 clientmsgid 重试速度过快，请间隔1分钟重试",
	ErrGroupMsgCodeClientMsgIDIsTooLong: "clientmsgid 长度超过限制",
}

// 群发的消息类型
type GroupMessageType string

const (
	GroupMsgTypeMpNews GroupMessageType = "mpnews"
	GroupMsgTypeText   GroupMessageType = "text"
	GroupMsgTypeVoice  GroupMessageType = "voice"
	GroupMsgTypeMusic  GroupMessageType = "music"
	GroupMsgTypeImage  GroupMessageType = "image"
	GroupMsgTypeVideo  GroupMessageType = "video"
	GroupMsgTypeWXCard GroupMessageType = "wxcard"
)

type GroupMessage struct {
	// 图文消息为mpnews，文本消息为text，语音为voice，音乐为music，图片为image，视频为video，卡券为wxcard
	MsgType GroupMessageType `json:"msgtype"`

	MPNews *MediaID `json:"mpnews,omitempty"`
	Text   *Text    `json:"text,omitempty"`
	Voice  *MediaID `json:"voice,omitempty"`
	Image  *MediaID `json:"image,omitempty"`

	// 群发视频时需要预先设置视频标题及描述, 可通过 SetGroupVideoMessageMediaInfo() 方法设置
	MPVideo *MediaID `json:"mpvideo,omitempty"`

	WxCard *WxCard `json:"wxcard,omitempty"`
}

// 预览接口【订阅号与服务号认证后均可用】
//
// 每日调用次数有限制（100次）
// 优先使用 toWXName(用户微信号)
// 发送视频消息时需要通过 SetGroupVideoMessageMediaInfo() 设置视频标题及描述
func (gm *GroupMessage) Preview(toOpenid, toWXName, token string) (msgID, msgDataID int, err error) {
	msg := &previewGroupMessage{
		ToUser:       toOpenid,
		ToWXName:     toWXName,
		GroupMessage: gm,
	}
	uri := "https://api.weixin.qq.com/cgi-bin/message/mass/preview?access_token=" + token
	return gm.postData(uri, msg)
}

func (gm *GroupMessage) postData(uri string, value interface{}) (msgID, msgDataID int, err error) {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(value)
	if err != nil {
		return 0, 0, err
	}
	resp, err := http.Post(uri, "application/json;charset=utf-8", bytes.NewReader(buf.Bytes()))
	if err != nil {
		return 0, 0, nil
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	res := &struct {
		MsgID     int `json:"msg_id"`
		MsgDataID int `json:"msg_data_id"`
		*pkg.Error
	}{}
	err = json.Unmarshal(data, res)
	if err != nil {
		return 0, 0, err
	} else {
		if res.Error.ErrCode != 0 {
			err = res.Error
		}
		return res.MsgID, res.MsgDataID, err
	}
}

// 根据标签进行群发【订阅号与服务号认证后均可用】
//
// 群发接口新增了原创校验, 开发者可指定待群发的文章被判定为转载时，是否继续群发.
// stopWhenReprint 设置为 true 时, 当文章被判断为转载时, 停止群发.
// stopWhenReprint 设置为 false 时, 文章被判定为转载时，且原创文允许转载时，将继续进行群发操作.
//
// 发送视频消息时需要通过 SetGroupVideoMessageMediaInfo() 设置视频标题及描述, 只能发送该函数返回的 MediaID
//
// 群发接口新增 clientMsgID 参数，开发者调用群发接口时可以主动设置 clientMsgID 参数，避免重复推送。
// 群发时，微信后台将对 24 小时内的群发记录进行检查，如果该 clientMsgID 已经存在一条群发记录，
// 则会拒绝本次群发请求，返回已存在的群发msgid，开发者可以调用“查询群发消息发送状态”接口查看该条群发的状态。
//
// msgID	消息发送任务的ID
// msgDataID	消息的数据ID，该字段只有在群发图文消息时，才会出现。可以用于在图文分析数据接口中，
// 获取到对应的图文消息的数据，是图文分析数据接口中的msgid字段中的前半部分，详见图文分析数据接口中
// 的msgid字段的介绍。
func (gm *GroupMessage) SendByTag(tagID, clientMsgID int, stopWhenReprint bool, token string) (msgID, msgDataID int, err error) {
	msg := &filterGroupMessage{
		Filter: &tagFilter{
			ISToAll: tagID == 0,
			TagID:   tagID,
		},
		GroupMessage: gm,
		ClientMsgID:  clientMsgID,
	}
	if !stopWhenReprint {
		msg.SendIgnoreReprint = 1
	}
	uri := "https://api.weixin.qq.com/cgi-bin/message/mass/sendall?access_token=" + token
	return gm.postData(uri, msg)
}

// 根据OpenID列表群发【订阅号不可用，服务号认证后可用】
//
// 群发接口新增了原创校验, 开发者可指定待群发的文章被判定为转载时，是否继续群发.
// stopWhenReprint 设置为 true 时, 当文章被判断为转载时, 停止群发.
// stopWhenReprint 设置为 false 时, 文章被判定为转载时，且原创文允许转载时，将继续进行群发操作.
//
// 群发接口新增 clientMsgID 参数，开发者调用群发接口时可以主动设置 clientMsgID 参数，避免重复推送。
// 群发时，微信后台将对 24 小时内的群发记录进行检查，如果该 clientMsgID 已经存在一条群发记录，
// 则会拒绝本次群发请求，返回已存在的群发msgid，开发者可以调用“查询群发消息发送状态”接口查看该条群发的状态。
//
// 发送视频消息时需要通过 SetGroupVideoMessageMediaInfo() 设置视频标题及描述, 只能发送该函数返回的 MediaID
//
// msgID	消息发送任务的ID
// msgDataID	消息的数据ID，该字段只有在群发图文消息时，才会出现。可以用于在图文分析数据接口中，
// 获取到对应的图文消息的数据，是图文分析数据接口中的msgid字段中的前半部分，详见图文分析数据接口中
// 的msgid字段的介绍。
func (gm *GroupMessage) SendByOpenIDs(clientMsgID int, openids []string, stopWhenReprint bool, token string) (msgID, msgDataID int, err error) {
	msg := &filterGroupMessage{
		ToUser:       openids,
		GroupMessage: gm,
		ClientMsgID:  clientMsgID,
	}
	if !stopWhenReprint {
		msg.SendIgnoreReprint = 1
	}
	uri := "https://api.weixin.qq.com/cgi-bin/message/mass/send?access_token=" + token
	return gm.postData(uri, msg)
}

// 预览群发消息
type previewGroupMessage struct {
	// 接收预览用户的 openid
	ToUser string `json:"touser,omitempty"`

	// 接收预览用户的微信号, 当微信号与 openid 同时使用时, 优先使用微信号
	ToWXName string `json:"towxname,omitempty"`

	*GroupMessage
}

type GroupWXCard struct {
	CardID  string          `json:"card_id"`
	CardExt *GroupWXCardExt `json:"card_ext"`
}

type GroupWXCardExt struct {
	Code      string `json:"code"`
	Openid    string `json:"openid"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

type filterGroupMessage struct {
	Filter *tagFilter `json:"filter,omitempty"`
	ToUser []string   `json:"touser,omitempty"`
	*GroupMessage

	// 群发接口新增 send_ignore_reprint 参数，开发者可以对群发接口的 send_ignore_reprint 参数进行设置，
	// 指定待群发的文章被判定为转载时，是否继续群发。
	//
	// 当 send_ignore_reprint 参数设置为1时，文章被判定为转载时，且原创文允许转载时，将继续进行群发操作。
	//
	// 当 send_ignore_reprint 参数设置为0时，文章被判定为转载时，将停止群发操作。
	//
	// send_ignore_reprint 默认为0。
	SendIgnoreReprint int `json:"send_ignore_reprint"`

	// 群发接口新增 clientmsgid 参数，开发者调用群发接口时可以主动设置 clientmsgid 参数，避免重复推送。
	//
	// 群发时，微信后台将对 24 小时内的群发记录进行检查，如果该 clientmsgid 已经存在一条群发记录，则会拒绝本次群发请求，
	// 返回已存在的群发msgid，开发者可以调用“查询群发消息发送状态”接口查看该条群发的状态。
	ClientMsgID int `json:"clientmsgid,omitempty"`
}

type tagFilter struct {
	ISToAll bool `json:"is_to_all"`
	TagID   int  `json:"tag_id,omitempty"`
}

// 设置群发视频的视频信息, 返回新的视频媒体 ID, 通过新的视频媒体 ID 发送给用户
func SetGroupVideoMessageMediaInfo(mediaID, title, description, token string) (msgMediaID string, createdAt int64, err error) {
	uri := "https://api.weixin.qq.com/cgi-bin/media/uploadvideo?access_token=" + token
	res := &struct {
		MediaID   string `json:"media_id"`
		CreatedAt int64  `json:"created_at"`
	}{}
	data := map[string]string{
		"media_id":    mediaID,
		"title":       title,
		"description": description,
	}
	err = pkg.PostSchema(pkg.KindJson, uri, data, res)
	if err != nil {
		return "", 0, err
	} else {
		return res.MediaID, res.CreatedAt, nil
	}
}

// 获取群发速度
//
// speed 群发速度的级别
// realspeed 群发速度的真实值 单位：万/分钟
func GetGroupMsgSpeed(token string) (speed, realSpeed int, err error) {
	uri := "https://api.weixin.qq.com/cgi-bin/message/mass/speed/get?access_token=" + token
	res := &struct {
		Speed     int `json:"speed"`
		RealSpeed int `json:"realspeed"`
	}{}
	err = pkg.PostSchema(pkg.KindJson, uri, nil, res)
	if err != nil {
		return 0, 0, err
	} else {
		return res.Speed, res.RealSpeed, nil
	}
}

// 设置群发速度
//
// speed 是一个0到4的整数，数字越大表示群发速度越慢。
//
// speed 与 realspeed 的关系如下：
//
// speed	realspeed
// 0	    80w/分钟
// 1	    60w/分钟
// 2	    45w/分钟
// 3	    30w/分钟
// 4	    10w/分钟
func SetGroupMsgSpeed(speed int, token string) error {
	uri := "https://api.weixin.qq.com/cgi-bin/message/mass/speed/set?access_token=" + token
	return pkg.PostSchema(pkg.KindJson, uri, map[string]int{"speed": speed}, nil)
}
