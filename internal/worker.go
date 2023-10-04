package internal

import (
	"cube/internal/builtin"
	m "cube/internal/module"
	"database/sql"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"net"
	"path"
	"strings"
)

type Worker struct {
	runtime  *goja.Runtime
	function goja.Callable
	handles  []interface{}
	loop     *builtin.EventLoop
}

func (w *Worker) Run(params ...goja.Value) (goja.Value, error) {
	return w.loop.Run(func(vm *goja.Runtime) (goja.Value, error) {
		return w.function(nil, params...)
	})
}

func (w *Worker) Runtime() *goja.Runtime {
	return w.runtime
}

func (w *Worker) EventLoop() *builtin.EventLoop {
	return w.loop
}

func (w *Worker) AddHandle(handle interface{}) {
	w.handles = append(w.handles, handle)
}

func (w *Worker) Interrupt(reason string) {
	w.Runtime().Interrupt(reason)
	w.ClearHandle()
}

func (w *Worker) ClearHandle() {
	for _, v := range w.handles {
		if l, ok := v.(*net.Listener); ok { // 如果已存在监听端口服务，这里需要先关闭，否则将导致 goja.Runtime.Interrupt 无法关闭
			(*l).Close()
			continue
		}
		if c, ok := v.(*net.UDPConn); ok {
			(*c).Close()
			continue
		}
		if l, ok := v.(*m.LockClient); ok {
			(*l).Unlock()
			continue
		}
		if t, ok := v.(*sql.Tx); ok {
			(*t).Rollback()
			continue
		}
		if c, ok := v.(*m.EventChannel); ok {
			close((*c).C)
			(*c).Closed = true
			continue
		}
		panic(fmt.Errorf("unknown handle: %T", v))
	}
	if len(w.handles) > 0 {
		w.handles = make([]interface{}, 0) // 清空所有句柄
	}
}

func CreateWorker(program *goja.Program) *Worker {
	runtime := goja.New()

	entry, err := runtime.RunProgram(program) // 这里使用 RunProgram，可复用已编译的代码，相比直接调用 RunString 更显著提升性能
	if err != nil {
		panic(err)
	}
	function, ok := goja.AssertFunction(entry)
	if !ok {
		panic("the program is not a function")
	}

	worker := Worker{runtime, function, make([]interface{}, 0), builtin.NewEventLoop(runtime)}

	runtime.Set("require", func(id string) (goja.Value, error) {
		program := Cache.Modules[id]
		if program == nil { // 如果已被缓存，直接从缓存中获取
			// 获取名称、类型
			var name, stype string
			if strings.HasPrefix(id, "./controller/") {
				name, stype = id[13:], "controller"
			} else if strings.HasPrefix(id, "./daemon/") {
				name, stype = id[9:], "daemon"
			} else if strings.HasPrefix(id, "./crontab/") {
				name, stype = id[10:], "crontab"
			} else if strings.HasPrefix(id, "./") {
				name, stype = path.Clean(id), "module"
			} else { // 如果没有 "./" 前缀，则视为 node_modules
				name, stype = "node_modules/"+id, "module"
			}

			// 根据名称查找源码
			var src string
			if err := Db.QueryRow("select compiled from source where name = ? and type = ? and active = true", name, stype).Scan(&src); err != nil {
				return nil, err
			}
			// 编译
			parsed, err := goja.Parse(
				name,
				"(function(exports, require, module) {"+src+"\n})",
				parser.WithSourceMapLoader(func(p string) ([]byte, error) {
					return []byte(src), nil
				}),
			)
			if err != nil {
				return nil, err
			}
			program, err = goja.CompileAST(parsed, false)
			if err != nil {
				return nil, err
			}

			// 缓存当前 module 的 program
			// 这里不应该直接缓存 module，因为 module 依赖当前 vm 实例，在开启多个 vm 实例池的情况下，调用会错乱从而导致异常 "TypeError: Illegal runtime transition of an Object at ..."
			Cache.Modules[id] = program
		}

		exports := runtime.NewObject()
		module := runtime.NewObject()
		module.Set("exports", exports)

		// 运行
		entry, err := runtime.RunProgram(program)
		if err != nil {
			return nil, err
		}
		if function, ok := goja.AssertFunction(entry); ok {
			_, err = function(
				exports,                // this
				exports,                // exports
				runtime.Get("require"), // require
				module,                 // module
			)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("the entry is not a function")
		}

		return module.Get("exports"), nil
	})

	runtime.Set("exports", runtime.NewObject())

	runtime.SetFieldNameMapper(goja.UncapFieldNameMapper()) // 该转换器会将 go 对象中的属性、方法以小驼峰式命名规则映射到 js 对象中

	runtime.Set("$native", func(name string) (interface{}, error) {
		// 通过 Set 方法内置的 []byte 类型的变量或方法：
		// 入参如果是 []byte 类型，可接受 js 中 string 或 Array<number> 类型的变量
		// 出参如果是 []byte 类型，将会隐式地转换为 js 的 Array<number> 类型的变量（见 goja.objectGoArrayReflect._init() 方法实现，class 为 "Array", prototype 为 ArrayPrototype）
		factory, ok := m.Factories[name]
		if ok {
			return factory(&worker, Db), nil
		}
		return nil, errors.New("module is not found: " + name)
	})

	for name, factory := range builtin.Builtins {
		runtime.Set(name, factory(&worker))
	}

	runtime.SetMaxCallStackSize(2048)

	return &worker
}
