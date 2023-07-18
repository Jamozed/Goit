// goit.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Goit struct {
	db *sql.DB
}

/* Initialise Goit. */
func InitGoit() (g *Goit, err error) {
	g = &Goit{}

	if g.db, err = sql.Open("sqlite3", "./goit.db"); err != nil {
		return nil, fmt.Errorf("[SQL:open] %w", err)
	}
	db = g.db

	if _, err = g.db.Exec(
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
		return nil, fmt.Errorf("[CREATE:users] %w", err)
	}

	if _, err = g.db.Exec(
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
		return nil, fmt.Errorf("[CREATE:repos] %w", err)
	}

	/* Create an admin user if one does not exist */
	if exists, err := g.UserExists("admin"); err != nil {
		log.Println("[admin:Exists]", err.Error())
		err = nil /* ignored */
	} else if !exists {
		if salt, err := Salt(); err != nil {
			log.Println("[admin:Salt]", err.Error())
			err = nil /* ignored */
		} else if _, err = g.db.Exec(
			"INSERT INTO users (id, name, name_full, pass, pass_algo, salt, is_admin) VALUES (?, ?, ?, ?, ?, ?, ?)",
			0, "admin", "Administrator", Hash("admin", salt), "argon2", salt, true,
		); err != nil {
			log.Println("[admin:INSERT]", err.Error())
			err = nil /* ignored */
		}
	}

	return g, nil
}

func (g *Goit) Close() error {
	return g.db.Close()
}
