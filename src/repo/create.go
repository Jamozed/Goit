package repo

import (
	"log"
	"net/http"

	goit "github.com/Jamozed/Goit/src"
	"github.com/Jamozed/Goit/src/util"
)

var reserved []string = []string{"admin", "repo", "static", "user"}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	ok, uid := goit.AuthCookie(w, r, true)
	if !ok {
		goit.HttpError(w, http.StatusUnauthorized)
		return
	}

	data := struct {
		Title, Message    string
		Name, Description string
		IsPrivate         bool
	}{Title: "Repository - Create"}

	if r.Method == http.MethodPost {
		data.Name = r.FormValue("reponame")
		data.Description = r.FormValue("description")
		data.IsPrivate = r.FormValue("visibility") == "private"

		if data.Name == "" {
			data.Message = "Name cannot be empty"
		} else if util.SliceContains(reserved, data.Name) {
			data.Message = "Name \"" + data.Name + "\" is reserved"
		} else if exists, err := goit.RepoExists(data.Name); err != nil {
			log.Println("[/repo/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists {
			data.Message = "Name \"" + data.Name + "\" is taken"
		} else if err := goit.CreateRepo(goit.Repo{
			OwnerId: uid, Name: data.Name, Description: data.Description, IsPrivate: data.IsPrivate,
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
