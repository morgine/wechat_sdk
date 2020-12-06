package pkg

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/google/go-querystring/query"
	"hash"
	"io"
	"net/url"
	"sort"
	"strings"
)

// 获得结构体数据的签名，结构体字段可以使用 url 标签进行重命名
// 获得 md5 签名：Sign(schema, md5.New(), ""), 微信还规定要将结果转换为大写
// 获得 sha1 签名：Sign(schema, sha1.New(), "")
// key 可为空
func SignSchema(schema interface{}, mod hash.Hash, key string) (sign string, err error) {
	vs, e := query.Values(schema)
	if e != nil {
		return "", e
	}
	//STEP 1: 对key进行升序排序，略过空值
	var keys []string
	for key, vues := range vs {
		if len(vues) > 0 && vues[0] != "" && vues[0] != "0" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	//STEP2: 对key=value的键值对用&连接起来
	var kv string
	for _, k := range keys {
		kv += k + "=" + vs[k][0] + "&"
	}

	//STEP3, 在键值对的最后加上key=API_KEY
	if key != "" {
		kv += "key=" + key
	} else {
		kv = strings.TrimSuffix(kv, "&")
	}
	//STEP4：hash 序列化，可以为 MD5 或 SHA1，如果是 MD5 需要将结果转换为大写
	mod.Write([]byte(kv))
	cipherStr := mod.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(cipherStr)), nil
}

// 检查是否是微信服务器发送的消息
func CheckSignature(vs url.Values, token string) bool {
	signature := vs.Get("signature")
	timestamp := vs.Get("timestamp")
	nonce := vs.Get("nonce")
	return SignParams(token, timestamp, nonce) == signature
}

// 生成请求参数签名, 当本地服务器送加密消息到服务器时, 需要加入签名数据已确保该消息不是第三方发送的消息
func SignParams(params ...string) string {
	sort.Strings(params)
	h := sha1.New()
	for _, s := range params {
		_, _ = io.WriteString(h, s)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// 获得 JS API 签名. 同一个 url 仅需调用一次, 见附录1: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141115
// nonce 为随机字符串, 需要与前端一致
// timestamp 为时间戳, 需要与前端一致
// refererUrl 为当前网页的URL, 包含 "http(s)://", 不包含#及其后面部分
func GetJsApiSignature(ticket, nonce, refererUrl string, timestamp int64) (signature string, err error) {
	s := &jsApiSignSchema{
		Noncestr:    nonce,
		JsapiTicket: ticket,
		Timestamp:   timestamp,
		Url:         refererUrl,
	}
	signature, err = SignSchema(s, sha1.New(), "")
	if err != nil {
		return "", fmt.Errorf("获取 JS API 签名出错：%s", err)
	} else {
		return signature, nil
	}
}

type jsApiSignSchema struct {
	Noncestr    string `url:"noncestr"`
	JsapiTicket string `url:"jsapi_ticket"`
	Timestamp   int64  `url:"timestamp"`
	Url         string `url:"url"`
}
