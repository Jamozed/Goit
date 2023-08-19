package repo

import (
	"fmt"
	"log"
	"net/http"

	goit "github.com/Jamozed/Goit/src"
	"github.com/Jamozed/Goit/src/util"
	"github.com/gorilla/mux"
)

func HandleEdit(w http.ResponseWriter, r *http.Request) {
	auth, uid := goit.AuthCookie(w, r, true)

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (!auth || repo.OwnerId != uid) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	owner, err := goit.GetUser(repo.OwnerId)
	if err != nil {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if owner == nil {
		log.Println("[/repo/edit]", repo.Id, "is owned by a nonexistent user")
		/* TODO have admin adopt the orphaned repository */
		owner = &goit.User{}
	}

	data := struct {
		Title, Name, Description, Url string
		Readme, Licence, Message      string
		Editable                      bool

		Form struct {
			Id, Owner, Name, Description string
			IsPrivate                    bool
		}
	}{
		Title:       "Repository - Edit",
		Name:        repo.Name,
		Description: repo.Description,
		Url:         util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable:    (auth && repo.OwnerId == uid),
	}

	data.Form.Id = fmt.Sprint(repo.Id)
	data.Form.Owner = owner.FullName + " (" + owner.Name + ")[" + fmt.Sprint(owner.Id) + "]"
	data.Form.Name = repo.Name
	data.Form.Description = repo.Description
	data.Form.IsPrivate = repo.IsPrivate

	if r.Method == http.MethodPost {
		data.Form.Name = r.FormValue("reponame")
		data.Form.Description = r.FormValue("description")
		data.Form.IsPrivate = r.FormValue("visibility") == "private"

		if data.Name == "" {
			data.Message = "Name cannot be empty"
		} else if util.SliceContains(reserved, data.Name) {
			data.Message = "Name \"" + data.Name + "\" is reserved"
		} else if exists, err := goit.RepoExists(data.Name); err != nil {
			log.Println("[/repo/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Name != repo.Name {
			data.Message = "Name \"" + data.Name + "\" is taken"
		} else if err := goit.UpdateRepo(repo.Id, goit.Repo{
			Name: data.Form.Name, Description: data.Form.Description, IsPrivate: data.Form.IsPrivate,
		}); err != nil {
			log.Println("[/repo/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else {
			http.Redirect(w, r, "/"+data.Form.Name+"/edit", http.StatusFound)
			return
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/edit", data); err != nil {
		log.Println("[/repo/edit]", err.Error())
	}
}
