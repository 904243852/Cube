package builtin

import "github.com/dop251/goja"

var Builtins = make(map[string]func(runtime *goja.Runtime) interface{})
