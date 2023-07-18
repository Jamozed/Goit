// repo.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jamozed/Goit/res"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

type Repo struct {
	Id            uint64
	OwnerId       uint64
	Name          string
	NameLower     string
	Description   string
	DefaultBranch string
	IsPrivate     bool
}

var (
	repoIndex  *template.Template = template.Must(template.New("repo_index").Parse(res.RepoIndex))
	repoCreate *template.Template = template.Must(template.New("repo_create").Parse(res.RepoCreate))
	repoLog    *template.Template = template.Must(template.New("repo_log").Parse(res.RepoLog))
	repoTree   *template.Template = template.Must(template.New("repo_tree").Parse(res.RepoTree))
	repoRefs   *template.Template = template.Must(template.New("repo_refs").Parse(res.RepoRefs))
)

func (g *Goit) HandleIndex(w http.ResponseWriter, r *http.Request) {
	authOk, uid := AuthHttp(r)

	if rows, err := g.db.Query("SELECT id, owner_id, name, description, is_private FROM repos"); err != nil {
		log.Println("[Index:SELECT]", err.Error())
		HttpError(w, http.StatusInternalServerError)
	} else {
		defer rows.Close()

		type row struct{ Name, Description, Owner, Visibility, LastCommit string }
		repos := []row{}

		for rows.Next() {
			r := Repo{}

			if err := rows.Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.IsPrivate); err != nil {
				log.Println("[Index:SELECT:Scan]", err.Error())
			} else if !r.IsPrivate || (authOk && uid == r.OwnerId) {
				owner, err := g.GetUser(r.OwnerId)
				if err != nil {
					log.Println("[Index:SELECT:UserName]", err.Error())
				}

				repos = append(repos, row{r.Name, "", owner.Name, If(r.IsPrivate, "private", "public"), ""})
			}
		}

		if err := rows.Err(); err != nil {
			log.Println("[Index:SELECT:Err]", err.Error())
			HttpError(w, http.StatusInternalServerError)
		} else {
			repoIndex.Execute(w, struct{ Repos []row }{repos})
		}
	}
}

func (g *Goit) HandleRepoCreate(w http.ResponseWriter, r *http.Request) {
	if ok, uid := AuthHttp(r); !ok {
		HttpError(w, http.StatusUnauthorized)
	} else if r.Method == http.MethodPost {
		name := r.FormValue("reponame")
		private := r.FormValue("visibility") == "private"

		if taken, err := RepoExists(g.db, name); err != nil {
			log.Println("[RepoCreate:RepoExists]", err.Error())
			HttpError(w, http.StatusInternalServerError)
		} else if taken {
			repoCreate.Execute(w, struct{ Msg string }{"Reponame is taken"})
		} else if SliceContains[string](reserved, name) {
			repoCreate.Execute(w, struct{ Msg string }{"Reponame is reserved"})
		} else {
			if _, err := g.db.Exec(
				`INSERT INTO repos (
					owner_id, name, name_lower, description, default_branch, is_private
				) VALUES (?, ?, ?, ?, ?, ?)`,
				uid, name, strings.ToLower(name), "", "master", private,
			); err != nil {
				log.Println("[RepoCreate:INSERT]", err.Error())
				HttpError(w, http.StatusInternalServerError)
			} else {
				http.Redirect(w, r, "/"+name+"/", http.StatusFound)
			}
		}
	} else /* GET */ {
		repoCreate.Execute(w, nil)
	}
}

func (g *Goit) HandleRepoLog(w http.ResponseWriter, r *http.Request) {
	reponame := mux.Vars(r)["repo"]

	repo, err := GetRepoByName(db, reponame)
	if err != nil {
		HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil {
		HttpError(w, http.StatusNotFound)
		return
	}

	type row struct{ Date, Message, Author string }
	commits := []row{}

	if gr, err := git.PlainOpen("./" + reponame + ".git"); err != nil {
		log.Println("[Repo:Log]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if ref, err := gr.Head(); err != nil {
		log.Println("[Repo:Log]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if iter, err := gr.Log(&git.LogOptions{From: ref.Hash()}); err != nil {
		log.Println("[Repo:Log]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, row{c.Author.When.UTC().Format(time.DateTime), strings.SplitN(c.Message, "\n", 2)[0], c.Author.Name})
		return nil
	}); err != nil {
		log.Println("[Repo:Log]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := repoLog.Execute(w, struct {
		Name, Description, Url string
		HasReadme, HasLicence  bool
		Commits                []row
	}{reponame, repo.Description, r.URL.Host + "/" + repo.Name + ".git", false, false, commits}); err != nil {
		log.Println("[Repo:Log]", err.Error())
	}
}

func (g *Goit) HandleRepoTree(w http.ResponseWriter, r *http.Request) {
	HttpError(w, http.StatusNoContent)
}

func (g *Goit) HandleRepoRefs(w http.ResponseWriter, r *http.Request) {
	reponame := mux.Vars(r)["repo"]

	repo, err := GetRepoByName(db, reponame)
	if err != nil {
		HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil {
		HttpError(w, http.StatusNotFound)
		return
	}

	type bra struct{ Name, Hash string }
	type tag struct{ Name, Hash string }
	bras := []bra{}
	tags := []tag{}

	if gr, err := git.PlainOpen("./" + reponame + ".git"); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if iter, err := gr.Branches(); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(b *plumbing.Reference) error {
		bras = append(bras, bra{b.Name().Short(), b.Hash().String()})
		return nil
	}); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if iter, err := gr.Tags(); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	} else if err := iter.ForEach(func(b *plumbing.Reference) error {
		tags = append(tags, tag{b.Name().Short(), b.Hash().String()})
		return nil
	}); err != nil {
		log.Println("[Repo:Refs]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := repoRefs.Execute(w, struct {
		Name, Description, Url string
		HasReadme, HasLicence  bool
		Branches               []bra
		Tags                   []tag
	}{reponame, repo.Description, r.URL.Host + "/" + repo.Name + ".git", false, false, bras, tags}); err != nil {
		log.Println("[Repo:Refs]", err.Error())
	}
}

func GetRepoByName(db *sql.DB, name string) (*Repo, error) {
	r := &Repo{}

	err := db.QueryRow(
		"SELECT id, owner_id, name, name_lower, description, default_branch, is_private FROM repos WHERE name = ?", name,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.NameLower, &r.Description, &r.DefaultBranch, &r.IsPrivate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return r, nil
}

func RepoExists(db *sql.DB, name string) (bool, error) {
	if err := db.QueryRow(
		"SELECT name FROM repos WHERE name_lower = ?", strings.ToLower(name),
	).Scan(&name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		} else {
			return false, nil
		}
	} else {
		return true, nil
	}
}
