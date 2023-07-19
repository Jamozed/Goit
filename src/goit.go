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

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	DataPath string `json:"data_path"`
	HttpAddr string `json:"http_addr"`
	HttpPort string `json:"http_port"`
	GitPath  string `json:"git_path"`
}

var config = Config{
	DataPath: ".",
	HttpAddr: "",
	HttpPort: "8080",
	GitPath:  "git",
}

var db *sql.DB

/* Initialise Goit. */
func InitGoit(conf string) (err error) {
	if dat, err := os.ReadFile(conf); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("[Config] %w", err)
		}
	} else if dat != nil {
		if json.Unmarshal(dat, &config); err != nil {
			return fmt.Errorf("[Config] %w", err)
		}
	}

	if db, err = sql.Open("sqlite3", path.Join(config.DataPath, "goit.db")); err != nil {
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
			default_branch TEXT NOT NULL,
			is_private BOOLEAN NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf("[CREATE:repos] %w", err)
	}

	/* Create an admin user if one does not exist */
	if exists, err := UserExists("admin"); err != nil {
		log.Println("[admin:Exists]", err.Error())
		err = nil /* ignored */
	} else if !exists {
		if salt, err := Salt(); err != nil {
			log.Println("[admin:Salt]", err.Error())
			err = nil /* ignored */
		} else if _, err = db.Exec(
			"INSERT INTO users (id, name, name_full, pass, pass_algo, salt, is_admin) VALUES (?, ?, ?, ?, ?, ?, ?)",
			0, "admin", "Administrator", Hash("admin", salt), "argon2", salt, true,
		); err != nil {
			log.Println("[admin:INSERT]", err.Error())
			err = nil /* ignored */
		}
	}

	return nil
}

func GetRepoPath(username, reponame string) string {
	return path.Join(config.DataPath, "repos", username, reponame+".git")
}
