package builtin

import (
	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		runtime := worker.Runtime()

		runtime.Set("WebSocket", func(call goja.ConstructorCall) *goja.Object {
			url, ok := call.Argument(0).Export().(string)
			if !ok {
				panic(runtime.NewTypeError("invalid url: not a string"))
			}

			c, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				panic(err)
			}

			iv := runtime.ToValue(NewWebSocket(c)).(*goja.Object)
			iv.SetPrototype(call.This.Prototype())
			return iv
		})
	})
}

//#region websocket

type WebSocket struct {
	connection *websocket.Conn
}

func (s *WebSocket) Read() (interface{}, error) {
	messageType, data, err := s.connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"messageType": messageType,
		"data":        Buffer(data),
	}, nil
}

func (s *WebSocket) Send(data []byte) error {
	return s.connection.WriteMessage(1, data) // message type：0 表示消息是文本格式，1 表示消息是二进制格式。这里 data 是 []byte，因此固定使用二进制格式类型
}

func (s *WebSocket) Close() {
	s.connection.Close()
}

func NewWebSocket(c *websocket.Conn) *WebSocket {
	return &WebSocket{
		connection: c,
	}
}

//#endregion
