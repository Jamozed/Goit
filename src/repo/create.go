// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"html/template"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/Jamozed/Goit/src/cron"
	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
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
		Title, Message                 string
		Name, Description              string
		DefaultBranch, Url, Visibility string
		IsMirror                       bool

		CsrfField template.HTML
	}{
		Title: "Repository - Create",

		CsrfField: csrf.TemplateField(r),
	}

	if r.Method == http.MethodPost {
		data.Name = r.FormValue("reponame")
		data.Description = r.FormValue("description")
		data.DefaultBranch = util.If(r.FormValue("branch") == "", "master", r.FormValue("branch"))
		data.Url = r.FormValue("url")
		data.Visibility = r.FormValue("visibility")
		data.IsMirror = r.FormValue("mirror") == "mirror"

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
		} else if len(data.Description) > 256 {
			data.Message = "Description cannot exceed 256 characters"
		} else if visibility := goit.VisibilityFromString(data.Visibility); visibility == -1 {
			data.Message = "Visibility \"" + data.Visibility + "\" is invalid"
		} else if rid, err := goit.CreateRepo(goit.Repo{
			OwnerId: user.Id, Name: data.Name, Description: data.Description, DefaultBranch: data.DefaultBranch,
			Upstream: data.Url, Visibility: visibility, IsMirror: data.IsMirror,
		}); err != nil {
			log.Println("[/repo/create]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		} else {
			if data.Url != "" {
				goit.Cron.Add(rid, cron.Immediate, func() {
					if err := goit.Pull(rid); err != nil {
						log.Println("[cron:import]", err.Error())
					}
					log.Println("[cron:import] imported", data.Name)
				})

				if data.IsMirror {
					util.Debugln("Adding mirror cron job for", data.Name)
					goit.Cron.Add(rid, cron.Daily, func() {
						if err := goit.Pull(rid); err != nil {
							log.Println("[cron:mirror]", err.Error())
						}
						log.Println("[cron:mirror] updated", data.Name)
					})
				}

				goit.Cron.Update()
			}

			http.Redirect(w, r, "/"+data.Name, http.StatusFound)
			return
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/create", data); err != nil {
		log.Println("[/repo/create]", err.Error())
	}
}
