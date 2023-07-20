// admin.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jamozed/Goit/src/util"
)

func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !authHttpAdmin(r) {
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
	}{Title: "Users"}

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

	if err := tmpl.ExecuteTemplate(w, "admin/users", data); err != nil {
		log.Println("[/admin/users]", err.Error())
	}
}

func HandleAdminUserCreate(w http.ResponseWriter, r *http.Request) {
	if !authHttpAdmin(r) {
		HttpError(w, http.StatusNotFound)
		return
	}

	data := struct{ Title, Message string }{"Create User", ""}

	if r.Method == http.MethodPost {
		username := strings.ToLower(r.FormValue("username"))
		fullname := r.FormValue("fullname")
		password := r.FormValue("password")
		isAdmin := r.FormValue("admin") == "true"

		if username == "" {
			data.Message = "Username cannot be empty"
		} else if util.SliceContains(reserved, username) {
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

	if err := tmpl.ExecuteTemplate(w, "admin/user_create", data); err != nil {
		log.Println("[/admin/user/create]", err.Error())
	}
}

func HandleAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	if !authHttpAdmin(r) {
		HttpError(w, http.StatusNotFound)
		return
	}

	id, err := strconv.ParseUint(r.URL.Query().Get("user"), 10, 64)
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
	}{Title: "Edit User ", Id: fmt.Sprint(user.Id), Name: user.Name, FullName: user.FullName, IsAdmin: user.IsAdmin}

	if r.Method == http.MethodPost {
		data.Name = strings.ToLower(r.FormValue("username"))
		data.FullName = r.FormValue("fullname")
		password := r.FormValue("password")
		data.IsAdmin = r.FormValue("admin") == "true"

		if data.Name == "" {
			data.Message = "Username cannot be empty"
		} else if util.SliceContains(reserved, data.Name) {
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

	if err := tmpl.ExecuteTemplate(w, "admin/user_edit", data); err != nil {
		log.Println("[/admin/user/edit]", err.Error())
	}
}

func HandleAdminRepos(w http.ResponseWriter, r *http.Request) {
	if !authHttpAdmin(r) {
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
	}{Title: "Repos"}

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

		data.Repos = append(data.Repos, row{
			fmt.Sprint(d.Id), user.Name, d.Name, util.If(d.IsPrivate, "private", "public"), "",
		})
	}

	if err := rows.Err(); err != nil {
		log.Println("[/admin/repos]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "admin/repos", data); err != nil {
		log.Println("[/admin/repos]", err.Error())
	}
}

func authHttpAdmin(r *http.Request) bool {
	if ok, uid := AuthHttp(r); ok {
		if user, err := GetUser(uid); err == nil && user.IsAdmin {
			return true
		}
	}

	return false
}
