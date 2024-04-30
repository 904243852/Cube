package module

import (
	"os/exec"

	"cube/internal/builtin"
	"github.com/dop251/goja"
)

func init() {
	register("process", func(worker Worker, db Db) interface{} {
		return &ProcessClient{worker}
	})
}

type ProcessClient struct {
	worker Worker
}

func (p *ProcessClient) Exec(command string, params ...string) (*builtin.Buffer, error) {
	output, err := exec.Command(command, params...).Output()
	if err != nil {
		return nil, err
	}
	return (*builtin.Buffer)(&output), nil
}

func (p *ProcessClient) Pexec(command string, params ...string) *goja.Promise {
	runtime := p.worker.Runtime()

	promise, resolve, reject := runtime.NewPromise()

	t := p.worker.EventLoop().NewEventTaskTrigger()

	t.AddTask(func() {
		output, err := exec.Command(command, params...).Output()
		if err != nil {
			t.AddMicroTask(func() {
				reject(runtime.NewGoError(err))
				t.Cancel()
			})
			return
		}
		t.AddMicroTask(func() { // resolve() must be called on the loop
			resolve(builtin.Buffer(output))
			t.Cancel()
		})
	})

	return promise
}
