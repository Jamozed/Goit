// auth.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/util"
	"golang.org/x/crypto/argon2"
)

type Session struct {
	Token, Ip string
	Expiry    time.Time
}

var Sessions = map[int64]map[string]Session{}

func NewSession(uid int64, ip string, expiry time.Time) (Session, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return Session{}, err
	}

	if Sessions[uid] == nil {
		Sessions[uid] = map[string]Session{}
	}

	t := base64.StdEncoding.EncodeToString(b)
	Sessions[uid][t] = Session{t, ip, expiry}
	return Sessions[uid][t], nil
}

func EndSession(id int64, token string) {
	delete(Sessions[id], token)
	if len(Sessions[id]) == 0 {
		delete(Sessions, id)
	}
}

func CleanupSessions() {
	var n uint64 = 0

	for k, v := range Sessions {
		for k1, v1 := range v {
			if v1.Expiry.Before(time.Now()) {
				EndSession(k, k1)
				n += 1
			}
		}
	}

	if n > 0 {
		log.Println("[Cleanup] cleaned up", n, "expired sessions")
	}
}

func SetSessionCookie(w http.ResponseWriter, uid int64, s Session) {
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: fmt.Sprint(uid) + "." + s.Token, Path: "/", Expires: s.Expiry,
	})
}

func GetSessionCookie(r *http.Request) (int64, Session) {
	if c := util.Cookie(r, "session"); c != nil {
		ss := strings.SplitN(c.Value, ".", 2)
		if len(ss) != 2 {
			return -1, Session{}
		}

		id, err := strconv.ParseInt(ss[0], 10, 64)
		if err != nil {
			return -1, Session{}
		}

		return id, Sessions[id][ss[1]]
	}

	return -1, Session{}
}

func EndSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", MaxAge: -1})
}

func AuthCookie(r *http.Request) (auth bool, uid int64) {
	if uid, s := GetSessionCookie(r); s != (Session{}) {
		if s.Expiry.After(time.Now()) {
			return true, uid
		}

		EndSession(uid, s.Token)
	}

	return false, -1
}

func AuthCookieAdmin(r *http.Request) (auth bool, admin bool, uid int64) {
	if ok, uid := AuthCookie(r); ok {
		if user, err := GetUser(uid); err == nil && user.IsAdmin {
			return true, true, uid
		}

		return true, false, uid
	}

	return false, false, -1
}

/* Hash a password with a salt using Argon2. */
func Hash(pass string, salt []byte) []byte {
	return argon2.IDKey([]byte(pass), salt, 3, 64*1024, 4, 32)
}

/* Generate a random Base64 salt. */
func Salt() ([]byte, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}
