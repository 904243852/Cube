package module

import (
	"errors"
	"sync"
	"time"
)

func init() {
	register("bqueue", func(worker Worker, db Db) interface{} {
		return func(size int) *BlockingQueueClient {
			return &BlockingQueueClient{
				queue: make(chan interface{}, size),
			}
		}
	})
}

type BlockingQueueClient struct {
	queue chan interface{}
	sync.Mutex
}

func (b *BlockingQueueClient) Put(input interface{}, timeout int) error {
	b.Lock()
	defer b.Unlock()
	select {
	case b.queue <- input:
		return nil
	case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列入列最大超时时间为 timeout 毫秒
		return errors.New("blocking queue is full, waiting for put timeout")
	}
}

func (b *BlockingQueueClient) Poll(timeout int) (interface{}, error) {
	b.Lock()
	defer b.Unlock()
	select {
	case output := <-b.queue:
		return output, nil
	case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列出列最大超时时间为 timeout 毫秒
		return nil, errors.New("blocking queue is empty, waiting for poll timeout")
	}
}

func (b *BlockingQueueClient) Drain(size int, timeout int) (output []interface{}) {
	b.Lock()
	defer b.Unlock()
	output = make([]interface{}, 0, size) // 创建切片，初始大小为 0，最大为 size
	c := make(chan int, 1)
	go func(ch chan int) {
		for i := 0; i < size; i++ {
			output = append(output, <-b.queue)
		}
		ch <- 0
	}(c)
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	select {
	case <-c:
	case <-timer.C: // 定时器也是一个通道
	}
	return
}
