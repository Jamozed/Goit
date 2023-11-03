package repo

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"slices"

	goit "github.com/Jamozed/Goit/src"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
		Readme, Licence               string
		Editable                      bool

		Edit struct {
			Id, Owner, Name, Description string
			IsPrivate                    bool
			Message                      string
		}

		Transfer struct{ Owner, Message string }
		Delete   struct{ Message string }
	}{
		Title:       "Repository - Edit",
		Name:        repo.Name,
		Description: repo.Description,
		Url:         util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable:    (auth && repo.OwnerId == uid),
	}

	data.Edit.Id = fmt.Sprint(repo.Id)
	data.Edit.Owner = owner.FullName + " (" + owner.Name + ")[" + fmt.Sprint(owner.Id) + "]"
	data.Edit.Name = repo.Name
	data.Edit.Description = repo.Description
	data.Edit.IsPrivate = repo.IsPrivate

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name))
	if err != nil {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if ref != nil {
		if readme, _ := findReadme(gr, ref); readme != "" {
			data.Readme = path.Join("/", repo.Name, "file", readme)
		}
		if licence, _ := findLicence(gr, ref); licence != "" {
			data.Licence = path.Join("/", repo.Name, "file", licence)
		}
	}

	if r.Method == http.MethodPost {
		switch r.FormValue("action") {
		case "edit":
			data.Edit.Name = r.FormValue("reponame")
			data.Edit.Description = r.FormValue("description")
			data.Edit.IsPrivate = r.FormValue("visibility") == "private"

			if data.Edit.Name == "" {
				data.Edit.Message = "Name cannot be empty"
			} else if slices.Contains(goit.Reserved, data.Edit.Name) {
				data.Edit.Message = "Name \"" + data.Edit.Name + "\" is reserved"
			} else if exists, err := goit.RepoExists(data.Edit.Name); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if exists && data.Edit.Name != repo.Name {
				data.Edit.Message = "Name \"" + data.Edit.Name + "\" is taken"
			} else if err := goit.UpdateRepo(repo.Id, goit.Repo{
				Name: data.Edit.Name, Description: data.Edit.Description, IsPrivate: data.Edit.IsPrivate,
			}); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/"+data.Edit.Name+"/edit", http.StatusFound)
				return
			}

		case "transfer":
			data.Transfer.Owner = r.FormValue("owner")

			if data.Transfer.Owner == "" {
				data.Transfer.Message = "New owner cannot be empty"
			} else if user, err := goit.GetUserByName(data.Transfer.Owner); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if user == nil {
				data.Transfer.Message = "User \"" + data.Transfer.Owner + "\" does not exist"
			} else if err := goit.ChownRepo(repo.Id, user.Id); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/"+data.Edit.Name, http.StatusFound)
				return
			}

		case "delete":
			var reponame = r.FormValue("reponame")

			if reponame != repo.Name {
				data.Delete.Message = "Input does not match the repository name"
			} else if err := goit.DelRepo(repo.Id); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/edit", data); err != nil {
		log.Println("[/repo/edit]", err.Error())
	}
}
