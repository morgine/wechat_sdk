package pkg

import (
	"math/rand"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().Unix()))
var source = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randBy(bits int, source []byte) string {
	bytes := make([]byte, bits)
	n := len(source)
	for i := 0; i < bits; i++ {
		bytes[i] = source[r.Intn(n)]
	}
	return string(bytes)
}

// randStr 生成随机字符串
func randStr(bits int) string {
	return randBy(bits, source)
}
