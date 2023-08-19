package repo

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	goit "github.com/Jamozed/Goit/src"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

func HandleLog(w http.ResponseWriter, r *http.Request) {
	auth, uid := goit.AuthCookie(w, r, true)

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || repo.OwnerId != uid)) {
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
		Editable: (auth && repo.OwnerId == uid),
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name))
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
		data.Readme = path.Join("/", repo.Name, "file", readme)
	}
	if licence, _ := findLicence(gr, ref); licence != "" {
		data.Licence = path.Join("/", repo.Name, "file", licence)
	}

	if iter, err := gr.Log(&git.LogOptions{From: ref.Hash()}); err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(c *object.Commit) error {
		var files, additions, deletions int

		if stats, err := goit.DiffStats(c); err != nil {
			log.Println("[/repo/log]", err.Error())
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
