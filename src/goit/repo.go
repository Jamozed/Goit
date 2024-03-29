// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type Repo struct {
	Id            int64      `json:"id"`
	OwnerId       int64      `json:"owner_id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	DefaultBranch string     `json:"default_branch"`
	Upstream      string     `json:"upstream"`
	Visibility    Visibility `json:"visibility"`
	IsMirror      bool       `json:"is_mirror"`
}

type Visibility int32

const (
	Public  Visibility = 0
	Private Visibility = 1
	Limited Visibility = 2
)

func VisibilityFromString(s string) Visibility {
	switch strings.ToLower(s) {
	case "public":
		return Public
	case "private":
		return Private
	case "limited":
		return Limited
	default:
		return -1
	}
}

func (v Visibility) String() string {
	return [...]string{"public", "private", "limited"}[v]
}

func GetRepos() ([]Repo, error) {
	repos := []Repo{}

	rows, err := db.Query(
		"SELECT id, owner_id, name, description, default_branch, upstream, visibility, is_mirror FROM repos",
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		r := Repo{}
		if err := rows.Scan(
			&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.DefaultBranch, &r.Upstream, &r.Visibility, &r.IsMirror,
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
		`SELECT id, owner_id, name, description, default_branch, upstream, visibility, is_mirror FROM repos
		WHERE id = ?`, rid,
	).Scan(
		&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.DefaultBranch, &r.Upstream, &r.Visibility, &r.IsMirror,
	); err != nil {
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
		`SELECT id, owner_id, name, description, default_branch, upstream, visibility, is_mirror FROM repos
		WHERE name = ?`, name,
	).Scan(
		&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.DefaultBranch, &r.Upstream, &r.Visibility, &r.IsMirror,
	); err != nil {
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
		`INSERT INTO repos (owner_id, name, name_lower, description, default_branch, upstream, visibility, is_mirror)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, repo.OwnerId, repo.Name, strings.ToLower(repo.Name), repo.Description,
		repo.DefaultBranch, repo.Upstream, repo.Visibility, repo.IsMirror,
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

	ref := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(repo.DefaultBranch))
	if err := r.Storer.SetReference(ref); err != nil {
		tx.Rollback()
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
			tx.Rollback()
			os.RemoveAll(RepoPath(repo.Name, true))
			return -1, err
		}
	}

	if err := tx.Commit(); err != nil {
		os.RemoveAll(RepoPath(repo.Name, true))
		return -1, err
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

	Cron.RemoveFor(rid)
	Cron.Update()

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
		`UPDATE repos SET name = ?, name_lower = ?, description = ?, default_branch = ?, upstream = ?, visibility = ?,
		is_mirror = ? WHERE id = ?`, repo.Name, strings.ToLower(repo.Name), repo.Description, repo.DefaultBranch,
		repo.Upstream, repo.Visibility, repo.IsMirror, rid,
	); err != nil {
		tx.Rollback()
		return err
	}

	if repo.Name != old.Name {
		if err := os.MkdirAll(filepath.Dir(RepoPath(repo.Name, true)), 0o777); err != nil {
			tx.Rollback()
			return err
		}

		if err := os.Rename(RepoPath(old.Name, true), RepoPath(repo.Name, true)); err != nil {
			tx.Rollback()
			return err
		}
	}

	var r *git.Repository
	if repo.DefaultBranch != old.DefaultBranch {
		// if r == nil {
		r, err = git.PlainOpen(RepoPath(repo.Name, true))
		if err != nil {
			tx.Rollback()
			return err
		}
		// }

		ref := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(repo.DefaultBranch))
		if err := r.Storer.SetReference(ref); err != nil {
			tx.Rollback()
			return err
		}
	}

	/* If the upstream URL has been removed, remove the remote */
	if repo.Upstream == "" && old.Upstream != "" {
		if r == nil {
			r, err = git.PlainOpen(RepoPath(repo.Name, true))
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := r.DeleteRemote("origin"); err != nil {
			tx.Rollback()
			return err
		}
	}

	/* If the upstream URL has been added or changed, update the remote */
	if repo.Upstream != "" && repo.Upstream != old.Upstream {
		if r == nil {
			r, err = git.PlainOpen(RepoPath(repo.Name, true))
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := r.DeleteRemote("origin"); err != nil && !errors.Is(err, git.ErrRemoteNotFound) {
			tx.Rollback()
			return err
		}

		if _, err := r.CreateRemote(&gitconfig.RemoteConfig{
			Name:   "origin",
			URLs:   []string{repo.Upstream},
			Mirror: util.If(repo.IsMirror, true, false),
			Fetch:  []gitconfig.RefSpec{gitconfig.RefSpec("+refs/heads/*:refs/heads/*")},
		}); err != nil {
			log.Println("[repo/update]", err.Error())
		}
	}

	if err := tx.Commit(); err != nil {
		os.Rename(RepoPath(repo.Name, true), RepoPath(old.Name, true))
		log.Println("[repo/update]", "error while editing, check repo \""+old.Name+"\"/\""+repo.Name+"\"")
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

func IsVisible(repo *Repo, auth bool, user *User) bool {
	if repo.Visibility == Public || (repo.Visibility == Limited && auth) || (auth && user.Id == repo.OwnerId) {
		return true
	}

	return false
}
