package res

import _ "embed"

//go:embed error.html
var Error string

//go:embed index.html
var Index string

//go:embed base/head.html
var BaseHead string

//go:embed admin/index.html
var AdminIndex string

//go:embed admin/users.html
var AdminUsers string

//go:embed admin/user_create.html
var AdminUserCreate string

//go:embed admin/user_edit.html
var AdminUserEdit string

//go:embed admin/repos.html
var AdminRepos string

//go:embed admin/repo_edit.html
var AdminRepoEdit string

//go:embed user/header.html
var UserHeader string

//go:embed user/login.html
var UserLogin string

//go:embed user/sessions.html
var UserSessions string

//go:embed repo/header.html
var RepoHeader string

//go:embed repo/create.html
var RepoCreate string

//go:embed repo/log.html
var RepoLog string

//go:embed repo/tree.html
var RepoTree string

//go:embed repo/refs.html
var RepoRefs string

//go:embed style.css
var Style string
