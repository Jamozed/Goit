package user

import (
	"fmt"
	"log"
	"net/http"
	"slices"

	goit "github.com/Jamozed/Goit/src"
)

func HandleEdit(w http.ResponseWriter, r *http.Request) {
	auth, uid := goit.AuthCookie(w, r, true)
	if !auth {
		goit.HttpError(w, http.StatusUnauthorized)
		return
	}

	user, err := goit.GetUser(uid)
	if err != nil {
		log.Println("[/user/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if user == nil {
		log.Println("[/user/edit]", uid, "is a nonexistent UID")
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	data := struct {
		Title, Message string

		Form struct{ Id, Name, FullName string }
	}{
		Title: "User - Edit",
	}

	data.Form.Id = fmt.Sprint(user.Id)
	data.Form.Name = user.Name
	data.Form.FullName = user.FullName

	if r.Method == http.MethodPost {
		data.Form.Name = r.FormValue("username")
		data.Form.FullName = r.FormValue("fullname")

		if data.Form.Name == "" {
			data.Message = "Username cannot be empty"
		} else if slices.Contains(reserved, data.Form.Name) && uid != 0 {
			data.Message = "Username \"" + data.Form.Name + "\" is reserved"
		} else if exists, err := goit.UserExists(data.Form.Name); err != nil {
			log.Println("[/user/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Form.Name != user.Name {
			data.Message = "Username \"" + data.Form.Name + "\" is taken"
		} else if err := goit.UpdateUser(user.Id, goit.User{
			Name: data.Form.Name, FullName: data.Form.FullName,
		}); err != nil {
			log.Println("[/user/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else {
			http.Redirect(w, r, "/user/edit", http.StatusFound)
			return
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "user/edit", data); err != nil {
		log.Println("[/user/edit]", err.Error())
	}
}
