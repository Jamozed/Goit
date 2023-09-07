// repo.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Repo struct {
	Id          int64
	OwnerId     int64
	Name        string
	Description string
	IsPrivate   bool
}

func GetRepo(id int64) (*Repo, error) {
	r := &Repo{}

	if err := db.QueryRow(
		"SELECT id, owner_id, name, description, is_private FROM repos WHERE id = ?", id,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.IsPrivate); err != nil {
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
		"SELECT id, owner_id, name, description, is_private FROM repos WHERE name = ?", name,
	).Scan(&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.IsPrivate); err != nil {
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
		`INSERT INTO repos (owner_id, name, name_lower, description, is_private)
		VALUES (?, ?, ?, ?, ?)`,
		repo.OwnerId, repo.Name, strings.ToLower(repo.Name), repo.Description, repo.IsPrivate,
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
		"UPDATE repos SET name = ?, name_lower = ?, description = ?, is_private = ? WHERE id = ?",
		repo.Name, strings.ToLower(repo.Name), repo.Description, repo.IsPrivate, rid,
	); err != nil {
		tx.Rollback()
		return err
	}

	if repo.Name != old.Name {
		/* TODO use a mutex lock or something to make sure this doesn't break */
		if err := os.Rename(RepoPath(old.Name), RepoPath(repo.Name)); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		os.Rename(RepoPath(repo.Name), RepoPath(old.Name))
		log.Println("[repo/update]", "error while renaming, check repo \""+old.Name+"\"/\""+repo.Name+"\"")
		return err
	}

	return nil
}
