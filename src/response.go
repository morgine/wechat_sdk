package src

import (
	"fmt"
	"github.com/morgine/wechat_sdk/pkg"
	"github.com/morgine/wechat_sdk/pkg/message"
	"net/http"
)

// 自动回复接口
type ResponseWriter interface {
	ResponseText(text string) error
	ResponseImage(mediaID string) error
	ResponseVoice(mediaID string) error
	ResponseVideo(mediaID, title, description string) error
	ResponseMusic(m Music) error
	ResponseArticles([]Article) error
}

type responseWriter struct {
	msgCrypt *pkg.WXBizMsgCrypt
	history  *message.ResponseMessage
	w        http.ResponseWriter
	msg      *message.ServerMessage
}

func (r *responseWriter) ResponseText(text string) error {
	return r.response(&message.ResponseMessage{
		MsgType: message.ResponseMsgTypeText,
		Content: &pkg.Cdata{Value: text},
	})
}

func (r *responseWriter) ResponseImage(mediaID string) error {
	return r.response(&message.ResponseMessage{
		MsgType: message.ResponseMsgTypeImage,
		Image:   &message.ResImage{MediaId: pkg.Cdata{Value: mediaID}},
	})
}

func (r *responseWriter) ResponseVoice(mediaID string) error {
	return r.response(&message.ResponseMessage{
		MsgType: message.ResponseMsgTypeVoice,
		Voice:   &message.ResVoice{MediaId: pkg.Cdata{Value: mediaID}},
	})
}

func (r *responseWriter) ResponseVideo(mediaID, title, description string) error {
	return r.response(&message.ResponseMessage{
		MsgType: message.ResponseMsgTypeVideo,
		Video: &message.ResVideo{
			MediaId:     pkg.Cdata{Value: mediaID},
			Title:       pkg.Cdata{Value: title},
			Description: pkg.Cdata{Value: description},
		},
	})
}

func (r *responseWriter) ResponseMusic(m Music) error {
	return r.response(&message.ResponseMessage{
		MsgType: message.ResponseMsgTypeMusic,
		Music: &message.ResMusic{
			Title:        pkg.Cdata{Value: m.Title},
			Description:  pkg.Cdata{Value: m.Description},
			MusicURL:     pkg.Cdata{Value: m.MusicURL},
			HQMusicUrl:   pkg.Cdata{Value: m.HQMusicUrl},
			ThumbMediaId: pkg.Cdata{Value: m.ThumbMediaId},
		},
	})
}

func (r *responseWriter) ResponseArticles(articles []Article) error {
	var arts = make(message.ResArticles, len(articles))
	for i := 0; i < len(articles); i++ {
		art := articles[i]
		arts[i] = &message.ResArticle{
			Title:       pkg.Cdata{Value: art.Title},
			Description: pkg.Cdata{Value: art.Description},
			Url:         pkg.Cdata{Value: art.Url},
			PicUrl:      pkg.Cdata{Value: art.PicUrl},
		}
	}
	return r.response(&message.ResponseMessage{
		MsgType:  message.ResponseMsgTypeNews,
		Articles: &arts,
	})
}

func (r *responseWriter) response(msg *message.ResponseMessage) error {
	if r.history != nil {
		return fmt.Errorf("cannot respone: %+v\nalready response: %+v\n", *msg, *r.history)
	} else {
		r.history = msg
		msg.CreateTime = Now().Unix()
		err := message.Response(r.msg, msg, r.w, r.msgCrypt)
		if err != nil {
			return err
		}
	}
	return nil
}
