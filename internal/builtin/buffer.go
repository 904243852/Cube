package builtin

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/dop251/goja"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		runtime := worker.Runtime()

		o := runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
			return runtime.ToValue(&Buffer{}).ToObject(runtime)
		}).ToObject(runtime)

		o.Set("from", func(input []byte, encoding string) (*Buffer, error) {
			dat, err := decode(input, encoding)
			return (*Buffer)(&dat), err
		})

		runtime.Set("Buffer", o)
	})
}

type Buffer []byte

func (b *Buffer) ToString(encoding string) (string, error) {
	return encode(*b, encoding)
}

func (b *Buffer) ToJson() (obj interface{}, err error) {
	err = json.Unmarshal(*b, &obj)
	return
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
