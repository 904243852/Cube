package module

import (
	"context"
	"database/sql"
	"github.com/dop251/goja"
)

var Factories = make(map[string]func(worker Worker, db Db) interface{})

func register(name string, factory func(worker Worker, db Db) interface{}) {
	Factories[name] = factory
}

type Worker interface {
	AddHandle(handle func())
	Runtime() *goja.Runtime
}

type Db interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
