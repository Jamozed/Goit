// user/login.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package user

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"time"

	goit "github.com/Jamozed/Goit/src"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if auth, _ := goit.AuthCookie(w, r, true); auth {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	data := struct {
		Title, Message, Name string
		FocusPw              bool
	}{Title: "Login"}

	if r.Method == http.MethodPost {
		data.Name = r.FormValue("username")
		password := r.FormValue("password")

		if data.Name == "" {
			data.Message = "Username cannot be empty"
			goto execute
		}

		user, err := goit.GetUserByName(data.Name)
		if err != nil {
			log.Println("[/user/login]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if user == nil || !bytes.Equal(goit.Hash(password, user.Salt), user.Pass) {
			data.Message = "Invalid credentials"
			data.FocusPw = true
			goto execute
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		sess, err := goit.NewSession(user.Id, ip, time.Now().Add(2*24*time.Hour))
		if err != nil {
			log.Println("[/user/login]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}

		goit.SetSessionCookie(w, user.Id, sess)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

execute:
	if err := goit.Tmpl.ExecuteTemplate(w, "user/login", data); err != nil {
		log.Println("[/user/login]", err.Error())
	}
}
