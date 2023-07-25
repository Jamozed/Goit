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
	"path/filepath"
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

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	auth, admin, uid := AuthCookieAdmin(w, r, true)

	user, err := GetUser(uid)
	if err != nil {
		log.Println("[/]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if rows, err := db.Query("SELECT id, owner_id, name, description, is_private FROM repos"); err != nil {
		log.Println("[/]", err.Error())
		HttpError(w, http.StatusInternalServerError)
	} else {
		defer rows.Close()

		type row struct{ Name, Description, Owner, Visibility, LastCommit string }
		data := struct {
			Title, Username string
			Admin, Auth     bool
			Repos           []row
		}{Title: "Repositories", Admin: admin, Auth: auth}

		if user != nil {
			data.Username = user.Name
		}

		for rows.Next() {
			d := Repo{}
			if err := rows.Scan(&d.Id, &d.OwnerId, &d.Name, &d.Description, &d.IsPrivate); err != nil {
				log.Println("[/]", err.Error())
			} else if !d.IsPrivate || (auth && uid == d.OwnerId) {
				owner, err := GetUser(d.OwnerId)
				if err != nil {
					log.Println("[/]", err.Error())
				}

				data.Repos = append(data.Repos, row{
					d.Name, d.Description, owner.Name, util.If(d.IsPrivate, "private", "public"), "",
				})
			}
		}

		if err := rows.Err(); err != nil {
			log.Println("[/]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		}

		if err := Tmpl.ExecuteTemplate(w, "index", data); err != nil {
			log.Println("[/]", err.Error())
		}
	}
}

func HandleRepoCreate(w http.ResponseWriter, r *http.Request) {
	ok, uid := AuthCookie(w, r, true)
	if !ok {
		HttpError(w, http.StatusUnauthorized)
		return
	}

	data := struct{ Title, Message string }{Title: "Repository - Create"}

	if r.Method == http.MethodPost {
		reponame := r.FormValue("reponame")
		description := r.FormValue("description")
		private := r.FormValue("visibility") == "private"

		if reponame == "" {
			data.Message = "Name cannot be empty"
		} else if util.SliceContains(reserved, reponame) {
			data.Message = "Name \"" + reponame + "\" is reserved"
		} else if exists, err := RepoExists(reponame); err != nil {
			log.Println("[/repo/create]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else if exists {
			data.Message = "Name \"" + reponame + "\" already exists"
		} else if _, err := db.Exec(
			`INSERT INTO repos (owner_id, name, name_lower, description, default_branch, is_private)
			VALUES (?, ?, ?, ?, ?, ?)`,
			uid, reponame, strings.ToLower(reponame), description, "master", private,
		); err != nil {
			log.Println("[/repo/create]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else {
			http.Redirect(w, r, "/"+reponame, http.StatusFound)
			return
		}
	}

	if err := Tmpl.ExecuteTemplate(w, "repo/create", data); err != nil {
		log.Println("[/repo/create]", err.Error())
	}
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

func RepoSize(name string) (uint64, error) {
	var size int64

	err := filepath.WalkDir(RepoPath(name), func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			} else {
				return err
			}
		}

		if d.IsDir() {
			return nil
		}

		f, err := d.Info()
		if err != nil {
			return err
		}

		/* Only count the size of regular files */
		if (f.Mode() & (os.ModeSymlink | os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice | os.ModeIrregular)) == 0 {
			size += f.Size()
		}

		return nil
	})

	return uint64(size), err
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
