package message

import (
	"github.com/morgine/wechat_sdk/pkg"
)

// 客服消息类型
// see: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140547
type CustomerMsgType string

const (
	CustomerMsgTypeText            CustomerMsgType = "text"
	CustomerMsgTypeImage           CustomerMsgType = "image"
	CustomerMsgTypeVoice           CustomerMsgType = "voice"
	CustomerMsgTypeVideo           CustomerMsgType = "video"
	CustomerMsgTypeMusic           CustomerMsgType = "music"
	CustomerMsgTypeNews            CustomerMsgType = "news"            // 点击后直接跳转到连接地址
	CustomerMsgTypeMPNews          CustomerMsgType = "mpnews"          // 点击后跳转到图文
	CustomerMsgTypeMenu            CustomerMsgType = "msgmenu"         // 菜单消息, 消息发送后用户会收到一个文本菜单, 用户点击后触发事件
	CustomerMsgTypeWXCard          CustomerMsgType = "wxcard"          // 卡券
	CustomerMsgTypeMiniProgramPage CustomerMsgType = "miniprogrampage" // 小程序卡片
)

// 客服消息, 需要用户触发过菜单事件, 或发送过消息到服务器, 且必须在 48 小时之内回复用户.
type CustomerMessage struct {
	ToUser          string           `json:"touser"` // 用户 openid
	MsgType         CustomerMsgType  `json:"msgtype"`
	Text            *Text            `json:"text,omitempty"`
	Image           *MediaID         `json:"image,omitempty"`
	Voice           *MediaID         `json:"voice,omitempty"`
	Video           *Video           `json:"video,omitempty"`
	Music           *Music           `json:"music,omitempty"`
	News            *CustomerNews    `json:"news,omitempty"`
	MpNews          *MediaID         `json:"mpnews,omitempty"`
	MsgMenu         *CustomerMenuMsg `json:"msgmenu,omitempty"`
	WxCard          *WxCard          `json:"wxcard,omitempty"`
	MiniProgramPage *MiniProgramPage `json:"miniprogrampage,omitempty"`
	CustomService   *CustomService   `json:"customservice,omitempty"`
}

//48001	api 功能未授权，请确认公众号已获得该接口，可以在公众平台官网 - 开发者中心页中查看接口权限
//48002	粉丝拒收消息（粉丝在公众号选项中，关闭了 “ 接收消息 ” ）
//48004	api 接口被封禁，请登录 mp.weixin.qq.com 查看详情
//48005	api 禁止删除被自动回复和自定义菜单引用的素材
//48006	api 禁止清零调用次数，因为清零次数达到上限
//48008	没有该类型消息的发送权限

// 判断是否是终止错误, 即所有同类型的消息都不可发送. 有可能某一个公众号被封号, 导致 API 功能受限, 出现大量错误信息
func IsBreakError(err error) bool {
	if we, ok := err.(*pkg.Error); ok {
		if 48001 <= we.ErrCode && we.ErrCode <= 48008 {
			return true
		}
	}
	return false
}

// 主动推送客服消息
func (m *CustomerMessage) Send(accessToken, toUser string) (err error) {
	if toUser != "" {
		m.ToUser = toUser
	}
	u := "https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=" + accessToken
	return pkg.PostSchema(pkg.KindJson, u, m, nil)
}

// 是否客服消息常见错误
func IsCMsgCommonError(err error) bool {
	if werr, ok := err.(*pkg.Error); ok {
		switch werr.ErrCode {
		case 45015 /*超时回复*/, 43004 /*未关注*/, 45047 /*发送条数超过上限*/ :
			return true
		}
	}
	return false
}

func IsSysBusyError(err error) bool {
	if werr, ok := err.(*pkg.Error); ok {
		switch werr.ErrCode {
		case -1:
			return true
		}
	}
	return false
}

type MediaID struct {
	MediaID string `json:"media_id"`
}

type WxCard struct {
	CardID string `json:"card_id"`
}

type Music struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Musicurl     string `json:"musicurl"`
	Hqmusicurl   string `json:"hqmusicurl"`
	ThumbMediaID string `json:"thumb_media_id"`
}

type Text struct {
	Content string `json:"content"`
}

type Video struct {
	MediaID      string `json:"media_id"`
	ThumbMediaID string `json:"thumb_media_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
}

type CustomerNews struct {
	Articles []*CustomerArticle `json:"articles"`
}

type CustomerArticle struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	PicUrl      string `json:"picurl"`
}

type CustomerMenuMsg struct {
	HeadContent string                 `json:"head_content"`
	List        []*CustomerMenuMsgItem `json:"list"`
	TailContent string                 `json:"tail_content"`
}

type CustomerMenuMsgItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type MiniProgramPage struct {
	Title        string `json:"title"`
	Appid        string `json:"appid"`
	PagePath     string `json:"pagepath"`
	ThumbMediaID string `json:"thumb_media_id"`
}

// 客服账号
type CustomService struct {
	KFAccount string `json:"kf_account"`
}
