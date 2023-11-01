package builtin

import (
	"errors"
	"github.com/dop251/goja"
	"time"
)

func init() {
	Builtins["setTimeout"] = func(worker Worker) interface{} {
		return func(call goja.FunctionCall) goja.Value { // 此处必须返回单个 goja.Value 类型，否则将会出现异常：TypeError: 'caller', 'callee', and 'arguments' properties may not be accessed on strict mode functions or the arguments objects for calls to them at ...
			val, _ := NewTimeoutOrInterval(call, false, worker.EventLoop())
			return worker.Runtime().ToValue(val)
		}
	}
	Builtins["setInterval"] = func(worker Worker) interface{} {
		return func(call goja.FunctionCall) goja.Value {
			val, _ := NewTimeoutOrInterval(call, true, worker.EventLoop())
			return worker.Runtime().ToValue(val)
		}
	}
	Builtins["clearTimeout"] = func(worker Worker) interface{} {
		return func(t *Timeout) {
			if t != nil && !t.cancelled {
				t.timer.Stop()
				t.cancelled = true
				worker.EventLoop().count--
			}
		}
	}
	Builtins["clearInterval"] = func(worker Worker) interface{} {
		return func(i *Interval) {
			if i != nil && !i.cancelled {
				i.cancelled = true
				close(i.stop)
				worker.EventLoop().count--
			}
		}
	}
}

//#region 事件循环

type EventLoop struct {
	queue     chan func() // 任务队列
	count     int32       // 任务个数
	interrupt chan interface{}
}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		queue:     make(chan func(), 10),
		interrupt: make(chan interface{}, 1),
	}
}

func (l *EventLoop) Put(start func(add func(task func()), stop func()) (interface{}, error)) (interface{}, error) {
	val, err := start(
		func(task func()) {
			l.queue <- task
		},
		func() {
			l.count--
		},
	)
	if err == nil {
		l.count++
	}
	return val, err
}

func (l *EventLoop) Run(fn func() (goja.Value, error)) (goja.Value, error) {
	// 执行同步任务
	val, err := fn()

	// 执行任务队列中的异步任务
L:
	for l.count > 0 {
		select {
		case task := <-l.queue:
			task() // 如果需要关闭任务，需要在 task 执行期间完成任务计数器扣减操作，否则在下次循环获取队列中下一个任务时会出现阻塞
		case <-l.interrupt:
			break L
		}
	}

	// 返回同步任务的结果
	return val, err
}

func (l *EventLoop) Interrupt() {
	if len(l.interrupt) == 0 { // 这里需要防止重复发送中断信号导致过满，从而导致 RUn 方法中异步任务队列 select 的阻塞
		l.interrupt <- nil
	}
}

func (l *EventLoop) Reset() {
	if l.count > 0 {
		l.count = 0
	}
	for len(l.queue) > 0 {
		<-l.queue
	}
	for len(l.interrupt) > 0 {
		<-l.interrupt
	}
}

//#endregion

//#region 内置定时器

type Timeout struct {
	cancelled bool
	timer     *time.Timer
}

type Interval struct {
	cancelled bool
	ticker    *time.Ticker
	stop      chan struct{}
}

func NewTimeoutOrInterval(call goja.FunctionCall, repeating bool, loop *EventLoop) (interface{}, error) {
	f, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		return nil, errors.New("invalid argument callback, not a function")
	}
	d := time.Duration(call.Argument(1).ToInteger()) * time.Millisecond

	var args []goja.Value
	if len(call.Arguments) > 2 {
		args = append(args, call.Arguments[2:]...)
	}

	return loop.Put(func(add func(task func()), stop func()) (interface{}, error) {
		if repeating {
			if d <= 0 {
				d = time.Millisecond
			}
			// 创建 Interval 定时器
			i := &Interval{
				ticker: time.NewTicker(d),
				stop:   make(chan struct{}),
			}
			go func() {
			L:
				for {
					select {
					case <-i.stop:
						i.ticker.Stop()
						stop() // 任务个数计数器减一
						break L
					case <-i.ticker.C:
						add(func() { f(nil, args...) }) // 定时将回调函数加入任务队列中
					}
				}
			}()
			return i, nil
		} else {
			// 创建 Timeout 定时器
			return &Timeout{
				timer: time.AfterFunc(d, func() {
					add(func() {
						f(nil, args...)
						stop() // 一次性任务在回调函数执行结束后就移除该任务
					})
				}),
			}, nil
		}
	})
}

//#endregion
