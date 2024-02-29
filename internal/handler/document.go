package handler

import (
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

func HandleDocument(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/document/")

	fp := path.Clean("docs/" + name)
	if !strings.HasPrefix(fp+"/", "docs/") {
		http.Error(w, "permission denial", http.StatusForbidden)
		return
	}

	content, err := ioutil.ReadFile(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	toSuccess(w, string(content))
}
