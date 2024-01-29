package handler

import (
	"cube/internal"
	"cube/internal/util"
	"net/http"
	"strings"
	"time"
)

func HandleService(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/service/")

	// 查询 controller
	name, vars := internal.Cache.GetRoute(path)
	if name == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	source := internal.Cache.GetController(name)
	if source.Method != "" && source.Method != r.Method { // 校验请求方法
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取 vm 实例
	var worker *internal.Worker
	select {
	case worker = <-internal.WorkerPool.Channels:
	default:
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		worker.Reset()
		internal.WorkerPool.Channels <- worker
	}()

	// 允许最大执行的时间为 60 秒
	timer := time.AfterFunc(60*time.Second, func() {
		worker.Interrupt("service executed timeout")
	})
	defer timer.Stop()

	// 脚本执行完成标记
	completed := false
	// 监听客户端是否主动取消请求
	go func() {
		<-r.Context().Done() // 客户端主动取消
		if !completed {      // 如果脚本已执行结束，不再中断 goja 运行时，否则中断信号无法被触发和清除（需要 goja 运行时执行指令栈才会触发中断操作），导致回收再复用时直接抛出 "Client cancelled." 的异常
			worker.Interrupt("client cancelled")
		}
	}()

	ctx := internal.CreateServiceContext(r, w, timer, &vars)

	// 执行
	value, err := worker.Run(
		worker.Runtime().ToValue("./controller/"+source.Name),
		worker.Runtime().ToValue(ctx),
	)
	// 标记脚本执行完成
	completed = true

	if internal.Returnless(ctx) == true { // 如果是 WebSocket 或 chunk 响应，不需要封装响应
		if err != nil {
			internal.LogWithError(err, worker)
		}
		return
	}

	if err != nil {
		toError(w, err) // 如果 returnless 为 true，则可能已经执行了 response.Write，此时不能调用 toError 或 toSuccess（该方法会间接调用 WriteHeader），由于 Write 必须在 WriteHeader 之后调用，从而导致异常 http: superfluous response.WriteHeader call from ...
		return
	}

	toSuccess(w, util.ExportGojaValue(value))
}
