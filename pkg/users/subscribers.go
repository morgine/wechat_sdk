package users

import (
	"github.com/morgine/wechat/pkg"
)

// 获得已订阅用户列表
func GetSubscribers(token string, walk func(openids []string) error) error {
	var nextOpenid string
	for {
		users, err := GetNextSubscribers(token, nextOpenid)
		if err != nil {
			return err
		} else {
			err = walk(users.Data.Openid)
			if err != nil {
				return err
			} else {
				if users.NextOpenid == "" {
					return nil
				} else {
					nextOpenid = users.NextOpenid
				}
			}
		}
	}
}

func GetNextSubscribers(token, nextOpenID string) (users *Users, err error) {
	uri := "https://api.weixin.qq.com/cgi-bin/user/get?access_token=" + token
	if nextOpenID != "" {
		uri += "&next_openid=" + nextOpenID
	}
	users = &Users{}
	err = pkg.GetJson(uri, users)
	if err != nil {
		return nil, err
	} else {
		return users, nil
	}
}

type Users struct {
	Total      int     `json:"total"`
	Count      int     `json:"count"`
	Data       Openids `json:"data"`
	NextOpenid string  `json:"next_openid"`
}

type Openids struct {
	Openid []string `json:"openid"`
}
