// user.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type User struct {
	Id       int64
	Name     string
	FullName string
	Pass     []byte
	PassAlgo string
	Salt     []byte
	IsAdmin  bool
}

func HandleUserLogout(w http.ResponseWriter, r *http.Request) {
	id, s := GetSessionCookie(r)
	EndSession(id, s.Token)
	EndSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func GetUser(id int64) (*User, error) {
	u := User{}

	if err := db.QueryRow(
		"SELECT id, name, name_full, pass, pass_algo, salt, is_admin FROM users WHERE id = ?", id,
	).Scan(&u.Id, &u.Name, &u.FullName, &u.Pass, &u.PassAlgo, &u.Salt, &u.IsAdmin); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("[SELECT:user] %w", err)
		} else {
			return nil, nil
		}
	} else {
		return &u, nil
	}
}

func GetUserByName(name string) (*User, error) {
	u := &User{}

	err := db.QueryRow(
		"SELECT id, name, name_full, pass, pass_algo, salt, is_admin FROM users WHERE name = ?", strings.ToLower(name),
	).Scan(&u.Id, &u.Name, &u.FullName, &u.Pass, &u.PassAlgo, &u.Salt, &u.IsAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return u, nil
}

func UserExists(name string) (bool, error) {
	if err := db.QueryRow("SELECT name FROM users WHERE name = ?", strings.ToLower(name)).Scan(&name); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		} else {
			return false, nil
		}
	} else {
		return true, nil
	}
}

func UpdateUser(uid int64, user User) error {
	if _, err := db.Exec(
		"UPDATE users SET name = ?, name_full = ? WHERE id = ?",
		user.Name, user.FullName, uid,
	); err != nil {
		return err
	}

	return nil
}

func UpdatePassword(uid int64, password string) error {
	salt, err := Salt()
	if err != nil {
		return err
	}

	if _, err := db.Exec(
		"UPDATE users SET pass = ?, pass_algo = ?, salt = ? WHERE id = ?",
		Hash(password, salt), "argon2", salt, uid,
	); err != nil {
		return err
	}

	return nil
}
