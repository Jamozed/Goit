package repo

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func HandleLog(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	tpath := chi.URLParam(r, "*")

	repo, err := goit.GetRepoByName(chi.URLParam(r, "repo"))
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || repo.OwnerId != user.Id)) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	// var offset uint64 = 0
	// if o := r.URL.Query().Get("o"); o != "" {
	// 	if i, err := strconv.ParseUint(o, 10, 64); err != nil {
	// 		goit.HttpError(w, http.StatusBadRequest)
	// 		return
	// 	} else {
	// 		offset = i
	// 	}
	// }

	type row struct{ Hash, Date, Message, Author, Files, Additions, Deletions string }
	data := struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Commits                       []row
		Editable                      bool
	}{
		Title: repo.Name + " - Log", Name: repo.Name, Description: repo.Description,
		Url:      util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable: (auth && repo.OwnerId == user.Id),
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		goto execute
	} else if err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if readme, _ := findReadme(gr, ref); readme != "" {
		data.Readme = filepath.Join("/", repo.Name, "file", readme)
	}
	if licence, _ := findLicence(gr, ref); licence != "" {
		data.Licence = filepath.Join("/", repo.Name, "file", licence)
	}

	if iter, err := gr.Log(&git.LogOptions{From: ref.Hash()}); err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(c *object.Commit) error {
		var files, additions, deletions int

		/* TODO speed this up or cache it, diff calculations are quite slow */
		if stats, err := goit.DiffStats(c); err != nil {
			log.Println("[/repo/log]", err.Error())
		} else if tpath != "" {
			for _, s := range stats {
				if s.Name == tpath || strings.HasPrefix(s.Name, tpath+"/") {
					files += 1
					additions += s.Addition
					deletions += s.Deletion
				}
			}

			if files == 0 {
				return nil
			}
		} else {
			files = len(stats)
			for _, s := range stats {
				additions += s.Addition
				deletions += s.Deletion
			}
		}

		data.Commits = append(data.Commits, row{
			Hash: c.Hash.String(), Date: c.Author.When.UTC().Format(time.DateTime),
			Message: strings.SplitN(c.Message, "\n", 2)[0], Author: c.Author.Name, Files: fmt.Sprint(files),
			Additions: "+" + fmt.Sprint(additions), Deletions: "-" + fmt.Sprint(deletions),
		})

		return nil
	}); err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

execute:
	if err := goit.Tmpl.ExecuteTemplate(w, "repo/log", data); err != nil {
		log.Println("[/repo/log]", err.Error())
	}
}
