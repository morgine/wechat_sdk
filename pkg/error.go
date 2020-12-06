package pkg

import "fmt"

type Error struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (we *Error) Error() string {
	return fmt.Sprintf("code:[%d] error:[%s]", we.ErrCode, we.ErrMsg)
}
