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

	"github.com/Jamozed/Goit/src/cron"
	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/csrf"
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
		log.Println("[/admin/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	id, err := strconv.ParseInt(r.FormValue("repo"), 10, 64)
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
		owner = &goit.User{}
	}

	data := struct {
		Title, Name string

		Edit struct {
			Id, Owner, Name, Description string
			DefaultBranch, Upstream      string
			IsPrivate, IsMirror          bool
			Message                      string
		}

		Transfer struct{ Owner, Message string }
		Delete   struct{ Message string }

		CsrfField template.HTML
	}{
		Title: "Admin - Edit Repository",
		Name:  repo.Name,

		CsrfField: csrf.TemplateField(r),
	}

	data.Edit.Id = fmt.Sprint(repo.Id)
	data.Edit.Owner = owner.FullName + " (" + owner.Name + ")[" + fmt.Sprint(owner.Id) + "]"
	data.Edit.Name = repo.Name
	data.Edit.Description = repo.Description
	data.Edit.DefaultBranch = repo.DefaultBranch
	data.Edit.Upstream = repo.Upstream
	data.Edit.IsPrivate = repo.IsPrivate
	data.Edit.IsMirror = repo.IsMirror

	if r.Method == http.MethodPost {
		switch r.FormValue("action") {
		case "edit":
			data.Edit.Name = r.FormValue("reponame")
			data.Edit.Description = r.FormValue("description")
			data.Edit.DefaultBranch = util.If(r.FormValue("branch") == "", "master", r.FormValue("branch"))
			data.Edit.Upstream = r.FormValue("upstream")
			data.Edit.IsPrivate = r.FormValue("visibility") == "private"
			data.Edit.IsMirror = r.FormValue("mirror") == "mirror"

			if data.Edit.Name == "" {
				data.Edit.Message = "Name cannot be empty"
			} else if slices.Contains(goit.Reserved, data.Edit.Name) || !goit.IsLegal(data.Name) {
				data.Edit.Message = "Name \"" + data.Edit.Name + "\" is illegal"
			} else if exists, err := goit.RepoExists(data.Edit.Name); err != nil {
				log.Println("[/admin/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if exists && !strings.EqualFold(data.Edit.Name, repo.Name) {
				data.Edit.Message = "Name \"" + data.Edit.Name + "\" is taken"
			} else if len(data.Edit.Description) > 256 {
				data.Edit.Message = "Description cannot exceed 256 characters"
			} else if err := goit.UpdateRepo(repo.Id, goit.Repo{
				Name: data.Edit.Name, Description: data.Edit.Description, DefaultBranch: data.Edit.DefaultBranch,
				Upstream: data.Edit.Upstream, IsPrivate: data.Edit.IsPrivate, IsMirror: data.Edit.IsMirror,
			}); err != nil {
				log.Println("[/admin/repo/edit]", err.Error())
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

				data.Edit.Message = "Repository \"" + repo.Name + "\" updated successfully"
			}

		case "transfer":
			data.Transfer.Owner = r.FormValue("owner")

			if data.Transfer.Owner == "" {
				data.Transfer.Message = "New owner cannot be empty"
			} else if u, err := goit.GetUserByName(data.Transfer.Owner); err != nil {
				log.Println("[/admin/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if u == nil {
				data.Transfer.Message = "User \"" + data.Transfer.Owner + "\" does not exist"
			} else if err := goit.ChownRepo(repo.Id, u.Id); err != nil {
				log.Println("[/admin/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				log.Println("User", user.Id, "transferred repo", repo.Id, "ownership to", u.Id)
				http.Redirect(w, r, "/admin/repo/edit?repo="+data.Edit.Id, http.StatusFound)
				return
			}

		case "delete":
			var reponame = r.FormValue("reponame")

			if reponame != repo.Name {
				data.Delete.Message = "Input does not match the repository name"
			} else if err := goit.DelRepo(repo.Id); err != nil {
				log.Println("[/admin/repo/edit]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/admin/repos", http.StatusFound)
				return
			}
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/repo/edit", data); err != nil {
		log.Println("[/admin/repo/edit]", err.Error())
	}
}
