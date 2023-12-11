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
	"sync"
	"time"

	"github.com/Jamozed/Goit/src/util"
	"golang.org/x/crypto/argon2"
)

type Session struct {
	Token, Ip    string
	Seen, Expiry time.Time
}

var Sessions = map[int64][]Session{}
var SessionsMutex = sync.RWMutex{}

/* Generate a new user session. */
func NewSession(uid int64, ip string, expiry time.Time) (Session, error) {
	var b = make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return Session{}, err
	}

	var t = base64.StdEncoding.EncodeToString(b)
	var s = Session{Token: t, Ip: util.If(Conf.IpSessions, ip, ""), Seen: time.Now(), Expiry: expiry}

	SessionsMutex.Lock()
	Debugln("[goit.NewSession] SessionsMutex lock")

	if Sessions[uid] == nil {
		Sessions[uid] = []Session{}
	}

	Sessions[uid] = append(Sessions[uid], s)

	SessionsMutex.Unlock()
	Debugln("[goit.EndSession] SessionsMutex unlock")

	return s, nil
}

/* End a user session. */
func EndSession(uid int64, token string) {
	SessionsMutex.Lock()
	Debugln("[goit.EndSession] SessionsMutex lock")
	defer SessionsMutex.Unlock()
	defer Debugln("[goit.EndSession] SessionsMutex unlock")

	if Sessions[uid] == nil {
		return
	}

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

/* Cleanup expired user sessions. */
func CleanupSessions() {
	var n int = 0

	SessionsMutex.Lock()
	Debugln("[goit.CleanupSessions] SessionsMutex lock")

	for uid, v := range Sessions {
		var i = 0
		for _, s := range v {
			if s.Expiry.After(time.Now()) {
				v[i] = s
				i += 1
			}
		}

		n += len(v) - i

		if i == 0 {
			delete(Sessions, uid)
		} else {
			Sessions[uid] = v[:i]
		}
	}

	SessionsMutex.Unlock()
	Debugln("[goit.CleanupSessions] SessionsMutex unlock")

	if n > 0 {
		log.Println("[Cleanup] cleaned up", n, "expired sessions")
	}
}

/* Set a user session cookie. */
func SetSessionCookie(w http.ResponseWriter, uid int64, s Session) {
	c := &http.Cookie{
		Name: "session", Value: fmt.Sprint(uid) + "." + s.Token, Path: "/", Expires: s.Expiry,
		Secure: util.If(Conf.UsesHttps, true, false), HttpOnly: true, SameSite: http.SameSiteLaxMode,
	}

	if err := c.Valid(); err != nil {
		log.Println("[Cookie]", err.Error())
	}

	http.SetCookie(w, c)
}

/* Get a user session cookie if one is present. */
func GetSessionCookie(r *http.Request) (int64, Session) {
	if c := util.Cookie(r, "session"); c != nil {
		ss := strings.SplitN(c.Value, ".", 2)
		if len(ss) != 2 {
			return -1, Session{}
		}

		uid, err := strconv.ParseInt(ss[0], 10, 64)
		if err != nil {
			return -1, Session{}
		}

		SessionsMutex.Lock()
		Debugln("[goit.GetSessionCookie] SessionsMutex lock")
		defer SessionsMutex.Unlock()
		defer Debugln("[goit.GetSessionCookie] SessionsMutex unlock")

		for i, s := range Sessions[uid] {
			if ss[1] == s.Token {
				if s != (Session{}) {
					s.Seen = time.Now()
					Sessions[uid][i] = s
				}

				return uid, s
			}
		}

		return uid, Session{}
	}

	return -1, Session{}
}

/* End the current user session cookie. */
func EndSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/", MaxAge: -1})
}

/* Authenticate a user session, returns auth, user, error. */
func Auth(w http.ResponseWriter, r *http.Request, renew bool) (bool, *User, error) {
	uid, s := GetSessionCookie(r)
	if s == (Session{}) {
		return false, nil, nil
	}

	/* Attempt to get the user associated with the session UID */
	user, err := GetUser(uid)
	if err != nil {
		return false, nil, fmt.Errorf("[auth] %w", err)
	}

	/* End invalid and expired sessions */
	if user == nil || s.Expiry.Before(time.Now()) {
		EndSession(uid, s.Token)
		return false, nil, nil
	}

	/* Renew the session if appropriate */
	if renew && time.Until(s.Expiry) < 24*time.Hour {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		s1, err := NewSession(uid, ip, time.Now().Add(2*24*time.Hour))
		if err != nil {
			log.Println("[auth/renew]", err.Error())
		} else {
			SetSessionCookie(w, uid, s1)
			EndSession(uid, s.Token)
		}
	}

	return true, user, nil
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
