// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Jamozed/Goit/res"
	"github.com/Jamozed/Goit/src/cron"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	_ "github.com/mattn/go-sqlite3"
)

var Conf config
var db *sql.DB
var Favicon []byte
var Cron *cron.Cron

var Reserved []string = []string{"admin", "repo", "static", "user"}

var StartTime = time.Now()

func Goit() error {
	if conf, err := loadConfig(); err != nil {
		return err
	} else {
		Conf = conf
	}

	if err := os.MkdirAll(Conf.LogsPath, 0o777); err != nil {
		return fmt.Errorf("[config] %w", err)
	}

	logFile, err := os.Create(filepath.Join(Conf.LogsPath, fmt.Sprint("goit_", time.Now().Unix(), ".log")))
	if err != nil {
		log.Fatalln("[log]", err.Error())
	}

	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	log.Println("Starting Goit", res.Version)

	log.Println("[Config] using data path:", Conf.DataPath)
	if err := os.MkdirAll(Conf.DataPath, 0o777); err != nil {
		return fmt.Errorf("[config] %w", err)
	}

	if dat, err := os.ReadFile(filepath.Join(Conf.DataPath, "favicon.png")); err != nil {
		log.Println("[favicon]", err.Error())
	} else {
		Favicon = dat
	}

	if db, err = sql.Open("sqlite3", filepath.Join(Conf.DataPath, "goit.db")); err != nil {
		return fmt.Errorf("[database] %w", err)
	}

	/* Update the database if necessary */
	if err := dbUpdate(db); err != nil {
		return fmt.Errorf("[database] %w", err)
	}

	/* Create an admin user if one does not exist */
	if exists, err := UserExists("admin"); err != nil {
		log.Println("[admin:exists]", err.Error())
		err = nil /* ignored */
	} else if !exists {
		if salt, err := Salt(); err != nil {
			log.Println("[admin:salt]", err.Error())
			err = nil /* ignored */
		} else if _, err = db.Exec(
			"INSERT INTO users (id, name, name_full, pass, pass_algo, salt, is_admin) VALUES (?, ?, ?, ?, ?, ?, ?)",
			0, "admin", "Administrator", Hash("admin", salt), "argon2", salt, true,
		); err != nil {
			log.Println("[admin:INSERT]", err.Error())
			err = nil /* ignored */
		}
	}

	/* Initialise and start the cron service */
	Cron = cron.New()
	Cron.Start()

	/* Periodically clean up expired sessions */
	Cron.Add(-1, cron.Hourly, CleanupSessions)

	/* Add cron jobs for mirror repositories */
	repos, err := GetRepos()
	if err != nil {
		return err
	}

	for _, r := range repos {
		if r.IsMirror {
			util.Debugln("Adding mirror cron job for", r.Name)
			rid, name := r.Id, r.Name
			Cron.Add(r.Id, cron.Daily, func() {
				if err := Pull(rid); err != nil {
					log.Println("[cron:mirror]", rid, name, err.Error())
				} else {
					log.Println("[cron:mirror] updated", rid, name)
				}
			})
		}
	}

	Cron.Update()

	return nil
}

func RepoPath(name string, abs bool) string {
	return util.If(abs, filepath.Join(Conf.DataPath, "repos", name+".git"), filepath.Join(name+".git"))
}

func IsLegal(s string) bool {
	for i := 0; i < len(s); i += 1 {
		if !slices.Contains([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~/"), s[i]) {
			return false
		}
	}

	return true
}

func Backup() error {
	data := struct {
		Users []User `json:"users"`
		Repos []Repo `json:"repos"`
	}{}

	bdir := filepath.Join(Conf.DataPath, "backup")
	if err := os.MkdirAll(bdir, 0o777); err != nil {
		return err
	}

	/* Dump users */
	rows, err := db.Query("SELECT id, name, name_full, pass, pass_algo, salt, is_admin FROM users")
	if err != nil {
		return err
	}

	for rows.Next() {
		u := User{}
		if err := rows.Scan(&u.Id, &u.Name, &u.FullName, &u.Pass, &u.PassAlgo, &u.Salt, &u.IsAdmin); err != nil {
			return err
		}

		data.Users = append(data.Users, u)
	}
	rows.Close()

	/* Dump repositories */
	rows, err = db.Query(
		"SELECT id, owner_id, name, description, default_branch, upstream, visibility, is_mirror FROM repos",
	)
	if err != nil {
		return err
	}

	for rows.Next() {
		r := Repo{}
		if err := rows.Scan(
			&r.Id, &r.OwnerId, &r.Name, &r.Description, &r.DefaultBranch, &r.Upstream, &r.Visibility, &r.IsMirror,
		); err != nil {
			return err
		}

		data.Repos = append(data.Repos, r)
	}
	rows.Close()

	/* Open an output ZIP file */
	ts := "goit_" + time.Now().UTC().Format("20060102T150405Z")

	zf, err := os.Create(filepath.Join(bdir, ts+".zip"))
	if err != nil {
		return err
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	/* Copy repositories to ZIP */
	td, err := os.MkdirTemp(os.TempDir(), "goit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(td)

	for _, r := range data.Repos {
		cd := filepath.Join(td, RepoPath(r.Name, false))

		gr, err := git.PlainClone(cd, true, &git.CloneOptions{
			URL: RepoPath(r.Name, true), Mirror: true,
		})
		if err != nil {
			if errors.Is(err, transport.ErrRepositoryNotFound) {
				continue
			}

			if errors.Is(err, transport.ErrEmptyRemoteRepository) {
				continue
			}

			return err
		}

		if err := gr.DeleteRemote("origin"); err != nil {
			return fmt.Errorf("%s %w", cd, err)
		}

		/* Walk duplicated repository and add it to the ZIP */
		if err = filepath.WalkDir(cd, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			head, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			head.Name = filepath.Join(ts, strings.TrimPrefix(path, Conf.DataPath))

			if d.IsDir() {
				head.Name += "/"
			} else {
				head.Method = zip.Store
			}

			w, err := zw.CreateHeader(head)
			if err != nil {
				return err
			}

			if !d.IsDir() {
				fi, err := os.Open(path)
				if err != nil {
					return err
				}

				if _, err := io.Copy(w, fi); err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			return err
		}

		os.RemoveAll(cd)
	}

	/* Write database as JSON to ZIP */
	if b, err := json.MarshalIndent(data, "", "\t"); err != nil {
		return err
	} else if w, err := zw.Create(filepath.Join(ts, "goit.json")); err != nil {
		return err
	} else if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}
