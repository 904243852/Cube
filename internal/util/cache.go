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
	timers map[interface{}]*time.Timer
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		timers: make(map[interface{}]*time.Timer),
	}
}

func (c *MemoryCache) Set(key interface{}, value interface{}, timeout int) {
	if value == nil || timeout <= 0 { // 如果值为 nil 或失效时间小于等于 0，则立即删除该缓存
		// 删除缓存
		c.Delete(key)
		// 清理定时器
		if t, ok := c.timers[key]; ok {
			t.Stop()
			delete(c.timers, key)
		}
		return
	}

	// 设置缓存
	c.Store(key, value)
	// 设置失效时间，单位毫秒
	c.timers[key] = time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		c.Delete(key)
		delete(c.timers, key)
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

func (c *MemoryCache) Expire(key interface{}, timeout int) {
	// 查询旧定时器，如果存在则停止并删除
	if t, ok := c.timers[key]; ok {
		t.Stop()
		delete(c.timers, key)
	} else {
		return // 定时器不存在
	}

	// 如果新的失效时间小于等于 0，则立即删除缓存
	if timeout <= 0 {
		c.Delete(key)
		return
	}

	// 设置新的失效时间
	c.timers[key] = time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		c.Delete(key)
		delete(c.timers, key)
	})
}
