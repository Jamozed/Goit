// goit.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/adrg/xdg"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	DataPath   string `json:"data_path"`
	HttpAddr   string `json:"http_addr"`
	HttpPort   string `json:"http_port"`
	GitPath    string `json:"git_path"`
	IpSessions bool   `json:"ip_sessions"`
	UsesHttps  bool   `json:"uses_https"`
}

var Conf = Config{
	DataPath:   path.Join(xdg.DataHome, "goit"),
	HttpAddr:   "",
	HttpPort:   "8080",
	GitPath:    "git",
	IpSessions: true,
	UsesHttps:  false,
}

var db *sql.DB
var Favicon []byte

func Goit(conf string) (err error) {
	if dat, err := os.ReadFile(conf); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("[Config] %w", err)
		}
	} else if dat != nil {
		if json.Unmarshal(dat, &Conf); err != nil {
			return fmt.Errorf("[Config] %w", err)
		}
	}

	log.Println("[Config] using data path:", Conf.DataPath)
	if err := os.MkdirAll(Conf.DataPath, 0o777); err != nil {
		return fmt.Errorf("[Config] %w", err)
	}

	if dat, err := os.ReadFile(path.Join(Conf.DataPath, "favicon.png")); err != nil {
		log.Println("[Favicon]", err.Error())
	} else {
		Favicon = dat
	}

	if db, err = sql.Open("sqlite3", path.Join(Conf.DataPath, "goit.db")); err != nil {
		return fmt.Errorf("[Database] %w", err)
	}

	if _, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			name_full TEXT UNIQUE NOT NULL,
			pass BLOB NOT NULL,
			pass_algo TEXT NOT NULL,
			salt BLOB NOT NULL,
			is_admin BOOLEAN NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf("[CREATE:users] %w", err)
	}

	if _, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS repos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_id INTEGER NOT NULL,
			name TEXT UNIQUE NOT NULL,
			name_lower TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL,
			is_private BOOLEAN NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf("[CREATE repos] %w", err)
	}

	/* Create an admin user if one does not exist */
	if exists, err := UserExists("admin"); err != nil {
		log.Println("[admin Exists]", err.Error())
		err = nil /* ignored */
	} else if !exists {
		if salt, err := Salt(); err != nil {
			log.Println("[admin Salt]", err.Error())
			err = nil /* ignored */
		} else if _, err = db.Exec(
			"INSERT INTO users (id, name, name_full, pass, pass_algo, salt, is_admin) VALUES (?, ?, ?, ?, ?, ?, ?)",
			0, "admin", "Administrator", Hash("admin", salt), "argon2", salt, true,
		); err != nil {
			log.Println("[admin INSERT]", err.Error())
			err = nil /* ignored */
		}
	}

	return nil
}

func ConfPath() string {
	if p, err := xdg.SearchConfigFile(path.Join("goit", "goit.json")); err != nil {
		log.Println("[Config]", err.Error())
		return ""
	} else {
		return p
	}
}

func RepoPath(name string) string {
	return path.Join(Conf.DataPath, "repos", name+".git")
}
