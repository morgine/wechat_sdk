package access

import (
	"github.com/morgine/wechat_sdk/pkg"
)

type AppAccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type AppTicket struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int64  `json:"expires_in"`
}

func GetAppAccessToken(appid, appSecret string) (token *AppAccessToken, err error) {
	uri := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + appid + "&secret=" + appSecret
	token = &AppAccessToken{}
	err = pkg.GetJson(uri, token)
	if err != nil {
		return nil, err
	} else {
		return token, nil
	}
}

func GetAppTicket(accessToken string) (ticket *AppTicket, err error) {
	uri := "https://api.weixin.qq.com/cgi-bin/ticket/getticket?type=jsapi&access_token=" + accessToken
	ticket = &AppTicket{}
	err = pkg.GetJson(uri, ticket)
	if err != nil {
		return nil, err
	} else {
		return ticket, nil
	}
}
