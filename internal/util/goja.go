package util

import (
	"errors"
	"github.com/dop251/goja"
)

func ExportGojaValue(value goja.Value) (interface{}, error) {
	if o, ok := value.(*goja.Object); ok {
		if b, ok := o.Export().(goja.ArrayBuffer); ok { // 如果返回值为 ArrayBuffer 类型，则转换为 []byte
			return b.Bytes(), nil
		}
		if "Uint8Array" == o.Get("constructor").(*goja.Object).Get("name").String() { // 如果返回值为 Uint8Array 类型，则转换为 []byte
			return o.Get("buffer").Export().(goja.ArrayBuffer).Bytes(), nil
		}
		if p, ok := o.Export().(*goja.Promise); ok {
			switch p.State() {
			case goja.PromiseStateRejected:
				return nil, errors.New(p.Result().String())
			case goja.PromiseStateFulfilled:
				return ExportGojaValue(p.Result())
			default:
				return nil, errors.New("unexpected promise state pending")
			}
		}
	}

	return value.Export(), nil
}
