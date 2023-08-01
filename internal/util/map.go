package util

func ExportMapValue(obj map[string]interface{}, name string, t string) (value interface{}, success bool) {
	if obj == nil {
		return
	}
	if o, k := obj[name]; k {
		switch t {
		case "string":
			value, success = o.(string)
		case "bool":
			value, success = o.(bool)
		case "int":
			value, success = o.(int)
		default:
			panic("type " + t + " is not supported")
		}
	}
	return
}
