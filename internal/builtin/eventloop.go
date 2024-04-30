package builtin

import (
	"errors"
	"time"

	"github.com/dop251/goja"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		runtime, loop := worker.Runtime(), worker.EventLoop()

		runtime.Set("setTimeout", func(call goja.FunctionCall) goja.Value { // 此处必须返回单个 goja.Value 类型，否则将会出现异常：TypeError: 'caller', 'callee', and 'arguments' properties may not be accessed on strict mode functions or the arguments objects for calls to them at ...
			value, _ := loop.NewTimeoutOrInterval(call, false)
			return runtime.ToValue(value)
		})
		runtime.Set("clearTimeout", func(t *Timeout) {
			if t != nil && t.trigger.Cancel() {
				t.timer.Stop()
			}
		})

		runtime.Set("setInterval", func(call goja.FunctionCall) goja.Value {
			value, _ := loop.NewTimeoutOrInterval(call, true)
			return runtime.ToValue(value)
		})
		runtime.Set("clearInterval", func(i *Interval) {
			if i != nil && i.trigger.Cancel() {
				close(i.stop)
			}
		})
	})
}

//#region 事件循环

type EventLoop struct {
	tasks      chan func()      // 宏任务队列，如 setTimeout、setInterval、Promise 中的主方法
	microtasks chan func()      // 微任务队列，如 Promise 中的 resolve 和 reject
	count      int              // 计数器
	interrupt  chan interface{} // 中断信号，用于中断事件循环
}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		tasks:      make(chan func(), 10),
		microtasks: make(chan func(), 10),
		interrupt:  make(chan interface{}, 1),
	}
}

func (l *EventLoop) Run(main func() (goja.Value, error)) (goja.Value, error) {
	// 执行主线程上的同步任务
	value, err := main()

	// 执行任务队列中的异步任务
L:
	for l.count > 0 {
		select {
		case <-l.interrupt:
			break L
		case microtask := <-l.microtasks: // 优先执行所有的微任务
			microtask()
		case task := <-l.tasks:
			task()
		}
	}

	// 返回主线程上的同步任务的结果
	return value, err
}

func (l *EventLoop) Interrupt() {
	if len(l.interrupt) == 0 { // 这里需要防止重复发送中断信号导致过满，从而导致 Run 方法中异步任务队列 select 的阻塞
		l.interrupt <- nil
	}
}

func (l *EventLoop) Reset() {
	l.count = 0
	for len(l.tasks) > 0 {
		<-l.tasks
	}
	for len(l.microtasks) > 0 {
		<-l.microtasks
	}
	for len(l.interrupt) > 0 {
		<-l.interrupt
	}
}

//#endregion

//#region 触发器、定时器

type EventTaskTrigger struct {
	cancelled bool
	loop      *EventLoop
}

func (t *EventTaskTrigger) AddTask(fn func()) {
	t.loop.tasks <- fn
}

func (t *EventTaskTrigger) AddMicroTask(fn func()) {
	t.loop.microtasks <- fn
}

func (t *EventTaskTrigger) IsCancelled() bool {
	return t.cancelled
}

func (t *EventTaskTrigger) Cancel() bool {
	if t.cancelled {
		return false
	}
	t.cancelled = true
	t.loop.count--
	return true
}

func (l *EventLoop) NewEventTaskTrigger() *EventTaskTrigger {
	l.count++
	return &EventTaskTrigger{
		loop: l,
	}
}

type Timeout struct {
	trigger *EventTaskTrigger
	timer   *time.Timer
}

type Interval struct {
	trigger *EventTaskTrigger
	ticker  *time.Ticker
	stop    chan struct{}
}

func (l *EventLoop) NewTimeoutOrInterval(call goja.FunctionCall, isInterval bool) (interface{}, error) {
	// 定时器到期后将要执行的方法
	fn, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		return nil, errors.New("invalid argument callback, not a function")
	}

	// 定时器在执行指定的方法前等待的时间，单位毫秒，默认为 0
	delay := time.Duration(call.Argument(1).ToInteger()) * time.Millisecond

	// 定时器到期后执行的方法的附加参数
	var params []goja.Value
	if len(call.Arguments) > 2 {
		params = append(params, call.Arguments[2:]...)
	}

	trigger := l.NewEventTaskTrigger()

	if isInterval {
		if delay <= 0 {
			delay = time.Millisecond
		}

		// 创建 Interval 定时器
		i := &Interval{trigger, time.NewTicker(delay), make(chan struct{}, 1)}
		// 开启定时器
		go func() {
		L:
			for {
				select {
				case <-i.stop:
					i.ticker.Stop() // ticker 的 Stop() 方法不会关闭通道 ticker.C，因此这里需要一个自定义通道 stop 以退出循环
					break L
				case <-i.ticker.C:
					// 定时将回调函数加入宏任务队列中
					if !trigger.IsCancelled() {
						trigger.AddTask(func() {
							fn(nil, params...)
						})
					}
				}
			}
		}()
		return i, nil
	}

	// 创建 Timeout 定时器
	return &Timeout{
		trigger,
		time.AfterFunc(delay, func() {
			trigger.AddTask(func() {
				if trigger.Cancel() {
					fn(nil, params...)
				}
			})
		}),
	}, nil
}

//#endregion
