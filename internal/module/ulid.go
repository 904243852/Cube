package module

import (
	"crypto/rand"
	"sync"
	"time"
)

func init() {
	register("ulid", func(worker Worker, db Db) interface{} {
		return CreateULID
	})
}

var ULIDCache struct {
	sync.Mutex
	timestamp  *int64
	randomness *[16]byte
	num        *uint64
}

func CreateULID() string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond) // 时间戳，精确到毫秒

	var randomness [16]byte
	var num uint64

	ULIDCache.Lock()
	defer ULIDCache.Unlock()

	if ULIDCache.timestamp != nil && *ULIDCache.timestamp == timestamp {
		randomness = *ULIDCache.randomness
		num = *ULIDCache.num + 1
		ULIDCache.num = &num
	} else {
		rand.Read(randomness[:])
		for i := 8; i < 16; i++ { // 后 8 个字节转数字
			num |= uint64(randomness[i]) << (56 - (i-8)*8)
		}
		ULIDCache.timestamp = &timestamp
		ULIDCache.randomness = &randomness
		ULIDCache.num = &num
	}

	var buf [26]byte

	alphabet := "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for i := 0; i < 10; i++ { // 前 10 个字符为时间戳
		buf[i] = alphabet[timestamp>>(45-i*5)&0b11111]
	}
	for i := 10; i < 18; i++ { // 中 8 个字符为随机数
		buf[i] = alphabet[randomness[i-10]&0b11111]
	}
	for i := 18; i < 26; i++ { // 后 8 个字符为递增随机数
		buf[i] = alphabet[num>>(56-(i-18)*8)&0b11111]
	}

	return string(buf[:])
}
