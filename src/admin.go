// admin.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
)

func HandleAdminIndex(w http.ResponseWriter, r *http.Request) {
	if _, admin, _ := AuthCookieAdmin(w, r, true); !admin {
		HttpError(w, http.StatusNotFound)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "admin/index", struct{ Title string }{"Admin"}); err != nil {
		log.Println("[/admin/index]", err.Error())
	}
}

func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if _, admin, _ := AuthCookieAdmin(w, r, true); !admin {
		HttpError(w, http.StatusNotFound)
		return
	}

	rows, err := db.Query("SELECT id, name, name_full, is_admin FROM users")
	if err != nil {
		log.Println("[/admin/users]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type row struct{ Id, Name, FullName, IsAdmin string }
	data := struct {
		Title string
		Users []row
	}{Title: "Admin - Users"}

	for rows.Next() {
		d := User{}
		if err := rows.Scan(&d.Id, &d.Name, &d.FullName, &d.IsAdmin); err != nil {
			log.Println("[/admin/users]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		}

		data.Users = append(data.Users, row{
			fmt.Sprint(d.Id), d.Name, d.FullName, util.If(d.IsAdmin, "true", "false"),
		})
	}

	if err := rows.Err(); err != nil {
		log.Println("[/admin/users]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "admin/users", data); err != nil {
		log.Println("[/admin/users]", err.Error())
	}
}

func HandleAdminUserCreate(w http.ResponseWriter, r *http.Request) {
	if _, admin, _ := AuthCookieAdmin(w, r, true); !admin {
		HttpError(w, http.StatusNotFound)
		return
	}

	data := struct{ Title, Message string }{"Admin - Create User", ""}

	if r.Method == http.MethodPost {
		username := strings.ToLower(r.FormValue("username"))
		fullname := r.FormValue("fullname")
		password := r.FormValue("password")
		isAdmin := r.FormValue("admin") == "true"

		if username == "" {
			data.Message = "Username cannot be empty"
		} else if slices.Contains(reserved, username) {
			data.Message = "Username \"" + username + "\" is reserved"
		} else if exists, err := UserExists(username); err != nil {
			log.Println("[/admin/user/create]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if exists {
			data.Message = "Username \"" + username + "\" is taken"
		} else if salt, err := Salt(); err != nil {
			log.Println("[/admin/user/create]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if _, err := db.Exec(
			"INSERT INTO users (name, name_full, pass, pass_algo, salt, is_admin) VALUES (?, ?, ?, ?, ?, ?)",
			username, fullname, Hash(password, salt), "argon2", salt, isAdmin,
		); err != nil {
			log.Println("[/admin/user/create]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else {
			data.Message = "User \"" + username + "\" created successfully"
		}
	}

	if err := Tmpl.ExecuteTemplate(w, "admin/user/create", data); err != nil {
		log.Println("[/admin/user/create]", err.Error())
	}
}

func HandleAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	if _, admin, _ := AuthCookieAdmin(w, r, true); !admin {
		HttpError(w, http.StatusNotFound)
		return
	}

	id, err := strconv.ParseInt(r.URL.Query().Get("user"), 10, 64)
	if err != nil {
		HttpError(w, http.StatusNotFound)
		return
	}

	user, err := GetUser(id)
	if err != nil {
		log.Println("[/admin/user/edit]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if user == nil {
		HttpError(w, http.StatusNotFound)
		return
	}

	data := struct {
		Title, Id, Name, FullName, Message string
		IsAdmin                            bool
	}{
		Title: "Admin - Edit User", Id: fmt.Sprint(user.Id), Name: user.Name, FullName: user.FullName,
		IsAdmin: user.IsAdmin,
	}

	if r.Method == http.MethodPost {
		data.Name = strings.ToLower(r.FormValue("username"))
		data.FullName = r.FormValue("fullname")
		password := r.FormValue("password")
		data.IsAdmin = r.FormValue("admin") == "true"

		if data.Name == "" {
			data.Message = "Username cannot be empty"
		} else if slices.Contains(reserved, data.Name) {
			data.Message = "Username \"" + data.Name + "\" is reserved"
		} else if exists, err := UserExists(data.Name); err != nil {
			log.Println("[/admin/user/edit]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Name != user.Name {
			data.Message = "Username \"" + data.Name + "\" is taken"
		} else if salt, err := Salt(); err != nil {
			log.Println("[/admin/user/edit]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else {
			if password == "" {
				_, err = db.Exec(
					"UPDATE users SET name = ?, name_full = ?, is_admin = ? WHERE id = ?",
					data.Name, data.FullName, data.IsAdmin, user.Id,
				)
			} else {
				_, err = db.Exec(
					"UPDATE users SET name = ?, name_full = ?, pass = ?, salt = ?, is_admin = ? WHERE id = ?",
					data.Name, data.FullName, Hash(password, salt), salt, data.IsAdmin, user.Id,
				)
			}

			if err != nil {
				log.Println("[/admin/user/edit]", err.Error())
				HttpError(w, http.StatusInternalServerError)
				return
			} else {
				data.Message = "User \"" + user.Name + "\" updated successfully"
			}
		}
	}

	if err := Tmpl.ExecuteTemplate(w, "admin/user/edit", data); err != nil {
		log.Println("[/admin/user/edit]", err.Error())
	}
}

func HandleAdminRepos(w http.ResponseWriter, r *http.Request) {
	if _, admin, _ := AuthCookieAdmin(w, r, true); !admin {
		HttpError(w, http.StatusNotFound)
		return
	}

	rows, err := db.Query("SELECT id, owner_id, name, is_private FROM repos")
	if err != nil {
		log.Println("[/admin/repos]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type row struct{ Id, Owner, Name, Visibility, Size string }
	data := struct {
		Title string
		Repos []row
	}{Title: "Admin - Repositories"}

	for rows.Next() {
		d := Repo{}
		if err := rows.Scan(&d.Id, &d.OwnerId, &d.Name, &d.IsPrivate); err != nil {
			log.Println("[/admin/repos]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		}

		user, err := GetUser(d.OwnerId)
		if err != nil {
			log.Println("[/admin/repos]", err.Error())
		}

		size, err := util.DirSize(RepoPath(d.Name))
		if err != nil {
			log.Println("[/admin/repos]", err.Error())
		}

		data.Repos = append(data.Repos, row{
			fmt.Sprint(d.Id), user.Name, d.Name, util.If(d.IsPrivate, "private", "public"), humanize.IBytes(size),
		})
	}

	if err := rows.Err(); err != nil {
		log.Println("[/admin/repos]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "admin/repos", data); err != nil {
		log.Println("[/admin/repos]", err.Error())
	}
}

func HandleAdminRepoEdit(w http.ResponseWriter, r *http.Request) {
	if _, admin, _ := AuthCookieAdmin(w, r, true); !admin {
		HttpError(w, http.StatusNotFound)
		return
	}

	id, err := strconv.ParseInt(r.URL.Query().Get("repo"), 10, 64)
	if err != nil {
		HttpError(w, http.StatusNotFound)
		return
	}

	repo, err := GetRepo(id)
	if err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil {
		HttpError(w, http.StatusNotFound)
		return
	}

	data := struct {
		Title, Id, Owner, Name, Description, Message string
		IsPrivate                                    bool
	}{
		Title: "Admin - Edit Repository", Id: fmt.Sprint(repo.Id), Name: repo.Name, Description: repo.Description,
		IsPrivate: repo.IsPrivate,
	}

	owner, err := GetUser(repo.OwnerId)
	if err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
		data.Owner = fmt.Sprint(repo.OwnerId)
	} else {
		data.Owner = owner.Name
	}

	if r.Method == http.MethodPost {
		data.Name = r.FormValue("reponame")
		data.Description = r.FormValue("description")
		data.IsPrivate = r.FormValue("visibility") == "private"

		if data.Name == "" {
			data.Message = "Name cannot be empty"
		} else if slices.Contains(reserved, data.Name) {
			data.Message = "Name \"" + data.Name + "\" is reserved"
		} else if exists, err := RepoExists(data.Name); err != nil {
			log.Println("[/admin/repo/edit]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Name != repo.Name {
			data.Message = "Name \"" + data.Name + "\" is taken"
		} else if _, err := db.Exec(
			"UPDATE repos SET name = ?, name_lower = ?, description = ?, is_private = ? WHERE id = ?",
			data.Name, strings.ToLower(data.Name), data.Description, data.IsPrivate, repo.Id,
		); err != nil {
			log.Println("[/admin/repo/edit]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else {
			data.Message = "Repository \"" + repo.Name + "\" updated successfully"
		}
	}

	if err := Tmpl.ExecuteTemplate(w, "admin/repo/edit", data); err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
	}
}
