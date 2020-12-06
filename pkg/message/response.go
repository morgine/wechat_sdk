package message

import (
	"encoding/xml"
	"github.com/morgine/wechat/pkg"
)

// 被动回复消息类型
type ResponseMsgType string

var (
	ResponseMsgTypeText  ResponseMsgType = "text"
	ResponseMsgTypeImage ResponseMsgType = "image"
	ResponseMsgTypeVoice ResponseMsgType = "voice"
	ResponseMsgTypeVideo ResponseMsgType = "video"
	ResponseMsgTypeMusic ResponseMsgType = "music"

	// 图文消息, 点击后直接跳转到连接地址
	ResponseMsgTypeNews ResponseMsgType = "news"
)

// ResponseMessage 是被动回复消息.
// 当用户触发微信事件时, 微信服务器将会 POST 一个消息到本地服务器, 本地服务器可在 5 秒之内响应被动消息.
type ResponseMessage struct {
	XMLName      xml.Name  `xml:"xml"`
	ToUserName   pkg.Cdata // 接收方帐号（收到的OpenID）
	FromUserName pkg.Cdata // 开发者微信号(原始ID)
	CreateTime   int64     // 消息创建时间 （整型）

	MsgType ResponseMsgType // text|image|voice|video|music|news

	Content *pkg.Cdata `xml:",omitempty"`

	Image *ResImage `xml:",omitempty"`

	Voice *ResVoice `xml:",omitempty"`

	Video *ResVideo `xml:",omitempty"`

	Music *ResMusic `xml:",omitempty"`

	ArticleCount int `xml:",omitempty"`

	// 被动图文消息可发8条，如果图文数超过限制，则将只发限制内的条数
	Articles *ResArticles `xml:"Articles>item,omitempty"`
}

// 转换为客服消息
func (rm *ResponseMessage) ToCustomerMessage() *CustomerMessage {
	cm := &CustomerMessage{
		MsgType: CustomerMsgType(rm.MsgType),
		ToUser:  rm.ToUserName.Value,
	}
	switch rm.MsgType {
	case ResponseMsgTypeText:
		if rm.Content != nil {
			cm.Text = &Text{}
			cm.Text.Content = rm.Content.Value
		}
	case ResponseMsgTypeNews:
		if rm.Articles != nil {
			var articles []*CustomerArticle
			arts := *rm.Articles
			for _, art := range arts {
				articles = append(articles, &CustomerArticle{
					Title:       art.Title.Value,
					Description: art.Description.Value,
					Url:         art.Url.Value,
					PicUrl:      art.PicUrl.Value,
				})
			}
			cm.News = &CustomerNews{Articles: articles}
		}
	case ResponseMsgTypeImage:
		if rm.Image != nil {
			cm.Image = &MediaID{}
			cm.Image.MediaID = rm.Image.MediaId.Value
		}
	case ResponseMsgTypeMusic:
		if rm.Music != nil {
			cm.Music = &Music{
				Title:        rm.Music.Title.Value,
				Description:  rm.Music.Description.Value,
				Musicurl:     rm.Music.MusicURL.Value,
				Hqmusicurl:   rm.Music.HQMusicUrl.Value,
				ThumbMediaID: rm.Music.ThumbMediaId.Value,
			}
		}
	case ResponseMsgTypeVideo:
		if rm.Video != nil {
			cm.Video = &Video{
				MediaID:      rm.Video.MediaId.Value,
				ThumbMediaID: "", // 被动消息没有该字段, see: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140543
				Title:        rm.Video.Title.Value,
				Description:  rm.Video.Description.Value,
			}
		}
	case ResponseMsgTypeVoice:
		if rm.Voice != nil {
			cm.Voice = &MediaID{
				MediaID: rm.Voice.MediaId.Value,
			}
		}
	}
	return cm
}

type ResArticles []*ResArticle

type ResArticle struct {
	Title       pkg.Cdata `json:"title"`
	Description pkg.Cdata `json:"description"`
	Url         pkg.Cdata `json:"url"`
	PicUrl      pkg.Cdata `json:"picurl"`
}

type ResImage struct {
	MediaId pkg.Cdata
}

type ResVoice struct {
	MediaId pkg.Cdata
}

type ResVideo struct {
	MediaId     pkg.Cdata
	Title       pkg.Cdata `xml:",omitempty"` // 标题(可选)
	Description pkg.Cdata `xml:",omitempty"` // 描述(可选)
}

type ResMusic struct {
	Title        pkg.Cdata `xml:",omitempty"` // 标题(可选)
	Description  pkg.Cdata `xml:",omitempty"` // 描述(可选)
	MusicURL     pkg.Cdata `xml:",omitempty"` // 音乐连接
	HQMusicUrl   pkg.Cdata `xml:",omitempty"` // 高质量音乐链接，WIFI环境优先使用该链接播放音乐
	ThumbMediaId pkg.Cdata `xml:",omitempty"` // 缩略图的媒体id，通过素材管理中的接口上传多媒体文件，得到的id
}
