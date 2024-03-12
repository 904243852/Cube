package internal

import (
	"bufio"
	"cube/internal/builtin"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"time"
)

//#region service context

type ServiceContextReader struct {
	reader *bufio.Reader
}

func (s *ServiceContextReader) Read(size int) (builtin.Buffer, error) {
	buf := make([]byte, size)
	_, err := s.reader.Read(buf)
	if err == io.EOF {
		return nil, nil
	}
	return buf, err
}

func (s *ServiceContextReader) ReadByte() (byte, error) {
	b, err := s.reader.ReadByte() // 如果是 chunk 传输，该方法不会返回 chunk size 和 "\r\n"，而是按 chunk data 到达顺序依次读取每个 chunk data 中的每个字节，如果已到达的 chunk 已读完且下一个 chunk 未到达，该方法将阻塞
	if err == io.EOF {
		return 0, nil
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

func (s *ServiceContext) GetBody() (builtin.Buffer, error) { // 返回类型如果是 []byte 则对应 goja 中 Array<number> 类型，这里转成 builtin.Buffer 类型以方便使用 toString、toJson 拓展方法
	if s.body != nil {
		return s.body.([]byte), nil
	}
	defer s.request.Body.Close()
	return io.ReadAll(s.request.Body)
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
		"data": builtin.Buffer(data),
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
		return nil, errors.New("server side push is not supported")
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

func (s *ServiceContext) ResetTimeout(timeout int) {
	// For a Timer created with NewTimer, Reset should be invoked only on stopped or expired timers with drained channels.
	if !s.timer.Stop() {
		select {
		case <-s.timer.C: // try to drain the channel
		default:
		}
	}
	if timeout > 0 {
		_ = s.timer.Reset(time.Duration(timeout) * time.Millisecond) // Reset 的返回值：true 表示定时器未超时，false 表示定时器已经停止或超时
	}
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
		"data":        builtin.Buffer(data),
	}, nil
}

func (s *ServiceWebSocket) Send(data []byte) error {
	return s.connection.WriteMessage(1, data) // message type：0 表示消息是文本格式，1 表示消息是二进制格式。这里 data 是 []byte，因此固定使用二进制格式类型
}

func (s *ServiceWebSocket) Close() {
	s.connection.Close()
}

//#endregion
