package pkg

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrValidateAppID     = errors.New("appid 校验失败")
	ErrValidateSignature = errors.New("签名校验失败")
)

// 发送的加密消息格式
type EncryptedMsg struct {
	XMLName      xml.Name `xml:"xml" json:"-"`
	Encrypt      Cdata    // 消息主体
	MsgSignature string   // 消息签名
	TimeStamp    int64    // 时间戳
	Nonce        string   // 随机字符串
}

// 收到的加密消息格式
type DecryptedMsg struct {
	XMLName xml.Name `xml:"xml" json:"-"`
	Encrypt string   // 消息主体
}

// 加/解密器
type WXBizMsgCrypt struct {
	token  string
	aesKey []byte
	appID  string
}

func NewWXBizMsgCrypt(aesToken, encodingAesKey, appid string) (crypt *WXBizMsgCrypt, err error) {
	data, err := base64.StdEncoding.DecodeString(encodingAesKey + "=")
	if err != nil {
		return nil, err
	} else {
		return &WXBizMsgCrypt{
			token:  aesToken,
			aesKey: data,
			appID:  appid,
		}, nil
	}
}

// 对明文进行加密
func (mc *WXBizMsgCrypt) Encrypt(random string, text []byte) (base64Encrypted string, err error) {
	buf := bytes.NewBuffer(nil)

	// randomStr + networkBytesOrder + text + appid
	buf.WriteString(random)
	buf.Write(getNetworkBytesOrder(len(text)))
	buf.Write(text)
	buf.WriteString(mc.appID)

	// ... + pad: 使用自定义的填充方式对明文进行补位填充
	padding := pkcs7Encode(buf.Len())
	buf.Write(padding)

	// 获得最终的字节流, 未加密
	unencrypted := buf.Bytes()

	// 设置加密模式为AES的CBC模式
	block, err := aes.NewCipher(mc.aesKey)
	if err != nil {
		return "", err
	}
	encrypter := cipher.NewCBCEncrypter(block, mc.aesKey[:16])
	encryped := make([]byte, len(unencrypted))
	encrypter.CryptBlocks(encryped, unencrypted)
	return base64.StdEncoding.EncodeToString(encryped), nil
}

// 对密文进行解密
func (mc *WXBizMsgCrypt) Decrypt(text string) (original []byte, err error) {
	// 设置解密模式为AES的CBC模式
	block, err := aes.NewCipher(mc.aesKey)
	if err != nil {
		return nil, err
	}
	decrypter := cipher.NewCBCDecrypter(block, mc.aesKey[:16])
	if err != nil {
		return nil, err
	}

	// 使用BASE64对密文进行解码
	src, err := base64.StdEncoding.DecodeString(text)
	dst := make([]byte, len(src))

	// 解密
	decrypter.CryptBlocks(dst, src)

	// 去除补位字符
	dst = pkcs7Decode(dst)

	// 分离16位随机字符串,网络字节序和AppId
	networkOrder := dst[16:20]
	xmlLength := recoverNetworkBytesOrder(networkOrder)
	xmlContent := dst[20 : 20+xmlLength]
	formAppID := string(dst[20+xmlLength:])
	if formAppID != mc.appID {
		return nil, ErrValidateAppID
	}
	return xmlContent, nil
}

// 将公众平台回复用户的消息加密打包.
func (mc *WXBizMsgCrypt) EncryptMsg(reply []byte, timeStamp int64, nonce string) (encrypted *EncryptedMsg, err error) {
	// 加密
	encryptText, err := mc.Encrypt(randStr(16), reply)
	if err != nil {
		return nil, err
	}
	// 生成安全签名
	if timeStamp == 0 {
		timeStamp = time.Now().Unix()
	}
	if nonce == "" {
		nonce = randStr(16)
	}
	timeStampStr := strconv.FormatInt(timeStamp, 10)
	// 数据签名
	signature := SignParams(mc.token, timeStampStr, nonce, encryptText)
	return &EncryptedMsg{
		Encrypt:      Cdata{Value: encryptText},
		MsgSignature: signature,
		TimeStamp:    timeStamp,
		Nonce:        nonce,
	}, nil
}

// 检验消息的真实性，并且获取解密后的明文
func (mc *WXBizMsgCrypt) DecryptMsg(msgSignature, timeStamp, nonce string, postData []byte) (decryptedMsg []byte, err error) {
	msg := &DecryptedMsg{}
	err = xml.Unmarshal(postData, msg)
	if err != nil {
		return nil, err
	}
	// 验证签名
	signature := SignParams(mc.token, timeStamp, nonce, msg.Encrypt)
	if signature != msgSignature {
		return nil, ErrValidateSignature
	}
	return mc.Decrypt(msg.Encrypt)
}

// 解析请求中的加密消息
func (mc *WXBizMsgCrypt) DecryptRequest(r *http.Request) (decryptedMsg []byte, err error) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	} else {
		defer r.Body.Close()
		q := r.URL.Query()
		timestamp := q.Get("timestamp")
		nonce := q.Get("nonce")
		signature := q.Get("msg_signature")
		return mc.DecryptMsg(signature, timestamp, nonce, data)
	}
}

// 生成4个字节的网络字节序
func getNetworkBytesOrder(n int) (orderBytes []byte) {
	orderBytes = make([]byte, 4)
	orderBytes[0] = byte(n >> 24)
	orderBytes[1] = byte(n >> 16)
	orderBytes[2] = byte(n >> 8)
	orderBytes[3] = byte(n)
	return orderBytes
}

// 还原4个字节的网络字节序
func recoverNetworkBytesOrder(orderBytes []byte) (n int) {
	return int(orderBytes[0])<<24 | int(orderBytes[1])<<16 | int(orderBytes[2])<<8 | int(orderBytes[3])
}
