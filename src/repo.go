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

	"github.com/Jamozed/Goit/res"
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
	repoIndex  *template.Template
	repoCreate *template.Template
)

func init() {
	repoIndex = template.Must(template.New("repo_index").Parse(res.RepoIndex))
	repoCreate = template.Must(template.New("repo_create").Parse(res.RepoCreate))
}

func (g *Goit) HandleIndex(w http.ResponseWriter, r *http.Request) {
	authOk, uid := AuthHttp(r)

	if rows, err := g.db.Query("SELECT id, owner_id, name, description, is_private FROM repos"); err != nil {
		log.Println("[Index:SELECT]", err.Error())
		http.Error(w, "500 internal server error", http.StatusInternalServerError)
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
			http.Error(w, "500 internal server error", http.StatusInternalServerError)
		} else {
			repoIndex.Execute(w, struct{ Repos []row }{repos})
		}
	}
}

func (g *Goit) HandleRepoCreate(w http.ResponseWriter, r *http.Request) {
	if ok, uid := AuthHttp(r); !ok {
		http.Error(w, "401 unauthorized", http.StatusUnauthorized)
	} else if r.Method == http.MethodPost {
		name := r.FormValue("reponame")
		private := r.FormValue("visibility") == "private"

		if taken, err := RepoExists(g.db, name); err != nil {
			log.Println("[RepoCreate:RepoExists]", err.Error())
			http.Error(w, "500 internal server error", http.StatusInternalServerError)
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
				http.Error(w, "500 internal server error", http.StatusInternalServerError)
			} else {
				http.Redirect(w, r, "/"+name+"/", http.StatusFound)
			}
		}
	} else /* GET */ {
		repoCreate.Execute(w, nil)
	}
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
