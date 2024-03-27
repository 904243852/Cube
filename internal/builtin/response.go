package builtin

import (
	"net/http"

	"cube/internal/util"
	"github.com/dop251/goja"
)

func init() {
	Builtins["ServiceResponse"] = func(worker Worker) interface{} {
		return func(call goja.ConstructorCall) *goja.Object { // 内置构造器不能同时返回 error 类型，否则将会失效
			runtime := worker.Runtime()

			output := &ServiceResponse{}

			if v, ok := call.Argument(0).Export().(int64); ok {
				output.status = int(v)
			} else {
				panic(runtime.NewTypeError("invalid status: not a number"))
			}

			if a := call.Argument(1).Export(); a != nil { // header 可以传 null
				if m, ok := a.(map[string]interface{}); ok {
					output.header = make(map[string]string, len(m))
					for k, v := range m {
						if s, ok := v.(string); !ok {
							panic(runtime.NewTypeError("invalid header " + k + ": not a string"))
						} else {
							output.header[k] = s
						}
					}
				} else {
					panic(runtime.NewTypeError("invalid headers: not a map"))
				}
			}

			if v := util.ExportGojaValue(call.Argument(2)); v != nil {
				if s, ok := v.(string); ok {
					output.data = []byte(s)
				} else if output.data, ok = v.([]byte); !ok {
					panic(runtime.NewTypeError("data should be a string or a byte array"))
				}
			}

			iv := runtime.ToValue(output).(*goja.Object)
			iv.SetPrototype(call.This.Prototype())
			return iv
		}
	}
}

type ServiceResponse struct {
	status  int
	header  map[string]string
	data    []byte
	cookies []*http.Cookie
}

func (s *ServiceResponse) SetStatus(status int) { // 设置响应状态码
	s.status = status
}

func (s *ServiceResponse) SetHeader(name string, value string) { // 设置响应消息头
	if s.header == nil {
		s.header = map[string]string{}
	}
	s.header[name] = value
}

func (s *ServiceResponse) SetData(data []byte) { // 设置响应消息体
	s.data = data
}

func (s *ServiceResponse) SetCookie(name string, value string) { // 设置 Cookie
	if s.cookies == nil {
		s.cookies = make([]*http.Cookie, 0)
	}
	cookie := &http.Cookie{
		Name:  name,
		Value: value,
	}
	s.cookies = append(s.cookies, cookie)
}

func ResponseWithServiceResponse(w http.ResponseWriter, v *ServiceResponse) {
	// header
	h := w.Header()
	for k, v := range v.header {
		h.Set(k, v)
	}

	// cookie
	for _, v := range v.cookies {
		http.SetCookie(w, v)
	}

	// status code
	w.WriteHeader(v.status) // WriteHeader 必须在 Set Header 之后调用，否则状态码将无法写入

	// data
	w.Write(v.data) // Write 必须在 WriteHeader 之后调用，否则将会抛出异常 http: superfluous response.WriteHeader call from ...
}
