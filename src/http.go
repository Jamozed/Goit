// http.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Jamozed/Goit/res"
)

var tmpl = template.Must(template.New("error").Parse(res.Error))

func init() {
	tmpl.Option("missingkey=zero")

	template.Must(tmpl.New("base/head").Parse(res.BaseHead))
	template.Must(tmpl.New("base/repo_header").Parse(res.RepoHeader))

	template.Must(tmpl.New("repo_index").Parse(res.RepoIndex))
	template.Must(tmpl.New("repo_create").Parse(res.RepoCreate))

	template.Must(tmpl.New("repo_log").Parse(res.RepoLog))
	template.Must(tmpl.New("repo_tree").Parse(res.RepoTree))
	template.Must(tmpl.New("repo_refs").Parse(res.RepoRefs))
}

func HttpError(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	s := fmt.Sprint(code) + " " + http.StatusText(code)
	tmpl.ExecuteTemplate(w, "error", struct{ Status string }{s})
}

func HttpNotFound(w http.ResponseWriter, r *http.Request) {
	HttpError(w, http.StatusNotFound)
}
