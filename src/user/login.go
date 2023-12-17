// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package user

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/gorilla/csrf"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	auth, _, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	if auth {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	data := struct {
		Title, Message, Name string
		FocusPw              bool

		CsrfField template.HTML
	}{
		Title: "Login",

		CsrfField: csrf.TemplateField(r),
	}

	if r.Method == http.MethodPost {
		data.Name = r.FormValue("username")
		password := r.FormValue("password")

		if data.Name == "" {
			data.Message = "Username cannot be empty"
			goto execute
		}

		ip := goit.Ip(r)

		user, err := goit.GetUserByName(data.Name)
		if err != nil {
			log.Println("[/user/login]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if user == nil || !bytes.Equal(goit.Hash(password, user.Salt), user.Pass) {
			data.Message = "Invalid credentials"
			data.FocusPw = true

			log.Println("[login] login attempt with", data.Name, "from", ip)

			goto execute
		}

		sess, err := goit.NewSession(user.Id, ip, time.Now().Add(2*24*time.Hour))
		if err != nil {
			log.Println("[/user/login]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}

		log.Println("[login]", user.Name, "logged in from", ip)

		goit.SetSessionCookie(w, user.Id, sess)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

execute:
	if err := goit.Tmpl.ExecuteTemplate(w, "user/login", data); err != nil {
		log.Println("[/user/login]", err.Error())
	}
}
