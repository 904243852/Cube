package module

import (
	"bytes"
	"text/template"
)

func init() {
	register("template", func(worker Worker, db Db) interface{} {
		return func(name string, input map[string]interface{}) (string, error) {
			var content string
			if err := db.QueryRow("select content from source where name = ? and type = 'template' and active = true", name).Scan(&content); err != nil {
				return "", err
			}

			if t, err := template.New(name).Parse(content); err != nil {
				return "", err
			} else {
				buf := new(bytes.Buffer)
				t.Execute(buf, input)
				return buf.String(), nil
			}
		}
	})
}
