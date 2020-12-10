package src

import (
	"github.com/morgine/wechat_sdk/pkg/message"
	"log"
)

// 客服消息发送器
// TODO: 完善消息发送种类
type CustomerMsgSender interface {
	SendMiniProgramPage(toOpenid []string, page *message.MiniProgramPage) error // 发送小程序卡片
}

// 客服消息发送器
type customerMsgSender struct {
	tokenGetter AccessTokenGetter
	logger      *log.Logger
}

func newCustomerMsgSender(tokenGetter AccessTokenGetter, logger *log.Logger) CustomerMsgSender {
	return &customerMsgSender{
		tokenGetter: tokenGetter,
		logger:      logger,
	}
}

func (c *customerMsgSender) SendMiniProgramPage(toOpenid []string, page *message.MiniProgramPage) error {
	token, err := c.tokenGetter()
	if err != nil {
		return err
	}
	msg := message.CustomerMessage{
		MsgType:         message.CustomerMsgTypeMiniProgramPage,
		MiniProgramPage: page,
	}
	for _, openid := range toOpenid {
		err = msg.Send(token, openid)
		if err != nil {
			if message.IsBreakError(err) {
				return err
			}
			if !message.IsCMsgCommonError(err) {
				c.logger.Printf("send customer mini program page to %s: %s\n", openid, err)
			}
		}
	}
	return nil
}

type CustomerMsgResponser interface {
	ResponseMiniProgramPage(page *message.MiniProgramPage) error // 回复小程序卡片
}

type customerMsgResponser struct {
	openid string
	sender CustomerMsgSender
}

func (r *customerMsgResponser) ResponseMiniProgramPage(page *message.MiniProgramPage) error {
	return r.sender.SendMiniProgramPage([]string{r.openid}, page)
}

func newCustomerMsgResponser(openid string, sender CustomerMsgSender) CustomerMsgResponser {
	return &customerMsgResponser{
		openid: openid,
		sender: sender,
	}
}
