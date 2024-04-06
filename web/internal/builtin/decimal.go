package builtin

import (
	"github.com/dop251/goja"
	"github.com/shopspring/decimal"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		runtime := worker.Runtime()

		runtime.Set("Decimal", func(call goja.ConstructorCall) *goja.Object {
			v, ok := call.Argument(0).Export().(string)
			if !ok {
				panic(runtime.NewTypeError("value is required"))
			}

			d, err := decimal.NewFromString(v)
			if err != nil {
				panic(err)
			}

			return runtime.ToValue(&d).(*goja.Object)
		})
	})
}
