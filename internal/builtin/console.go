package builtin

import (
	"log"
	"time"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		worker.Runtime().Set("console", &ConsoleClient{worker})
	})
}

type ConsoleClient struct {
	worker Worker
}

func (c *ConsoleClient) Log(a ...interface{}) {
	log.Println(append([]interface{}{time.Now().Format("2006-01-02 15:04:05.000"), c.worker.Id(), "Log"}, a...)...)
}

func (c *ConsoleClient) Debug(a ...interface{}) {
	log.Println(append(append([]interface{}{"\033[1;30m" + time.Now().Format("2006-01-02 15:04:05.000"), c.worker.Id(), "Debug"}, a...), "\033[m")...)
}

func (c *ConsoleClient) Info(a ...interface{}) {
	log.Println(append(append([]interface{}{"\033[0;34m" + time.Now().Format("2006-01-02 15:04:05.000"), c.worker.Id(), "Info"}, a...), "\033[m")...)
}

func (c *ConsoleClient) Warn(a ...interface{}) {
	log.Println(append(append([]interface{}{"\033[0;33m" + time.Now().Format("2006-01-02 15:04:05.000"), c.worker.Id(), "Warn"}, a...), "\033[m")...)
}

func (c *ConsoleClient) Error(a ...interface{}) {
	log.Println(append(append([]interface{}{"\033[0;31m" + time.Now().Format("2006-01-02 15:04:05.000"), c.worker.Id(), "Error"}, a...), "\033[m")...)
}
