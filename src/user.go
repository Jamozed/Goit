// user.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jamozed/Goit/res"
)

type User struct {
	Id       uint64
	Name     string
	NameFull string
	Pass     []byte
	PassAlgo string
	Salt     []byte
	IsAdmin  bool
}

var (
	reserved []string = []string{"admin", "repo", "static", "user"}

	userLogin  *template.Template = template.Must(template.New("user_login").Parse(res.UserLogin))
	userCreate *template.Template = template.Must(template.New("user_create").Parse(res.UserCreate))
)

func (g *Goit) HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	if ok, _ := AuthHttp(r); ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := struct{ Msg string }{""}

	if r.Method == http.MethodPost {
		u := User{}
		username := strings.ToLower(r.FormValue("username"))
		password := r.FormValue("password")

		if username == "" {
			data.Msg = "Username cannot be empty"
		} else if exists, err := g.UserExists(username); err != nil {
			log.Println("[User:Login:Exists]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if !exists {
			data.Msg = "Invalid credentials"
		} else if err := g.db.QueryRow(
			"SELECT id, name, pass, pass_algo, salt FROM users WHERE name = ?", username,
		).Scan(&u.Id, &u.Name, &u.Pass, &u.PassAlgo, &u.Salt); err != nil {
			log.Println("[User:Login:SELECT]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if !bytes.Equal(Hash(password, u.Salt), u.Pass) {
			data.Msg = "Invalid credentials"
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

	userLogin.Execute(w, data)
}

func (g *Goit) HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	EndSession(SessionCookie(r))
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (g *Goit) GetUser(id uint64) (*User, error) {
	u := User{}

	if err := g.db.QueryRow(
		"SELECT id, name, name_full, is_admin FROM users WHERE id = ?", id,
	).Scan(&u.Id, &u.Name, &u.NameFull, &u.IsAdmin); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("[SELECT:user] %w", err)
		} else {
			return nil, nil
		}
	} else {
		return &u, nil
	}
}

func (g *Goit) UserExists(name string) (bool, error) {
	if err := g.db.QueryRow("SELECT name FROM users WHERE name = ?", strings.ToLower(name)).Scan(&name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		} else {
			return false, nil
		}
	} else {
		return true, nil
	}
}
