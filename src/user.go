// user.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type User struct {
	Id       uint64
	Name     string
	FullName string
	Pass     []byte
	PassAlgo string
	Salt     []byte
	IsAdmin  bool
}

var reserved []string = []string{"admin", "repo", "static", "user"}

func HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	if ok, _ := AuthHttp(r); ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := struct{ Title, Message string }{"Login", ""}

	if r.Method == http.MethodPost {
		u := User{}
		username := strings.ToLower(r.FormValue("username"))
		password := r.FormValue("password")

		if username == "" {
			data.Message = "Username cannot be empty"
		} else if exists, err := UserExists(username); err != nil {
			log.Println("[User:Login:Exists]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if !exists {
			data.Message = "Invalid credentials"
		} else if err := db.QueryRow(
			"SELECT id, name, pass, pass_algo, salt FROM users WHERE name = ?", username,
		).Scan(&u.Id, &u.Name, &u.Pass, &u.PassAlgo, &u.Salt); err != nil {
			log.Println("[User:Login:SELECT]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if !bytes.Equal(Hash(password, u.Salt), u.Pass) {
			data.Message = "Invalid credentials"
		} else {
			expiry := time.Now().Add(15 * time.Minute)
			if s, err := NewSession(u.Id, expiry); err != nil {
				log.Println("[User:Login:Session]", err.Error())
				HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: s, Path: "/", Expires: expiry})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}
	}

	tmpl.ExecuteTemplate(w, "user_login", data)
}

func HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	EndSession(SessionCookie(r))
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func GetUser(id uint64) (*User, error) {
	u := User{}

	if err := db.QueryRow(
		"SELECT id, name, name_full, is_admin FROM users WHERE id = ?", id,
	).Scan(&u.Id, &u.Name, &u.FullName, &u.IsAdmin); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("[SELECT:user] %w", err)
		} else {
			return nil, nil
		}
	} else {
		return &u, nil
	}
}

func GetUserByName(name string) (*User, error) {
	u := &User{}

	err := db.QueryRow(
		"SELECT id, name, name_full, pass, pass_algo, salt, is_admin FROM users WHERE name = ?", strings.ToLower(name),
	).Scan(&u.Id, &u.Name, &u.FullName, &u.Pass, &u.PassAlgo, &u.Salt, &u.IsAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return u, nil
}

func UserExists(name string) (bool, error) {
	if err := db.QueryRow("SELECT name FROM users WHERE name = ?", strings.ToLower(name)).Scan(&name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		} else {
			return false, nil
		}
	} else {
		return true, nil
	}
}
