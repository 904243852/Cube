package util

import (
	"sync"
	"time"
)

type Cache interface {
	Get(key interface{}) interface{}
	Set(key interface{}, value interface{}, timeout int)
}

type MemoryCache struct {
	sync.Map
}

func (c *MemoryCache) Set(key interface{}, value interface{}, timeout int) {
	if value == nil { // 如果值为 nil，表示删除该缓存
		c.Delete(key)
		return
	}

	c.Store(key, value)

	if timeout > 0 { // 如果 timeout 大于 0，缓存超过该毫秒将会自动被清除
		time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
			c.Delete(key)
		})
	}
}

func (c *MemoryCache) Get(key interface{}) interface{} {
	if value, ok := c.Load(key); ok {
		return value
	}
	return nil
}

func (c *MemoryCache) Has(key interface{}) bool {
	_, ok := c.Load(key)
	return ok
}
