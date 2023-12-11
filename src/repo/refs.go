// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"errors"
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
)

func HandleRefs(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	repo, err := goit.GetRepoByName(chi.URLParam(r, "repo"))
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || repo.OwnerId != user.Id)) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct{ Name, Message, Author, LastCommit, Hash string }
	data := struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Branches, Tags                []row
		Editable                      bool
	}{
		Title: repo.Name + " - References", Name: repo.Name, Description: repo.Description,
		Url:      util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable: (auth && repo.OwnerId == user.Id),
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			log.Println("[/repo/log]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}
	} else {
		if readme, _ := findReadme(gr, ref); readme != "" {
			data.Readme = filepath.Join("/", repo.Name, "file", readme)
		}
		if licence, _ := findLicence(gr, ref); licence != "" {
			data.Licence = filepath.Join("/", repo.Name, "file", licence)
		}
	}

	if iter, err := gr.Branches(); err != nil {
		log.Println("[/repo/refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(r *plumbing.Reference) error {
		commit, err := gr.CommitObject(r.Hash())
		if err != nil {
			return err
		}

		data.Branches = append(data.Branches, row{
			Name: r.Name().Short(), Message: strings.SplitN(commit.Message, "\n", 2)[0], Author: commit.Author.Name,
			LastCommit: commit.Author.When.UTC().Format(time.DateTime), Hash: r.Hash().String(),
		})

		return nil
	}); err != nil {
		log.Println("[/repo/refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if iter, err := gr.Tags(); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(r *plumbing.Reference) error {
		commit, err := gr.CommitObject(r.Hash())
		if err != nil {
			return err
		}

		data.Tags = append(data.Tags, row{
			Name: r.Name().Short(), Message: strings.SplitN(commit.Message, "\n", 2)[0], Author: commit.Author.Name,
			LastCommit: commit.Author.When.UTC().Format(time.DateTime), Hash: r.Hash().String(),
		})

		return nil
	}); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/refs", data); err != nil {
		log.Println("[/repo/refs]", err.Error())
	}
}
