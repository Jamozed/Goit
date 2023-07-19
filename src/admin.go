// admin.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/Jamozed/Goit/res"
)

var (
	adminUserIndex *template.Template
)

func init() {
	adminUserIndex = template.Must(template.New("admin_user_index").Parse(res.AdminUserIndex))
}

func HandleAdminUserIndex(w http.ResponseWriter, r *http.Request) {
	if ok, uid := AuthHttp(r); !ok {
		HttpError(w, http.StatusNotFound)
		return
	} else if user, err := GetUser(uid); err != nil {
		log.Println("[Admin:User:Create:Auth]", err.Error())
		HttpError(w, http.StatusNotFound)
		return
	} else if !user.IsAdmin {
		HttpError(w, http.StatusNotFound)
		return
	}

	if rows, err := db.Query("SELECT id, name, name_full, is_admin FROM users"); err != nil {
		log.Println("[Admin:User:Index:SELECT]", err.Error())
		HttpError(w, http.StatusInternalServerError)
	} else {
		defer rows.Close()

		type row struct{ Id, Name, FullName, IsAdmin string }
		users := []row{}

		for rows.Next() {
			u := User{}

			if err := rows.Scan(&u.Id, &u.Name, &u.NameFull, &u.IsAdmin); err != nil {
				log.Println("[Admin:User:Index:SELECT:Scan]", err.Error())
			} else {
				users = append(users, row{fmt.Sprint(u.Id), u.Name, u.NameFull, If(u.IsAdmin, "true", "false")})
			}
		}

		if err := rows.Err(); err != nil {
			log.Println("[Admin:User:Index:SELECT:Err]", err.Error())
			HttpError(w, http.StatusInternalServerError)
		} else {
			adminUserIndex.Execute(w, struct{ Users []row }{users})
		}
	}
}

func HandleAdminUserCreate(w http.ResponseWriter, r *http.Request) {
	if ok, uid := AuthHttp(r); !ok {
		HttpError(w, http.StatusNotFound)
		return
	} else if user, err := GetUser(uid); err != nil {
		log.Println("[Admin:User:Create:Auth]", err.Error())
		HttpError(w, http.StatusNotFound)
		return
	} else if !user.IsAdmin {
		HttpError(w, http.StatusNotFound)
		return
	}

	data := struct{ Msg string }{""}

	if r.Method == http.MethodPost {
		username := strings.ToLower(r.FormValue("username"))
		fullname := r.FormValue("fullname")
		password := r.FormValue("password")
		admin := r.FormValue("admin") == "true"

		if username == "" {
			data.Msg = "Username cannot be empty"
		} else if SliceContains(reserved, username) {
			data.Msg = "Username \"" + username + "\" is reserved"
		} else if exists, err := UserExists(username); err != nil {
			log.Println("[Admin:User:Create:Exists]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if exists {
			data.Msg = "Username \"" + username + "\" is taken"
		} else if salt, err := Salt(); err != nil {
			log.Println("[Admin:User:Create:Salt]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if _, err := db.Exec(
			"INSERT INTO users (name, name_full, pass, pass_algo, salt, is_admin) VALUES (?, ?, ?, ?, ?, ?)",
			username, fullname, Hash(password, salt), "argon2", salt, admin,
		); err != nil {
			log.Println("[Admin:User:Create:INSERT]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else {
			data.Msg = "User \"" + username + "\" created successfully"
		}
	}

	userCreate.Execute(w, data)
}
