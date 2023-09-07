// user.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/util"
)

type User struct {
	Id       int64
	Name     string
	FullName string
	Pass     []byte
	PassAlgo string
	Salt     []byte
	IsAdmin  bool
}

var reserved []string = []string{"admin", "repo", "static", "user"}

func HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	id, s := GetSessionCookie(r)
	EndSession(id, s.Token)
	EndSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func HandleUserSessions(w http.ResponseWriter, r *http.Request) {
	auth, uid := AuthCookie(w, r, true)
	if !auth {
		HttpError(w, http.StatusUnauthorized)
		return
	}

	_, ss := GetSessionCookie(r)

	type row struct{ Ip, Seen, Expiry, Current string }
	data := struct {
		Title    string
		Sessions []row
	}{Title: "User - Sessions"}

	for k, v := range Sessions[uid] {
		data.Sessions = append(data.Sessions, row{
			Ip: v.Ip, Seen: v.Seen.Format(time.DateTime), Expiry: v.Expiry.Format(time.DateTime),
			Current: util.If(k == ss.Token, "(current)", ""),
		})
	}

	if err := Tmpl.ExecuteTemplate(w, "user/sessions", data); err != nil {
		log.Println("[/user/login]", err.Error())
	}
}

func GetUser(id int64) (*User, error) {
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

func UpdateUser(uid int64, user User) error {
	if _, err := db.Exec(
		"UPDATE users SET name = ?, name_full = ? WHERE id = ?",
		user.Name, user.FullName, uid,
	); err != nil {
		return err
	}

	return nil
}
