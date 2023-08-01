package internal

import (
	"bufio"
	"cube/internal/util"
	"encoding/json"
	"errors"
	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"time"
)

//#region service context

type ServiceContextReader struct {
	reader *bufio.Reader
}

func (s *ServiceContextReader) Read(count int) ([]byte, error) {
	buf := make([]byte, count)
	_, err := s.reader.Read(buf)
	if err == io.EOF {
		return nil, nil
	}
	return buf, err
}

func (s *ServiceContextReader) ReadByte() (interface{}, error) {
	b, err := s.reader.ReadByte() // 如果是 chunk 传输，该方法不会返回 chunk size 和 "\r\n"，而是按 chunk data 到达顺序依次读取每个 chunk data 中的每个字节，如果已到达的 chunk 已读完且下一个 chunk 未到达，该方法将阻塞
	if err == io.EOF {
		return -1, nil
	}
	return b, err
}

type ServiceContext struct {
	request        *http.Request
	responseWriter http.ResponseWriter
	timer          *time.Timer
	returnless     bool
	body           interface{} // 用于缓存请求消息体，防止重复读取和关闭 body 流
	vars           *map[string]string
}

func (s *ServiceContext) GetHeader() map[string]string {
	var headers = make(map[string]string)
	for name, values := range s.request.Header {
		for _, value := range values {
			headers[name] = value
		}
	}
	return headers
}

func (s *ServiceContext) GetURL() interface{} {
	u := s.request.URL

	var params = make(map[string][]string)
	for name, values := range u.Query() {
		params[name] = values
	}

	return map[string]interface{}{
		"path":   u.Path,
		"params": params,
	}
}

func (s *ServiceContext) GetBody() ([]byte, error) {
	if s.body != nil {
		return s.body.([]byte), nil
	}
	defer s.request.Body.Close()
	return io.ReadAll(s.request.Body)
}

func (s *ServiceContext) GetJsonBody() (interface{}, error) {
	data, err := s.GetBody()
	if err != nil {
		return nil, err
	}
	return s.body, json.Unmarshal(data, &s.body)
}

func (s *ServiceContext) GetMethod() string {
	return s.request.Method
}

func (s *ServiceContext) GetForm() interface{} {
	s.request.ParseForm() // 需要转换后才能获取表单

	var params = make(map[string][]string)
	for name, values := range s.request.Form {
		params[name] = values
	}

	return params
}

func (s *ServiceContext) GetPathVariables() interface{} {
	return s.vars
}

func (s *ServiceContext) GetFile(name string) (interface{}, error) {
	file, header, err := s.request.FormFile(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name": header.Filename,
		"size": header.Size,
		"data": data,
	}, nil
}

func (s *ServiceContext) GetCerts() interface{} { // 获取客户端证书
	return s.request.TLS.PeerCertificates
}

func (s *ServiceContext) UpgradeToWebSocket() (*ServiceWebSocket, error) {
	s.returnless = true // upgrader.Upgrade 内部已经调用过 WriteHeader 方法了，后续不应再次调用，否则将会出现 http: superfluous response.WriteHeader call from ... 的异常
	s.timer.Stop()      // 关闭定时器，WebSocket 不需要设置超时时间
	upgrader := websocket.Upgrader{}
	if conn, err := upgrader.Upgrade(s.responseWriter, s.request, nil); err != nil {
		return nil, err
	} else {
		return &ServiceWebSocket{
			connection: conn,
		}, nil
	}
}

func (s *ServiceContext) GetReader() *ServiceContextReader {
	return &ServiceContextReader{
		reader: bufio.NewReader(s.request.Body),
	}
}

func (s *ServiceContext) GetPusher() (http.Pusher, error) {
	pusher, ok := s.responseWriter.(http.Pusher)
	if !ok {
		return nil, errors.New("the server side push is not supported")
	}
	return pusher, nil
}

func (s *ServiceContext) Write(data []byte) (int, error) {
	return s.responseWriter.Write(data)
}

func (s *ServiceContext) Flush() error {
	flusher, ok := s.responseWriter.(http.Flusher)
	if !ok {
		return errors.New("failed to get a http flusher")
	}
	if !s.returnless {
		s.returnless = true
		s.responseWriter.Header().Set("X-Content-Type-Options", "nosniff") // https://stackoverflow.com/questions/18337630/what-is-x-content-type-options-nosniff
	}
	flusher.Flush() // 此操作将自动设置响应头 Transfer-Encoding: chunked，并发送一个 chunk
	return nil
}

func CreateServiceContext(r *http.Request, w http.ResponseWriter, t *time.Timer, v *map[string]string) *ServiceContext {
	return &ServiceContext{
		request:        r,
		responseWriter: w,
		timer:          t,
		vars:           v,
	}
}

func Returnless(ctx *ServiceContext) bool {
	return ctx.returnless
}

//#endregion

//#region service response

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

func CreateGojaServiceResponseConstructor(runtime *goja.Runtime) func(call goja.ConstructorCall) *goja.Object {
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

func ResponseWithServiceResponse(w http.ResponseWriter, v *ServiceResponse) {
	h := w.Header()
	for k, a := range v.header {
		h.Set(k, a)
	}
	w.WriteHeader(v.status) // WriteHeader 必须在 Set Header 之后调用，否则状态码将无法写入
	w.Write(v.data)         // Write 必须在 WriteHeader 之后调用，否则将会抛出异常 http: superfluous response.WriteHeader call from ...
}

//#endregion

//#region service websocket

type ServiceWebSocket struct {
	connection *websocket.Conn
}

func (s *ServiceWebSocket) Read() (interface{}, error) {
	messageType, data, err := s.connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"messageType": messageType,
		"data":        data,
	}, nil
}

func (s *ServiceWebSocket) Send(data []byte) error {
	return s.connection.WriteMessage(1, data) // message type：0 表示消息是文本格式，1 表示消息是二进制格式。这里 data 是 []byte，因此固定使用二进制格式类型
}

func (s *ServiceWebSocket) Close() {
	s.connection.Close()
}

//#endregion
