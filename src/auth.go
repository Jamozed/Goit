// auth.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jamozed/Goit/src/util"
	"golang.org/x/crypto/argon2"
)

type Session struct {
	Token, Ip    string
	Seen, Expiry time.Time
}

var Sessions = map[int64][]Session{}

func NewSession(uid int64, ip string, expiry time.Time) (Session, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return Session{}, err
	}

	if Sessions[uid] == nil {
		Sessions[uid] = []Session{}
	}

	t := base64.StdEncoding.EncodeToString(b)
	s := Session{Token: t, Ip: util.If(Conf.IpSessions, ip, ""), Seen: time.Now(), Expiry: expiry}

	Sessions[uid] = append(Sessions[uid], s)
	return s, nil
}

func EndSession(uid int64, token string) {
	for i, t := range Sessions[uid] {
		if t.Token == token {
			Sessions[uid] = append(Sessions[uid][:i], Sessions[uid][i+1:]...)
			break
		}
	}

	if len(Sessions[uid]) == 0 {
		delete(Sessions, uid)
	}
}

func CleanupSessions() {
	var n uint64 = 0

	for k, v := range Sessions {
		for _, v1 := range v {
			if v1.Expiry.Before(time.Now()) {
				EndSession(k, v1.Token)
				n += 1
			}
		}
	}

	if n > 0 {
		log.Println("[Cleanup] cleaned up", n, "expired sessions")
	}
}

func SetSessionCookie(w http.ResponseWriter, uid int64, s Session) {
	c := &http.Cookie{Name: "session", Value: fmt.Sprint(uid) + "." + s.Token, Path: "/", Expires: s.Expiry}
	if err := c.Valid(); err != nil {
		log.Println("[Cookie]", err.Error())
	}

	http.SetCookie(w, c)
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

		for i, s := range Sessions[id] {
			if ss[1] == s.Token {
				if s != (Session{}) {
					s.Seen = time.Now()
					Sessions[id][i] = s
				}

				return id, s
			}
		}

		return id, Session{}
	}

	return -1, Session{}
}

func EndSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", MaxAge: -1})
}

func AuthCookie(w http.ResponseWriter, r *http.Request, renew bool) (bool, int64) {
	if uid, s := GetSessionCookie(r); s != (Session{}) {
		if s.Expiry.After(time.Now()) {
			if renew && time.Until(s.Expiry) < 24*time.Hour {
				ip, _, _ := net.SplitHostPort(r.RemoteAddr)
				s1, err := NewSession(uid, ip, time.Now().Add(2*24*time.Hour))
				if err != nil {
					log.Println("[Renew Auth]", err.Error())
				} else {
					SetSessionCookie(w, uid, s1)
					EndSession(uid, s.Token)
				}
			}

			return true, uid
		}

		EndSession(uid, s.Token)
	}

	return false, -1
}

func AuthCookieAdmin(w http.ResponseWriter, r *http.Request, renew bool) (bool, bool, int64) {
	if ok, uid := AuthCookie(w, r, renew); ok {
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
