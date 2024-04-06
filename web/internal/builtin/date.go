package builtin

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/dop251/goja"
)

func init() {
	Builtins = append(Builtins, func(worker Worker) {
		runtime := worker.Runtime()

		runtime.Get("Date").ToObject(runtime).Get("prototype").ToObject(runtime).Set("toString", func(call goja.FunctionCall) goja.Value {
			t, ok := call.This.Export().(time.Time)
			if !ok {
				panic(runtime.NewTypeError("Method Date.prototype.toString is called on incompatible receiver"))
			}

			layout := call.Argument(0).String()
			if layout == "undefined" {
				return runtime.ToValue(t.Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)"))
			}

			return runtime.ToValue(timeToString(t, layout))
		})

		runtime.Get("Date").ToObject(runtime).Set("toDate", func(value string, layout string) (*goja.Object, error) {
			c, _ := goja.AssertConstructor(runtime.Get("Date"))
			t, err := stringToTime(value, layout)
			if err != nil {
				return nil, err
			}
			return c(runtime.NewObject(), runtime.ToValue(t.UnixMilli()))
		})
	})
}

func timeToString(t time.Time, layout string) string {
	r, _ := regexp.Compile("y{1,4}|M{1,2}|d{1,2}|H{1,2}|m{1,2}|s{1,2}|S{1,3}")
	return r.ReplaceAllStringFunc(layout, func(s string) string {
		switch s[0] {
		case 'y':
			return fmt.Sprint(t.Year())[max(4-len(s), 0):]
		case 'M':
			return fmt.Sprintf("%02d", t.Month())[max(2-len(s), 0):]
		case 'd':
			return fmt.Sprintf("%02d", t.Day())[max(2-len(s), 0):]
		case 'H':
			return fmt.Sprintf("%02d", t.Hour())[max(2-len(s), 0):]
		case 'm':
			return fmt.Sprintf("%02d", t.Minute())[max(2-len(s), 0):]
		case 's':
			return fmt.Sprintf("%02d", t.Second())[max(2-len(s), 0):]
		case 'S':
			return fmt.Sprintf("%03d", t.Nanosecond()/1e6)[:min(len(s), 3)]
		default:
			return s
		}
	})
}

func stringToTime(value string, layout string) (*time.Time, error) {
	r, _ := regexp.Compile("y{1,4}|M{1,2}|d{1,2}|H{1,2}|m{1,2}|s{1,2}|S{1,3}")

	p, m := make([]int, 7), map[byte]byte{'y': 0, 'M': 1, 'd': 2, 'H': 3, 'm': 4, 's': 5, 'S': 6}

	idxes := r.FindAllStringIndex(layout, -1)
	for _, a := range idxes {
		v, err := strconv.Atoi(value[a[0]:a[1]])
		if err != nil {
			return nil, err
		}
		c := layout[a[0]]
		if c == 'S' {
			p[m[c]] = int(float64(v) * math.Pow10(3-a[1]+a[0]))
			continue
		}
		p[m[c]] = v
	}

	t := time.Date(p[0], time.Month(p[1]), p[2], p[3], p[4], p[5], p[6]*1e6, time.Local)
	return &t, nil
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}
