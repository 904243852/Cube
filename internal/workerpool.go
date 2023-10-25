package internal

import (
	"cube/internal/config"
	"github.com/dop251/goja"
)

var WorkerPool struct {
	Channels chan *Worker
	Workers  []*Worker
}

func InitWorkerPool() {
	WorkerPool.Workers = make([]*Worker, config.Count) // 创建 goja 实例池
	WorkerPool.Channels = make(chan *Worker, config.Count)

	// 编译程序
	program, _ := goja.Compile("index", "(function (id, ...params) { return require(id).default(...params); })", false) // 编译源码为 Program，strict 为 false

	for i := 0; i < config.Count; i++ {
		worker := CreateWorker(program, i) // 创建 goja 运行时

		WorkerPool.Workers[i] = worker
		WorkerPool.Channels <- worker
	}
}
