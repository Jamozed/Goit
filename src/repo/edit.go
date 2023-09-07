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

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name))
	if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		goit.HttpError(w, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if readme, _ := findReadme(gr, ref); readme != "" {
		data.Readme = path.Join("/", repo.Name, "file", readme)
	}
	if licence, _ := findLicence(gr, ref); licence != "" {
		data.Licence = path.Join("/", repo.Name, "file", licence)
	}

	if r.Method == http.MethodPost {
		data.Form.Name = r.FormValue("reponame")
		data.Form.Description = r.FormValue("description")
		data.Form.IsPrivate = r.FormValue("visibility") == "private"

		if data.Form.Name == "" {
			data.Message = "Name cannot be empty"
		} else if slices.Contains(reserved, data.Form.Name) {
			data.Message = "Name \"" + data.Form.Name + "\" is reserved"
		} else if exists, err := goit.RepoExists(data.Form.Name); err != nil {
			log.Println("[/repo/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Form.Name != repo.Name {
			data.Message = "Name \"" + data.Form.Name + "\" is taken"
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
