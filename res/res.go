package res

import _ "embed"

//go:embed error.html.tmpl
var Error string

//go:embed base/head.html.tmpl
var BaseHead string

//go:embed base/repo_header.html.tmpl
var RepoHeader string

//go:embed repo_index.html
var RepoIndex string

//go:embed user_login.html
var UserLogin string

//go:embed repo_create.html
var RepoCreate string

//go:embed repo_log.html
var RepoLog string

//go:embed repo_tree.html
var RepoTree string

//go:embed repo_refs.html
var RepoRefs string

//go:embed user_create.html
var UserCreate string

//go:embed admin_user_index.html
var AdminUserIndex string

//go:embed style.css
var Style string
