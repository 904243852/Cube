package module

import "github.com/shopspring/decimal"

func init() {
	register("decimal", func(worker Worker, db Db) interface{} {
		return func(value string) (decimal.Decimal, error) {
			return decimal.NewFromString(value)
		}
	})
}
