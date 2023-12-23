// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Jamozed/Goit/src/cron"
	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gorilla/csrf"
)

func HandleEdit(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	repo, err := goit.GetRepoByName(chi.URLParam(r, "repo"))
	if err != nil {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (!auth || repo.OwnerId != user.Id) {
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
		Editable, IsMirror            bool

		Edit struct {
			Id, Owner, Name, Description, Upstream string
			IsPrivate, IsMirror                    bool
			Message                                string
		}

		Transfer struct{ Owner, Message string }
		Delete   struct{ Message string }

		CsrfField template.HTML
	}{
		Title:       "Repository - Edit",
		Name:        repo.Name,
		Description: repo.Description,
		Url:         util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable:    (auth && repo.OwnerId == user.Id),
		IsMirror:    repo.IsMirror,

		CsrfField: csrf.TemplateField(r),
	}

	data.Edit.Id = fmt.Sprint(repo.Id)
	data.Edit.Owner = owner.FullName + " (" + owner.Name + ")[" + fmt.Sprint(owner.Id) + "]"
	data.Edit.Name = repo.Name
	data.Edit.Description = repo.Description
	data.Edit.Upstream = repo.Upstream
	data.Edit.IsPrivate = repo.IsPrivate
	data.Edit.IsMirror = repo.IsMirror

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
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
		if readme, _ := findPattern(gr, ref, readmePattern); readme != "" {
			data.Readme = filepath.Join("/", repo.Name, "file", readme)
		}
		if licence, _ := findPattern(gr, ref, licencePattern); licence != "" {
			data.Licence = filepath.Join("/", repo.Name, "file", licence)
		}
	}

	if r.Method == http.MethodPost {
		switch r.FormValue("action") {
		case "edit":
			data.Edit.Name = r.FormValue("reponame")
			data.Edit.Description = r.FormValue("description")
			data.Edit.Upstream = r.FormValue("upstream")
			data.Edit.IsPrivate = r.FormValue("visibility") == "private"
			data.Edit.IsMirror = r.FormValue("mirror") == "mirror"

			if data.Edit.Name == "" {
				data.Edit.Message = "Name cannot be empty"
			} else if slices.Contains(goit.Reserved, data.Edit.Name) || !goit.IsLegal(data.Name) {
				data.Edit.Message = "Name \"" + data.Edit.Name + "\" is illegal"
			} else if exists, err := goit.RepoExists(data.Edit.Name); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if exists && !strings.EqualFold(data.Edit.Name, repo.Name) {
				data.Edit.Message = "Name \"" + data.Edit.Name + "\" is taken"
			} else if len(data.Edit.Description) > 256 {
				data.Edit.Message = "Description cannot exceed 256 characters"
			} else if err := goit.UpdateRepo(repo.Id, goit.Repo{
				Name: data.Edit.Name, Description: data.Edit.Description, Upstream: data.Edit.Upstream,
				IsPrivate: data.Edit.IsPrivate, IsMirror: data.Edit.IsMirror,
			}); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				if (data.Edit.Upstream == "" && repo.Upstream != "") || !data.Edit.IsMirror {
					goit.Cron.RemoveFor(repo.Id)
					goit.Cron.Update()
				} else if data.Edit.Upstream != "" && data.Edit.IsMirror &&
					(data.Edit.Upstream != repo.Upstream || !repo.IsMirror) {
					goit.Cron.RemoveFor(repo.Id)

					goit.Cron.Add(repo.Id, cron.Immediate, func() {
						if err := goit.Pull(repo.Id); err != nil {
							log.Println("[cron:import]", err.Error())
						}
						log.Println("[cron:import] imported", data.Name)
					})

					goit.Cron.Add(repo.Id, cron.Daily, func() {
						if err := goit.Pull(repo.Id); err != nil {
							log.Println("[cron:mirror]", err.Error())
						}
						log.Println("[cron:mirror] updated", data.Edit.Name)
					})

					goit.Cron.Update()
				}

				http.Redirect(w, r, "/"+data.Edit.Name+"/edit", http.StatusFound)
				return
			}

		case "transfer":
			data.Transfer.Owner = r.FormValue("owner")

			if data.Transfer.Owner == "" {
				data.Transfer.Message = "New owner cannot be empty"
			} else if u, err := goit.GetUserByName(data.Transfer.Owner); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if u == nil {
				data.Transfer.Message = "User \"" + data.Transfer.Owner + "\" does not exist"
			} else if err := goit.ChownRepo(repo.Id, u.Id); err != nil {
				log.Println("[/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				log.Println("User", user.Id, "transferred repo", repo.Id, "ownership to", u.Id)
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
