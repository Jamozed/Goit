package user

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/gorilla/csrf"
)

func HandleEdit(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	if !auth {
		goit.HttpError(w, http.StatusUnauthorized)
		return
	}

	data := struct {
		Title, MessageA, MessageB string

		Form struct{ Id, Name, FullName string }

		CsrfField template.HTML
	}{
		Title: "User - Edit",

		CsrfField: csrf.TemplateField(r),
	}

	data.Form.Id = fmt.Sprint(user.Id)
	data.Form.Name = user.Name
	data.Form.FullName = user.FullName

	if r.Method == http.MethodPost {
		if r.FormValue("submit") == "Update" {
			data.Form.Name = r.FormValue("username")
			data.Form.FullName = r.FormValue("fullname")

			if data.Form.Name == "" {
				data.MessageA = "Username cannot be empty"
			} else if slices.Contains(goit.Reserved, data.Form.Name) && user.Id != 0 {
				data.MessageA = "Username \"" + data.Form.Name + "\" is reserved"
			} else if exists, err := goit.UserExists(data.Form.Name); err != nil {
				log.Println("[/user/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if exists && data.Form.Name != user.Name {
				data.MessageA = "Username \"" + data.Form.Name + "\" is taken"
			} else if err := goit.UpdateUser(user.Id, goit.User{
				Name: data.Form.Name, FullName: data.Form.FullName, IsAdmin: user.IsAdmin,
			}); err != nil {
				log.Println("[/user/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/user/edit?m=a", http.StatusFound)
				return
			}
		} else if r.FormValue("submit") == "Update Password" {
			password := r.FormValue("password")
			newPassword := r.FormValue("new_password")
			confirmPassword := r.FormValue("confirm_password")

			if password == "" {
				data.MessageB = "Current Password cannot be empty"
			} else if newPassword == "" {
				data.MessageB = "New Password cannot be empty"
			} else if confirmPassword == "" {
				data.MessageB = "Confirm New Password cannot be empty"
			} else if newPassword != confirmPassword {
				data.MessageB = "New Password and Confirm Password do not match"
			} else if !bytes.Equal(goit.Hash(password, user.Salt), user.Pass) {
				data.MessageB = "Password incorrect"
			} else if err := goit.UpdatePassword(user.Id, newPassword); err != nil {
				log.Println("[/user/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/user/edit?m=b", http.StatusFound)
				return
			}
		} else {
			data.MessageA = "Invalid submit value"
		}
	}

	switch r.FormValue("m") {
	case "a":
		data.MessageA = "User updated successfully"
	case "b":
		data.MessageB = "Password updated successfully"
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "user/edit", data); err != nil {
		log.Println("[/user/edit]", err.Error())
	}
}
