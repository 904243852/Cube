package module

import (
	"sync"

	"cube/internal/builtin"

	"github.com/dop251/goja"
)

func init() {
	register("event", func(worker Worker, db Db) interface{} {
		return &EventClient{worker}
	})
}

//#region 事件订阅者

type EventSubscriber struct {
	trigger *builtin.EventTaskTrigger
	data    chan interface{} // 用于接收生产者（即 EventBus）发送的数据
	stop    chan struct{}    // 用于向生产者（即 EventBus）发送关闭通知
}

func (s *EventSubscriber) Next() interface{} {
	return <-s.data
}

func (s *EventSubscriber) Cancel() {
	if s.trigger.Cancel() {
		close(s.stop) // 广播通知 EventBus，不再推送数据
		close(s.data)
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
		// 虽然 go func() 可以避免阻塞发布者，但这里不使用该方式，如果需要同时发送大量的事件，这里会出现丢失
		i := 0
		for _, s := range subscribers {
			if !s.trigger.IsCancelled() {
				select {
				case <-s.stop: // 这里不能简单的直接发送数据，消费者和生产者可能位于不同的线程，closed 不是线程安全的，因此这里优先监听 stop 通道的关闭事件，如果已关闭则不发送数据
				case s.data <- data: // 发送数据
					subscribers[i] = s // 通过位移法删除已关闭的通道
					i += 1
				}
			}
		}
		b.subscribers[topic] = subscribers[:i] // 通过位移法删除已关闭的通道
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
	worker := c.worker

	s := &EventSubscriber{worker.EventLoop().NewEventTaskTrigger(), make(chan interface{}), make(chan struct{}, 1)}

	worker.AddDefer(s.Cancel)

	for _, topic := range topics {
		MyEventBus.subscribe(topic, s)
	}

	return s
}

func (c *EventClient) On(call goja.FunctionCall) goja.Value { // 见 goja.Runtime.ToValue 函数，这里需要传递 func(FunctionCall) Value 类型的方法
	// 主题
	topic, ok := call.Argument(0).Export().(string)
	if !ok {
		c.worker.Interrupt("invalid argument topic, not a string")
		return nil
	}

	// 回调方法
	fn, ok := goja.AssertFunction(call.Argument(1))
	if !ok {
		c.worker.Interrupt("invalid argument callback, not a function")
		return nil
	}

	runtime := c.worker.Runtime()

	s := c.CreateSubscriber(topic)

	go func() {
	L:
		for {
			select {
			case <-s.stop:
				break L
			case data, received := <-s.data:
				if !received {
					s.Cancel()
					break L
				}
				s.trigger.AddTask(func() {
					fn(nil, runtime.ToValue(data))
				})
			}
		}
	}()

	return runtime.ToValue(s)
}
