package module

import (
	"github.com/dop251/goja"
	"sync"
)

func init() {
	register("event", func(worker Worker, db Db) interface{} {
		return &EventClient{worker}
	})
}

type EventChannel struct {
	C      chan interface{}
	Closed bool
}

var EventBus struct {
	sync.RWMutex
	Subscribers map[string][]*EventChannel
}

type EventClient struct {
	worker Worker
}

func (e *EventClient) Emit(topic string, data interface{}) {
	EventBus.RLock()

	if chans, found := EventBus.Subscribers[topic]; found {
		go func() { // 避免阻塞发布者
			i := 0
			for _, ch := range chans {
				if !ch.Closed { // 通过位移法删除已关闭的通道
					chans[i] = ch
					i += 1
					ch.C <- data
				}
			}
		}()
	}

	EventBus.RUnlock()
}

func (e *EventClient) On(call goja.FunctionCall) goja.Value { // 见 goja.Runtime.ToValue 函数，这里需要传递 func(FunctionCall) Value 类型的方法
	topic, ok := call.Argument(0).Export().(string)
	if !ok {
		panic("invalid argument topic, not a string")
	}
	function, ok := goja.AssertFunction(call.Argument(1))
	if !ok {
		panic("invalid argument func, not a function")
	}

	ch := &EventChannel{
		C: make(chan interface{}, 1),
	}

	e.worker.AddHandle(ch)

	EventBus.Lock()

	if EventBus.Subscribers == nil {
		EventBus.Subscribers = make(map[string][]*EventChannel)
	}
	if prev, found := EventBus.Subscribers[topic]; found {
		EventBus.Subscribers[topic] = append(prev, ch)
	} else {
		EventBus.Subscribers[topic] = append([]*EventChannel{}, ch)
	}

	EventBus.Unlock()

	go func(c chan interface{}, r *goja.Runtime) {
		for data := range c {
			function(nil, r.ToValue(data))
		}
	}(ch.C, e.worker.Runtime())

	return nil
}
