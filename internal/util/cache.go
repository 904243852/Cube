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
	c.Store(key, value)
	time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		c.Delete(key)
	})
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
