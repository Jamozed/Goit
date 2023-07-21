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
	template.Must(tmpl.New("index").Parse(res.Index))
	template.Must(tmpl.New("base/head").Parse(res.BaseHead))
	template.Must(tmpl.New("base/header").Parse(res.BaseHeader))

	template.Must(tmpl.New("admin/users").Parse(res.AdminUsers))
	template.Must(tmpl.New("admin/user/create").Parse(res.AdminUserCreate))
	template.Must(tmpl.New("admin/user/edit").Parse(res.AdminUserEdit))
	template.Must(tmpl.New("admin/repos").Parse(res.AdminRepos))
	template.Must(tmpl.New("admin/repo/edit").Parse(res.AdminRepoEdit))

	template.Must(tmpl.New("base/repo_header").Parse(res.RepoHeader))
	template.Must(tmpl.New("user_login").Parse(res.UserLogin))

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
