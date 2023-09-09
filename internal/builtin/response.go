package builtin

import (
	"cube/internal/util"
	"github.com/dop251/goja"
	"net/http"
)

func init() {
	Builtins["ServiceResponse"] = func(runtime *goja.Runtime) interface{} {
		return func(call goja.ConstructorCall) *goja.Object { // 内置构造器不能同时返回 error 类型，否则将会失效
			a0, ok := call.Argument(0).Export().(int64)
			if !ok {
				panic("invalid argument status, not a int")
			}
			a1, ok := call.Argument(1).Export().(map[string]interface{})
			if !ok {
				panic("invalid argument header, not a map")
			}
			header := make(map[string]string, len(a1))
			for k, v := range a1 {
				if s, ok := v.(string); !ok {
					panic("invalid argument " + k + ", not a string")
				} else {
					header[k] = s
				}
			}
			data := []byte(nil)
			if a2 := util.ExportGojaValue(call.Argument(2)); a2 != nil {
				if s, ok := a2.(string); !ok {
					if data, ok = a2.([]byte); !ok {
						panic("the data should be a string or a byte array")
					}
				} else {
					data = []byte(s)
				}
			}
			i := &ServiceResponse{
				status: int(a0),
				header: header,
				data:   data,
			}
			iv := runtime.ToValue(i).(*goja.Object)
			iv.SetPrototype(call.This.Prototype())
			return iv
		}
	}
}

type ServiceResponse struct {
	status int
	header map[string]string
	data   []byte
}

func (s *ServiceResponse) SetStatus(status int) { // 设置响应状态码
	s.status = status
}

func (s *ServiceResponse) SetHeader(header map[string]string) { // 设置响应消息头
	s.header = header
}

func (s *ServiceResponse) SetData(data []byte) { // 设置响应消息体
	s.data = data
}

func ResponseWithServiceResponse(w http.ResponseWriter, v *ServiceResponse) {
	h := w.Header()
	for k, a := range v.header {
		h.Set(k, a)
	}
	w.WriteHeader(v.status) // WriteHeader 必须在 Set Header 之后调用，否则状态码将无法写入
	w.Write(v.data)         // Write 必须在 WriteHeader 之后调用，否则将会抛出异常 http: superfluous response.WriteHeader call from ...
}
