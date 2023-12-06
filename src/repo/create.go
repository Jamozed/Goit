package repo

import (
	"html/template"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/gorilla/csrf"
)

func HandleCreate(w http.ResponseWriter, r *http.Request) {
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
		Title, Message    string
		Name, Description string
		IsPrivate         bool

		CsrfField template.HTML
	}{
		Title: "Repository - Create",

		CsrfField: csrf.TemplateField(r),
	}

	if r.Method == http.MethodPost {
		data.Name = r.FormValue("reponame")
		data.Description = r.FormValue("description")
		data.IsPrivate = r.FormValue("visibility") == "private"

		if data.Name == "" {
			data.Message = "Name cannot be empty"
		} else if slices.Contains(goit.Reserved, strings.SplitN(data.Name, "/", 2)[0]) || !goit.IsLegal(data.Name) {
			data.Message = "Name \"" + data.Name + "\" is illegal"
		} else if exists, err := goit.RepoExists(data.Name); err != nil {
			log.Println("[/repo/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists {
			data.Message = "Name \"" + data.Name + "\" is taken"
		} else if err := goit.CreateRepo(goit.Repo{
			OwnerId: user.Id, Name: data.Name, Description: data.Description, IsPrivate: data.IsPrivate,
		}); err != nil {
			log.Println("[/repo/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else {
			http.Redirect(w, r, "/"+data.Name, http.StatusFound)
			return
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/create", data); err != nil {
		log.Println("[/repo/create]", err.Error())
	}
}
