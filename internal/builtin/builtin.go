package builtin

import "github.com/dop251/goja"

var Builtins = make(map[string]func(worker Worker) interface{})

type Worker interface {
	AddHandle(handle interface{})
	Runtime() *goja.Runtime
	EventLoop() *EventLoop
}
