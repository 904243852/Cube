package builtin

import (
	"github.com/dop251/goja"
	"log"
	"time"
)

func init() {
	Builtins["console"] = func(runtime *goja.Runtime) interface{} {
		return &ConsoleClient{runtime}
	}
}

type ConsoleClient struct {
	runtime *goja.Runtime
}

func (c *ConsoleClient) Log(a ...interface{}) {
	log.Println(append([]interface{}{"\r" + time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Log"}, a...)...)
}

func (c *ConsoleClient) Debug(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[1;30m" + time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Debug"}, a...), "\033[m")...)
}

func (c *ConsoleClient) Info(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[0;34m" + time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Info"}, a...), "\033[m")...)
}

func (c *ConsoleClient) Warn(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[0;33m" + time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Warn"}, a...), "\033[m")...)
}

func (c *ConsoleClient) Error(a ...interface{}) {
	log.Println(append(append([]interface{}{"\r" + "\033[0;31m" + time.Now().Format("2006-01-02 15:04:05.000"), &c.runtime, "Error"}, a...), "\033[m")...)
}
