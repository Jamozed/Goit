package res

import _ "embed"

//go:embed repo_index.html
var RepoIndex string

//go:embed user_login.html
var UserLogin string

//go:embed repo_create.html
var RepoCreate string

//go:embed repo_log.html
var RepoLog string

//go:embed user_create.html
var UserCreate string

//go:embed admin_user_index.html
var AdminUserIndex string

//go:embed style.css
var Style string
