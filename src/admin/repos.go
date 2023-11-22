package admin

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
)

func HandleRepos(w http.ResponseWriter, r *http.Request) {
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

	type row struct{ Id, Owner, Name, Visibility, Size string }
	data := struct {
		Title string
		Repos []row
	}{Title: "Admin - Repositories"}

	repos, err := goit.GetRepos()
	if err != nil {
		log.Println("[/admin/repos]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	for _, r := range repos {
		u, err := goit.GetUser(r.OwnerId)
		if err != nil {
			log.Println("[/admin/repos]", err.Error())
			u = &goit.User{}
		}

		size, err := util.DirSize(goit.RepoPath(r.Name, true))
		if err != nil {
			log.Println("[/admin/repos]", err.Error())
		}

		data.Repos = append(data.Repos, row{
			fmt.Sprint(r.Id), u.Name, r.Name, util.If(r.IsPrivate, "private", "public"), humanize.IBytes(size),
		})
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/repos", data); err != nil {
		log.Println("[/admin/repos]", err.Error())
	}
}

func HandleRepoEdit(w http.ResponseWriter, r *http.Request) {
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

	id, err := strconv.ParseInt(r.URL.Query().Get("repo"), 10, 64)
	if err != nil {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	repo, err := goit.GetRepo(id)
	if err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	owner, err := goit.GetUser(repo.OwnerId)
	if err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if owner == nil {
		log.Println("[/admin/repo/edit]", repo.Id, "is owned by a nonexistent user")
		/* TODO have admin adopt the orphaned repository */
		owner = &goit.User{}
	}

	data := struct {
		Title, Message string

		Form struct {
			Id, Owner, Name, Description string
			IsPrivate                    bool
		}
	}{
		Title: "Admin - Edit Repository",
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

		if data.Form.Name == "" {
			data.Message = "Name cannot be empty"
		} else if slices.Contains(goit.Reserved, data.Form.Name) {
			data.Message = "Name \"" + data.Form.Name + "\" is reserved"
		} else if exists, err := goit.RepoExists(data.Form.Name); err != nil {
			log.Println("[/admin/repo/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else if exists && data.Form.Name != repo.Name {
			data.Message = "Name \"" + data.Form.Name + "\" is taken"
		} else if len(data.Form.Description) > 256 {
			data.Message = "Description cannot exceed 256 characters"
		} else if err := goit.UpdateRepo(repo.Id, goit.Repo{
			Name: data.Form.Name, Description: data.Form.Description, IsPrivate: data.Form.IsPrivate,
		}); err != nil {
			log.Println("[/admin/repo/edit]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else {
			data.Message = "Repository \"" + repo.Name + "\" updated successfully"
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/repo/edit", data); err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
	}
}
