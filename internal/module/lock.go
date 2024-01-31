package module

import (
	"errors"
	"sync"
	"time"
)

func init() {
	register("lock", func(worker Worker, db Db) interface{} {
		return func(name string) *LockClient {
			LockCache.Lock()
			defer LockCache.Unlock()
			if LockCache.clients == nil {
				LockCache.clients = make(map[string]*LockClient)
			}
			client := LockCache.clients[name]
			if client == nil {
				var mutex sync.Mutex
				client = &LockClient{
					name:   &name,
					mutex:  &mutex,
					locked: new(bool),
				}
				LockCache.clients[name] = client
			}
			worker.AddDefer(func() {
				client.Unlock()
			})
			return client
		}
	})
}

var LockCache struct {
	sync.Mutex
	clients map[string]*LockClient
}

type LockClient struct {
	name   *string
	mutex  *sync.Mutex
	locked *bool
}

func (l *LockClient) tryLock() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if *l.locked == true {
		return false
	}
	*l.locked = true
	return true
}

func (l *LockClient) Lock(timeout int) error {
	for i := 0; i < timeout; i++ {
		if l.tryLock() {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	l.Unlock()
	return errors.New("acquire lock " + *l.name + " timeout")
}

func (l *LockClient) Unlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	*l.locked = false
}
