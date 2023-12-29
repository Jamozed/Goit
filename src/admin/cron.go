// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package admin

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
)

func HandleCron(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[/admin/cron]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct{ Id, Repo, Schedule, Next, Last string }
	data := struct {
		Title string
		Jobs  []row
	}{Title: "Admin - Cron"}

	for _, job := range goit.Cron.Jobs() {
		repo := &goit.Repo{}

		if job.Rid != -1 {
			if r, err := goit.GetRepo(job.Rid); err != nil {
				log.Println("[/admin/cron]", err.Error())
			} else if r != nil {
				repo = r
			}
		}

		data.Jobs = append(data.Jobs, row{
			Id:       fmt.Sprint(job.Id),
			Repo:     repo.Name,
			Schedule: job.Schedule.String(),
			Next:     job.Next.String(),
			Last:     util.If(job.Last == time.Time{}, "never", job.Last.String()),
		})
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/cron", data); err != nil {
		log.Println("[/admin/cron]", err.Error())
	}
}
