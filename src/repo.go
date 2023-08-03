// repo.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gorilla/mux"
)

type Repo struct {
	Id            int64
	OwnerId       int64
	Name          string
	Description   string
	DefaultBranch string
	IsPrivate     bool
}

func HandleRepoRefs(w http.ResponseWriter, r *http.Request) {
	reponame := mux.Vars(r)["repo"]

	repo, err := GetRepoByName(reponame)
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

	if gr, err := git.PlainOpen(RepoPath(reponame)); err != nil {
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

	if err := Tmpl.ExecuteTemplate(w, "repo/refs", struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Branches                      []bra
		Tags                          []tag
	}{
		"Refs", reponame, repo.Description, util.If(Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		"", "", bras, tags,
	}); err != nil {
		log.Println("[Repo:Refs]", err.Error())
	}
}

func GetRepo(id int64) (*Repo, error) {
	r := &Repo{}

	if err := db.QueryRow(
		"SELECT id, owner_id, name, description, default_branch, is_private FROM repos WHERE id = ?", id,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.DefaultBranch, &r.IsPrivate); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		return nil, nil
	} else {
		return r, nil
	}
}

func GetRepoByName(name string) (*Repo, error) {
	r := &Repo{}

	if err := db.QueryRow(
		"SELECT id, owner_id, name, description, default_branch, is_private FROM repos WHERE name = ?", name,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.DefaultBranch, &r.IsPrivate); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		return nil, nil
	}

	return r, nil
}

func CreateRepo(repo Repo) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO repos (owner_id, name, name_lower, description, default_branch, is_private)
		VALUES (?, ?, ?, ?, ?, ?)`,
		repo.OwnerId, repo.Name, strings.ToLower(repo.Name), repo.Description, repo.DefaultBranch, repo.IsPrivate,
	); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := git.PlainInit(RepoPath(repo.Name), true); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		os.RemoveAll(RepoPath(repo.Name))
		return err
	}

	return nil
}

func DelRepo(name string) error {
	if err := os.RemoveAll(RepoPath(name)); err != nil {
		return err
	}

	if _, err := db.Exec("DELETE FROM repos WHERE name = ?", name); err != nil {
		return err
	}

	return nil
}

func RepoExists(name string) (bool, error) {
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
