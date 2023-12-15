// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Jamozed/Goit/res"
)

var Tmpl = template.Must(template.New("error").Parse(res.Error))

func init() {
	template.Must(Tmpl.New("index").Parse(res.Index))
	template.Must(Tmpl.New("base/head").Parse(res.BaseHead))

	template.Must(Tmpl.New("admin/header").Parse(res.AdminHeader))
	template.Must(Tmpl.New("admin/index").Parse(res.AdminIndex))
	template.Must(Tmpl.New("admin/users").Parse(res.AdminUsers))
	template.Must(Tmpl.New("admin/user/create").Parse(res.AdminUserCreate))
	template.Must(Tmpl.New("admin/user/edit").Parse(res.AdminUserEdit))
	template.Must(Tmpl.New("admin/repos").Parse(res.AdminRepos))
	template.Must(Tmpl.New("admin/repo/edit").Parse(res.AdminRepoEdit))

	template.Must(Tmpl.New("user/header").Parse(res.UserHeader))
	template.Must(Tmpl.New("user/login").Parse(res.UserLogin))
	template.Must(Tmpl.New("user/sessions").Parse(res.UserSessions))
	template.Must(Tmpl.New("user/edit").Parse(res.UserEdit))

	template.Must(Tmpl.New("repo/header").Parse(res.RepoHeader))
	template.Must(Tmpl.New("repo/create").Parse(res.RepoCreate))
	template.Must(Tmpl.New("repo/import").Parse(res.RepoImport))
	template.Must(Tmpl.New("repo/edit").Parse(res.RepoEdit))

	template.Must(Tmpl.New("repo/log").Parse(res.RepoLog))
	template.Must(Tmpl.New("repo/commit").Parse(res.RepoCommit))
	template.Must(Tmpl.New("repo/tree").Parse(res.RepoTree))
	template.Must(Tmpl.New("repo/file").Parse(res.RepoFile))
	template.Must(Tmpl.New("repo/refs").Parse(res.RepoRefs))
}

func HttpError(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	s := fmt.Sprint(code) + " " + http.StatusText(code)
	Tmpl.ExecuteTemplate(w, "error", struct{ Status string }{s})
}

func HttpNotFound(w http.ResponseWriter, r *http.Request) {
	HttpError(w, http.StatusNotFound)
}
