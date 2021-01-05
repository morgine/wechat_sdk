package src

import (
	"encoding/json"
	"github.com/morgine/wechat_sdk/pkg/open_platform"
	"time"
)

// 开放平台数据存储接口
type ComponentStorage interface {
	SaveVerifyTicket(ticket string) error
	GetVerifyTicket() (string, error) // 获得 ticket, 如果 ticket 不存在，则返回空字符串
	SaveAccessToken(data *ExpireData) error
	GetAccessToken() (*ExpireData, error) // 获得 access token, 如果 token 不存在，则返回 nil
	//SavePreAuthCode(data *ExpireData) error
	//GetPreAuthCode() (*ExpireData, error) // 获得 pre auth code, 如果 code 不存在，则返回 nil
	SaveAppAccessToken(appid string, token *AppAccessToken) error
	GetAppAccessToken(appid string) (*AppAccessToken, error) // 获得公众号 token, 如果 token 不存在，则返回 nil
}

type AccessStorage interface {
	Set(key string, value []byte, expiration time.Duration) error
	Get(key string) (value []byte, err error)
}

type AppNickname struct {
	Appid    string
	Nickname string
}

type AppStorage interface {
	SaveAppInfo(appid string, app *open_platform.AuthorizerInfo) error // 保存公众号信息
	GetAppInfo(appid string) (*open_platform.AuthorizerInfo, error)    // 获得公众号信息，如果信息不存在，则返回 nil
	DelAppInfo(appid string) error                                     // 删除公众号信息
	DelAppInfoNotIn(appids []string) error                             // 删除不存在于 appids 中的所有其他公众号信息
}

type componentStorage struct {
	keyPrefix string
	client    AccessStorage
}

func NewComponentStorage(appid string, storage AccessStorage) ComponentStorage {
	return &componentStorage{
		keyPrefix: appid + "_",
		client:    storage,
	}
}

func (c *componentStorage) SaveVerifyTicket(ticket string) error {
	return c.set("ticket", []byte(ticket), 0)
}

func (c *componentStorage) GetVerifyTicket() (string, error) {
	value, err := c.get("ticket")
	if err != nil {
		return "", err
	} else {
		return string(value), nil
	}
}

func (c *componentStorage) SaveAccessToken(data *ExpireData) error {
	return c.marshalJSON("access_token", data, time.Unix(data.ExpiredAt, 0).Sub(time.Now()))
}

func (c *componentStorage) GetAccessToken() (*ExpireData, error) {
	data := &ExpireData{}
	err := c.unmarshalJSON("access_token", data)
	if err != nil {
		return nil, err
	}
	if data.Value != "" {
		return data, nil
	} else {
		return nil, nil
	}
}

//func (c *componentStorage) SavePreAuthCode(data *ExpireData) error {
//	return c.marshalJSON("pre_auth_code", data, time.Unix(data.ExpiredAt, 0).Sub(time.Now()))
//}
//
//func (c *componentStorage) GetPreAuthCode() (*ExpireData, error) {
//	data := &ExpireData{}
//	err := c.unmarshalJSON("pre_auth_code", data)
//	if err != nil {
//		return nil, err
//	}
//	if data.Value != "" {
//		return data, nil
//	} else {
//		return nil, nil
//	}
//}

func (c *componentStorage) SaveAppAccessToken(appid string, token *AppAccessToken) error {
	return c.marshalJSON("app_access_token_"+appid, token, 0)
}

func (c *componentStorage) GetAppAccessToken(appid string) (*AppAccessToken, error) {
	data := &AppAccessToken{}
	err := c.unmarshalJSON("app_access_token_"+appid, data)
	if err != nil {
		return nil, err
	}
	if data.RefreshToken != "" {
		return data, nil
	} else {
		return nil, nil
	}
}

func (c *componentStorage) set(key string, value []byte, expiration time.Duration) error {
	return c.client.Set(c.keyPrefix+key, value, expiration)
}

func (c *componentStorage) get(key string) (value []byte, err error) {
	return c.client.Get(c.keyPrefix + key)
}

func (c *componentStorage) marshalJSON(key string, obj interface{}, expiration time.Duration) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.set(key, data, expiration)
}

func (c *componentStorage) unmarshalJSON(key string, obj interface{}) error {
	data, err := c.get(key)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, obj)
}
