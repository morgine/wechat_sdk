package custom_menu

import "github.com/morgine/wechat_sdk/pkg"

func Delete(accessToken string) error {
	return pkg.GetJson("https://api.weixin.qq.com/cgi-bin/menu/delete?access_token="+accessToken, nil)
}
