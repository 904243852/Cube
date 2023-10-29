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

//#region 事件订阅者

type EventSubscriber struct {
	ch     chan interface{}
	stop   chan interface{}
	closed bool
}

func (s *EventSubscriber) Next() interface{} {
	return <-s.ch
}

func (s *EventSubscriber) Cancel() {
	if !s.closed {
		s.stop <- nil
	}
}

//#endregion

//#region 事件总线

var MyEventBus EventBus

type EventBus struct {
	sync.RWMutex
	subscribers map[string][]*EventSubscriber
}

func (b *EventBus) subscribe(topic string, s *EventSubscriber) {
	b.RLock()

	if b.subscribers == nil {
		b.subscribers = make(map[string][]*EventSubscriber)
	}
	if subscribers, found := b.subscribers[topic]; found {
		b.subscribers[topic] = append(subscribers, s)
	} else {
		b.subscribers[topic] = append([]*EventSubscriber{}, s)
	}

	b.RUnlock()
}

func (b *EventBus) emit(topic string, data interface{}) {
	b.RLock()

	if subscribers, found := b.subscribers[topic]; found {
		go func() { // 避免阻塞发布者
			i := 0
			for _, s := range subscribers {
				if !s.closed { // 通过位移法删除已关闭的通道
					subscribers[i] = s
					i += 1
					s.ch <- data
				}
			}
		}()
	}

	b.RUnlock()
}

//#endregion

type EventClient struct {
	worker Worker
}

func (c *EventClient) Emit(topic string, data interface{}) {
	MyEventBus.emit(topic, data)
}

func (c *EventClient) CreateSubscriber(topics ...string) *EventSubscriber {
	s := &EventSubscriber{
		ch:   make(chan interface{}),
		stop: make(chan interface{}),
	}
	c.worker.AddDefer(func() {
		if !s.closed {
			close(s.stop)
			s.closed = true
			close(s.ch)
		}
	})
	for _, topic := range topics {
		MyEventBus.subscribe(topic, s)
	}
	return s
}

func (c *EventClient) On(call goja.FunctionCall) goja.Value { // 见 goja.Runtime.ToValue 函数，这里需要传递 func(FunctionCall) Value 类型的方法
	topic, ok := call.Argument(0).Export().(string)
	if !ok {
		c.worker.Interrupt("invalid argument topic, not a string")
		return nil
	}
	fn, ok := goja.AssertFunction(call.Argument(1))
	if !ok {
		c.worker.Interrupt("invalid argument callback, not a function")
		return nil
	}

	runtime := c.worker.Runtime()

	val, _ := c.worker.EventLoop().Put(func(add func(task func()), stop func()) (interface{}, error) {
		s := c.CreateSubscriber(topic)
		go func() {
		L:
			for {
				select {
				case <-s.stop:
					s.closed = true
					stop()
					break L
				case data := <-s.ch:
					add(func() {
						fn(nil, runtime.ToValue(data))
					})
				}
			}
		}()
		return s, nil
	})
	return runtime.ToValue(val)
}
