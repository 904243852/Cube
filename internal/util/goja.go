package util

import "github.com/dop251/goja"

func ExportGojaValue(value goja.Value) interface{} {
	if o, ok := value.(*goja.Object); ok {
		if b, ok := o.Export().(goja.ArrayBuffer); ok { // 如果返回值为 ArrayBuffer 类型，则转换为 []byte
			return b.Bytes()
		}
		if "Uint8Array" == o.Get("constructor").(*goja.Object).Get("name").String() { // 如果返回值为 Uint8Array 类型，则转换为 []byte
			return o.Get("buffer").Export().(goja.ArrayBuffer).Bytes()
		}
	}
	return value.Export()
}
