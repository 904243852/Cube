package handler

import (
	"cube/internal"
	"net/http"
	"strings"
)

func HandleResource(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/resource/")

	var content string
	if err := internal.Db.QueryRow("select content from source where url = ? and type = 'resource' and active = true", name).Scan(&content); err != nil {
		toError(w, err)
		return
	}
	toSuccess(w, content)
}
