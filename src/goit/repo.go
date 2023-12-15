// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
)

type Repo struct {
	Id          int64  `json:"id"`
	OwnerId     int64  `json:"owner_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Upstream    string `json:"upstream"`
	IsPrivate   bool   `json:"is_private"`
	IsMirror    bool   `json:"is_mirror"`
}

func GetRepos() ([]Repo, error) {
	repos := []Repo{}

	rows, err := db.Query("SELECT id, owner_id, name, description, upstream, is_private, is_mirror FROM repos")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		r := Repo{}
		if err := rows.Scan(
			&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.Upstream, &r.IsPrivate, &r.IsMirror,
		); err != nil {
			return nil, err
		}

		repos = append(repos, r)
	}

	if rows.Err() != nil {
		return nil, err
	}

	return repos, nil
}

func GetRepo(rid int64) (*Repo, error) {
	r := &Repo{}

	if err := db.QueryRow(
		"SELECT id, owner_id, name, description, upstream, is_private, is_mirror FROM repos WHERE id = ?", rid,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.Upstream, &r.IsPrivate, &r.IsMirror); err != nil {
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
		"SELECT id, owner_id, name, description, upstream, is_private, is_mirror FROM repos WHERE name = ?", name,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.Upstream, &r.IsPrivate, &r.IsMirror); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		return nil, nil
	}

	return r, nil
}

func CreateRepo(repo Repo) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return -1, err
	}

	res, err := tx.Exec(
		`INSERT INTO repos (owner_id, name, name_lower, description, upstream, is_private, is_mirror)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, repo.OwnerId, repo.Name, strings.ToLower(repo.Name), repo.Description,
		repo.Upstream, repo.IsPrivate, repo.IsMirror,
	)
	if err != nil {
		tx.Rollback()
		return -1, err
	}

	r, err := git.PlainInit(RepoPath(repo.Name, true), true)
	if err != nil {
		tx.Rollback()
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		os.RemoveAll(RepoPath(repo.Name, true))
		return -1, err
	}

	if repo.Upstream != "" {
		if _, err := r.CreateRemote(&gitconfig.RemoteConfig{
			Name:   "origin",
			URLs:   []string{repo.Upstream},
			Mirror: util.If(repo.IsMirror, true, false),
			Fetch:  []gitconfig.RefSpec{gitconfig.RefSpec("+refs/heads/*:refs/heads/*")},
		}); err != nil {
			log.Println("[repo/upstream]", err.Error())
		}
	}

	rid, _ := res.LastInsertId()

	return rid, nil
}

func DelRepo(rid int64) error {
	repo, err := GetRepo(rid)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(RepoPath(repo.Name, true)); err != nil {
		return err
	}

	if _, err := db.Exec("DELETE FROM repos WHERE id = ?", rid); err != nil {
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

func UpdateRepo(rid int64, repo Repo) error {
	old, err := GetRepo(rid)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(
		`UPDATE repos SET name = ?, name_lower = ?, description = ?, upstream = ?, is_private = ?, is_mirror = ?
		WHERE id = ?`, repo.Name, strings.ToLower(repo.Name), repo.Description, repo.Upstream, repo.IsPrivate,
		repo.IsMirror, rid,
	); err != nil {
		tx.Rollback()
		return err
	}

	if repo.Name != old.Name {
		if err := os.Rename(RepoPath(old.Name, true), RepoPath(repo.Name, true)); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		os.Rename(RepoPath(repo.Name, true), RepoPath(old.Name, true))
		log.Println("[repo/update]", "error while renaming, check repo \""+old.Name+"\"/\""+repo.Name+"\"")
		return err
	}

	return nil
}

func ChownRepo(rid int64, uid int64) error {
	if _, err := db.Exec("UPDATE repos SET owner_id = ? WHERE id = ?", uid, rid); err != nil {
		return err
	}

	return nil
}

func Pull(rid int64) error {
	repo, err := GetRepo(rid)
	if err != nil {
		return err
	}

	r, err := git.PlainOpen(RepoPath(repo.Name, true))
	if err != nil {
		return err
	}

	if err := r.Fetch(&git.FetchOptions{}); err != nil {
		return err
	}

	return nil
}
