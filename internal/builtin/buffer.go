package builtin

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/dop251/goja"
)

func init() {
	Builtins["Buffer"] = func(runtime *goja.Runtime) interface{} {
		o := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
			return runtime.ToValue(&Buffer{}).ToObject(runtime)
		}).ToObject(runtime)

		o.Set("from", func(input []byte, encoding string) (*Buffer, error) {
			dat, err := decode(input, encoding)
			return (*Buffer)(&dat), err
		})

		return o
	}
}

type Buffer []byte

func (b *Buffer) ToString(encoding string) (string, error) {
	return encode(([]byte)(*b), encoding)
}

func encode(input []byte, encoding string) (string, error) {
	switch encoding {
	case "", "utf8":
		return string(input), nil
	case "hex":
		return hex.EncodeToString(input), nil
	case "base64":
		return base64.StdEncoding.EncodeToString(input), nil
	case "base64url":
		return base64.URLEncoding.EncodeToString(input), nil
	}
	return "", errors.New("unsupported encoding: " + encoding)
}

func decode(input []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "", "utf8":
		return input, nil
	case "hex":
		return hex.DecodeString(string(input))
	case "base64":
		return base64.StdEncoding.DecodeString(string(input))
	case "base64url":
		return base64.URLEncoding.DecodeString(string(input))
	}
	return nil, errors.New("unsupported encoding: " + encoding)
}
