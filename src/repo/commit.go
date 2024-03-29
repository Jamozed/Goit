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
	"strconv"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/buildkite/terminal-to-html/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func HandleCommit(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[/repo/commit]", err.Error())
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

	type stat struct {
		Name, Path, Status, Num, Plusses, Minuses string
		IsBinary                                  bool
	}

	data := struct {
		HeaderFields
		Title                       string
		Author, Date, Commit        string
		Parents                     []string
		MessageSubject, MessageBody string
		Stats                       []stat
		Summary                     string
		Diff                        template.HTML
	}{
		Title:        repo.Name + " - Log",
		HeaderFields: GetHeaderFields(auth, user, repo, r.Host),
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/commit]", err.Error())
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

	commit, err := gr.CommitObject(plumbing.NewHash(chi.URLParam(r, "hash")))
	if errors.Is(err, plumbing.ErrObjectNotFound) {
		goit.HttpError(w, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("[/repo/commit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	data.Author = commit.Author.String()
	data.Date = commit.Author.When.UTC().Format(time.DateTime)
	data.Commit = commit.Hash.String()

	for _, h := range commit.ParentHashes {
		data.Parents = append(data.Parents, h.String())
	}

	data.MessageSubject, data.MessageBody, _ = strings.Cut(commit.Message, "\n")

	st, err := goit.DiffStats(commit)
	if err != nil {
		log.Println("[/repo/commit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	var files, additions, deletions int = len(st), 0, 0
	for _, s := range st {
		f := stat{Name: s.Name, Path: s.Name, Status: s.Status}
		f.Num = strconv.FormatInt(int64(s.Addition+s.Deletion), 10)

		if s.Addition+s.Deletion > 80 {
			f.Plusses = strings.Repeat("+", (s.Addition*80)/(s.Addition+s.Deletion))
			f.Minuses = strings.Repeat("-", (s.Deletion*80)/(s.Addition+s.Deletion))
		} else {
			f.Plusses = strings.Repeat("+", s.Addition)
			f.Minuses = strings.Repeat("-", s.Deletion)
		}

		if s.Status == "R" {
			f.Name = s.Prev + " -> " + s.Name
		}
		if s.IsBinary {
			f.IsBinary = true
		}

		data.Stats = append(data.Stats, f)

		additions += s.Addition
		deletions += s.Deletion
	}

	data.Summary = fmt.Sprintf("%d files changed, %d insertions, %d deletions", files, additions, deletions)

	var phash string
	if commit.NumParents() > 0 {
		phash = commit.ParentHashes[0].String()
	} else {
		phash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	}

	c := goit.NewGitCommand("diff", "--color=always", "-p", phash, commit.Hash.String())
	c.Dir = goit.RepoPath(repo.Name, true)
	out, _, err := c.Run(nil, nil)
	if err != nil {
		log.Println("[/repo/commit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	data.Diff = template.HTML(terminal.Render(out))

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/commit", data); err != nil {
		log.Println("[/repo/commit]", err.Error())
	}
}
