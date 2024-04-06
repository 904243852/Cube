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

	// 编译源码
	program, _ := goja.Compile(
		"index",
		"(function (id, ...params) { return require(id).default(...params); })", // 使用闭包，防止全局变量污染
		false, // 关闭严格模式，增加运行时的容错能力
	)

	for i := 0; i < config.Count; i++ {
		worker := CreateWorker(program, i) // 创建 goja 运行时

		WorkerPool.Workers[i] = worker
		WorkerPool.Channels <- worker
	}
}
