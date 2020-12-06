package src

import (
	"github.com/morgine/wechat_sdk/pkg/message"
	"log"
)

// 客服消息发送器
// TODO: 完善消息发送种类
type CustomerMsgSender interface {
	SendMiniProgramPage(toOpenid []string, page *message.MiniProgramPage) error // 回复小程序卡片
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
