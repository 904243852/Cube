package module

import "encoding/base64"

func init() {
	register("base64", func(worker Worker, db Db) interface{} {
		return &Base64Client{}
	})
}

type Base64Client struct{}

func (b *Base64Client) Encode(input []byte) string { // 在 js 中调用该方法时，入参可接受 string 或 Uint8Array 类型
	return base64.StdEncoding.EncodeToString(input)
}

func (b *Base64Client) Decode(input string) ([]byte, error) { // 返回的 []byte 类型将隐式地转换为 js/ts 中的 Uint8Array 类型
	return base64.StdEncoding.DecodeString(input)
}
