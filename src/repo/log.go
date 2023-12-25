// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const PAGE = 100

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

	offset := int64(0)
	if o := r.URL.Query().Get("o"); o != "" {
		if i, err := strconv.ParseInt(o, 10, 64); err != nil {
			goit.HttpError(w, http.StatusBadRequest)
			return
		} else {
			offset = i
		}
	}

	type row struct{ Hash, Date, Message, Author, Files, Additions, Deletions string }
	data := struct {
		HeaderFields
		Title                        string
		Commits                      []row
		Page, PrevOffset, NextOffset int64
	}{
		Title:        repo.Name + " - Log",
		HeaderFields: GetHeaderFields(auth, user, repo, r.Host),

		Page:       offset/PAGE + 1,
		PrevOffset: util.Max(offset-PAGE, -1),
		NextOffset: offset + PAGE,
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		data.NextOffset = 0
		goto execute
	} else if err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if readme, _ := findPattern(gr, ref, readmePattern); readme != "" {
		data.Readme = filepath.Join("/", repo.Name, "file", readme)
	}
	if licence, _ := findPattern(gr, ref, licencePattern); licence != "" {
		data.Licence = filepath.Join("/", repo.Name, "file", licence)
	}

	if iter, err := gr.Log(&git.LogOptions{
		From: ref.Hash(), Order: git.LogOrderCommitterTime, PathFilter: func(s string) bool {
			return tpath == "" || s == tpath || strings.HasPrefix(s, tpath+"/")
		},
	}); err != nil {
		log.Println("[/repo/log]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else {
		for i := int64(0); i < offset; i += 1 {
			if _, err := iter.Next(); err != nil {
				if errors.Is(err, io.EOF) {
					data.NextOffset = 0
					goto execute
				}

				log.Println("[/repo/log]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			}
		}

		for i := 0; i < PAGE; i += 1 {
			c, err := iter.Next()
			if errors.Is(err, io.EOF) {
				data.NextOffset = 0
				goto execute
			} else if err != nil {
				log.Println("[/repo/log]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			}

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
		}

		if _, err := iter.Next(); errors.Is(err, io.EOF) {
			data.NextOffset = 0
		}
	}

execute:
	if err := goit.Tmpl.ExecuteTemplate(w, "repo/log", data); err != nil {
		log.Println("[/repo/log]", err.Error())
	}
}
