package builtin

import "github.com/dop251/goja"

var Builtins = make([]func(worker Worker), 0)

type Worker interface {
	AddDefer(d func())
	Id() int
	Runtime() *goja.Runtime
	EventLoop() *EventLoop
}
