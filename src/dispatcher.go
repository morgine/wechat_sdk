package src

import (
	"github.com/morgine/wechat/pkg/message"
)

type Context struct {
	ResponseWriter
	app                     *AppInfo
	appGetter               func() (*AppInfo, error)
	customerMsgSender       CustomerMsgSender
	customerMsgSenderGetter func() CustomerMsgSender
	values                  map[string]interface{}
}

func newContext(
	ag func() (*AppInfo, error),
	cg func() CustomerMsgSender,
	w ResponseWriter,
) *Context {
	return &Context{
		ResponseWriter:          w,
		app:                     nil,
		appGetter:               ag,
		customerMsgSender:       nil,
		customerMsgSenderGetter: cg,
		values:                  map[string]interface{}{},
	}
}

func (ctx *Context) Get(key string) (value interface{}, ok bool) {
	value, ok = ctx.values[key]
	return
}

func (ctx *Context) Set(key string, value interface{}) {
	ctx.values[key] = value
}

func (ctx *Context) AppInfo() (*AppInfo, error) {
	if ctx.app == nil {
		app, err := ctx.appGetter()
		if err != nil {
			return nil, err
		} else {
			ctx.app = app
		}
	}
	return ctx.app, nil
}

func (ctx *Context) CustomerMsgSender() CustomerMsgSender {
	if ctx.customerMsgSender == nil {
		ctx.customerMsgSender = ctx.customerMsgSenderGetter()
	}
	return ctx.customerMsgSender
}

type TextMsgHandler func(msg *message.TextMessage, ctx *Context)

type EventMsgHandler func(msg *message.EventMessage, ctx *Context)

type Music struct {
	Title        string // 标题(可选)
	Description  string // 描述(可选)
	MusicURL     string // 音乐连接
	HQMusicUrl   string // 高质量音乐链接，WIFI环境优先使用该链接播放音乐
	ThumbMediaId string // 缩略图的媒体id，通过素材管理中的接口上传多媒体文件，得到的id
}

type Article struct {
	Title       string
	Description string
	Url         string
	PicUrl      string
}

type Dispatcher struct {
	textMsgHandlers []TextMsgHandler
	eventHandlers   map[message.EventType][]EventMsgHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{eventHandlers: map[message.EventType][]EventMsgHandler{}}
}

// 添加事件处理器
func (d *Dispatcher) SubscribeEvent(evt message.EventType, h EventMsgHandler) {
	d.eventHandlers[evt] = append(d.eventHandlers[evt], h)
}

// 添加文本消息处理器
func (d *Dispatcher) SubscribeTextMsg(h TextMsgHandler) {
	d.textMsgHandlers = append(d.textMsgHandlers, h)
}

func (d *Dispatcher) trigger(
	msg *message.ServerMessage,
	data message.ServerMessageData,
	ag func() (*AppInfo, error),
	cg func() CustomerMsgSender,
	w ResponseWriter,
) error {
	switch msg.MsgType {
	case message.ServerMsgTypeEvent:
		eventMsg, err := data.MarshalEvent()
		if err != nil {
			return err
		}
		ctx := newContext(ag, cg, w)
		if handlers, ok := d.eventHandlers[eventMsg.Event]; ok {
			for _, h := range handlers {
				h(eventMsg, ctx)
			}
		}
	case message.ServerMsgTypeText:
		if len(d.textMsgHandlers) > 0 {
			textMsg, err := data.MarshalTextMessage()
			if err != nil {
				return err
			} else {
				ctx := newContext(ag, cg, w)
				for _, handler := range d.textMsgHandlers {
					handler(textMsg, ctx)
				}
			}
		}
	}
	return nil
}
