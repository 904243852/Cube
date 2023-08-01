package module

import (
	"sync"
	"time"
)

func init() {
	register("cache", func(worker Worker, db Db) interface{} {
		return &CacheClient{}
	})
}

var Cache sync.Map // 存放并发安全的 map

type CacheClient struct{}

func (c *CacheClient) Set(key interface{}, value interface{}, timeout int) {
	Cache.Store(key, value)
	time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		Cache.Delete(key)
	})
}

func (c *CacheClient) Get(key interface{}) interface{} {
	if value, ok := Cache.Load(key); ok {
		return value
	}
	return nil
}
