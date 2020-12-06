package src

type AccessTokenGetter func() (token string, err error)

type ExpireData struct {
	Value     string
	ExpiredAt int64
}

// 公众号 access token， refresh token 没有过期时间
type AppAccessToken struct {
	AccessToken  string
	ExpireAt     int64
	RefreshToken string
}
