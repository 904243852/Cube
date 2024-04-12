package module

import (
	"cube/internal/util"
)

var cache = util.NewMemoryCache()

func init() {
	register("cache", func(worker Worker, db Db) interface{} {
		return &cache
	})
}
