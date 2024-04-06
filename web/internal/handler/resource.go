package handler

import (
	"net/http"
	"strings"

	"cube/internal"
)

func HandleResource(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/resource/")

	var content string
	if err := internal.Db.QueryRow("select content from source where url = ? and type = 'resource' and active = true", name).Scan(&content); err != nil {
		Error(w, err)
		return
	}
	Success(w, content)
}
