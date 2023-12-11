// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package admin

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/gorilla/csrf"
)

func HandleUsers(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin/users]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct{ Id, Name, FullName, IsAdmin string }
	data := struct {
		Title string
		Users []row
	}{Title: "Admin - Users"}

	users, err := goit.GetUsers()
	if err != nil {
		log.Println("[admin/users]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	for _, u := range users {
		data.Users = append(data.Users, row{
			fmt.Sprint(u.Id), u.Name, u.FullName, util.If(u.IsAdmin, "true", "false"),
		})
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/users", data); err != nil {
		log.Println("[/admin/users]", err.Error())
	}
}

func HandleUserCreate(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin/users]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	data := struct {
		Title, Message string

		Form struct {
			Name, FullName string
			IsAdmin        bool
		}

		CsrfField template.HTML
	}{
		Title: "Admin - Create User",

		CsrfField: csrf.TemplateField(r),
	}

	if r.Method == http.MethodPost {
		data.Form.Name = strings.ToLower(r.FormValue("username"))
		data.Form.FullName = r.FormValue("fullname")
		password := r.FormValue("password")
		data.Form.IsAdmin = r.FormValue("admin") == "true"

		if data.Form.Name == "" {
			data.Message = "Username cannot be empty"
		} else if slices.Contains(goit.Reserved, data.Form.Name) || !goit.IsLegal(data.Form.Name) {
			data.Message = "Username \"" + data.Form.Name + "\" is illegal"
		} else if exists, err := goit.UserExists(data.Form.Name); err != nil {
			log.Println("[/admin/user/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists {
			data.Message = "Username \"" + data.Form.Name + "\" is taken"
		} else if salt, err := goit.Salt(); err != nil {
			log.Println("[/admin/user/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if err := goit.CreateUser(goit.User{
			Name: data.Form.Name, FullName: data.Form.FullName, Pass: goit.Hash(password, salt), PassAlgo: "argon2",
			Salt: salt, IsAdmin: data.Form.IsAdmin,
		}); err != nil {
			log.Println("[/admin/user/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else {
			// data.Message = "User \"" + data.Form.Name + "\" created successfully"
			http.Redirect(w, r, "/admin/users", http.StatusFound)
			return
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/user/create", data); err != nil {
		log.Println("[/admin/user/create]", err.Error())
	}
}

func HandleUserEdit(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin/users]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	uid, err := strconv.ParseInt(r.URL.Query().Get("user"), 10, 64)
	if err != nil {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	u, err := goit.GetUser(uid)
	if err != nil {
		log.Println("[/admin/user/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if u == nil {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	data := struct {
		Title, Message string

		Form struct {
			Id, Name, FullName string
			IsAdmin            bool
		}

		CsrfField template.HTML
	}{
		Title: "Admin - Edit User",

		CsrfField: csrf.TemplateField(r),
	}

	data.Form.Id = fmt.Sprint(u.Id)
	data.Form.Name = u.Name
	data.Form.FullName = u.FullName
	data.Form.IsAdmin = u.IsAdmin

	if r.Method == http.MethodPost {
		data.Form.Name = strings.ToLower(r.FormValue("username"))
		data.Form.FullName = r.FormValue("fullname")
		password := r.FormValue("password")
		data.Form.IsAdmin = r.FormValue("admin") == "true"

		if data.Form.Name == "" {
			data.Message = "Username cannot be empty"
		} else if slices.Contains(goit.Reserved, data.Form.Name) && user.Id != 0 || !goit.IsLegal(data.Form.Name) {
			data.Message = "Username \"" + data.Form.Name + "\" is illegal"
		} else if exists, err := goit.UserExists(data.Form.Name); err != nil {
			log.Println("[/admin/user/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Form.Name != u.Name {
			data.Message = "Username \"" + data.Form.Name + "\" is taken"
		} else {
			if err := goit.UpdateUser(u.Id, goit.User{
				Name: data.Form.Name, FullName: data.Form.FullName, IsAdmin: data.Form.IsAdmin,
			}); err != nil {
				log.Println("[/admin/user/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			}

			if password != "" {
				if err := goit.UpdatePassword(u.Id, password); err != nil {
					log.Println("[/admin/user/edit]", err.Error())
					goit.HttpError(w, http.StatusInternalServerError)
					return
				}
			}

			data.Message = "User \"" + u.Name + "\" updated successfully"
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/user/edit", data); err != nil {
		log.Println("[/admin/user/edit]", err.Error())
	}
}
