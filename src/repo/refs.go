// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	} else if repo == nil || !goit.IsVisible(repo, auth, user) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct {
		Name, Hash, Message, Author, LastCommit string
		Commits                                 uint64
	}
	data := struct {
		HeaderFields
		Title          string
		Branches, Tags []row
	}{
		Title:        repo.Name + " - References",
		HeaderFields: GetHeaderFields(auth, user, repo, r.Host),
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
		if readme, _ := findPattern(gr, ref, readmePattern); readme != "" {
			data.Readme = filepath.Join("/", repo.Name, "file", readme)
		}
		if licence, _ := findPattern(gr, ref, licencePattern); licence != "" {
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

		commits, err := goit.CommitCount(repo.Name, r.Name().Short(), r.Hash())
		if err != nil {
			return err
		}

		data.Branches = append(data.Branches, row{
			Name: r.Name().Short(), Hash: r.Hash().String(), Message: strings.SplitN(commit.Message, "\n", 2)[0],
			Author: commit.Author.Name, LastCommit: commit.Author.When.UTC().Format(time.DateTime), Commits: commits,
		})

		return nil
	}); err != nil {
		log.Println("[/repo/refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if iter, err := gr.Tags(); err != nil {
		log.Println("[/repo/refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(r *plumbing.Reference) error {
		var c *object.Commit

		if tag, err := gr.TagObject(r.Hash()); err != nil {
			if !errors.Is(err, plumbing.ErrObjectNotFound) {
				return err
			}
		} else {
			if c, err = gr.CommitObject(tag.Target); err != nil {
				return err
			}
		}

		if c == nil {
			if c, err = gr.CommitObject(r.Hash()); err != nil {
				return err
			}
		}

		data.Tags = append(data.Tags, row{
			Name: r.Name().Short(), Message: strings.SplitN(c.Message, "\n", 2)[0], Author: c.Author.Name,
			LastCommit: c.Author.When.UTC().Format(time.DateTime), Hash: r.Hash().String(),
		})

		return nil
	}); err != nil {
		log.Println("[/repo/refs]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	slices.Reverse(data.Tags)

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/refs", data); err != nil {
		log.Println("[/repo/refs]", err.Error())
	}
}
