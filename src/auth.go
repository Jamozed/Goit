// auth.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"golang.org/x/crypto/argon2"
)

type session struct {
	id     uint64
	expiry time.Time
}

var sessions = map[string]session{}

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

func NewSession(id uint64, expiry time.Time) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	s := base64.StdEncoding.EncodeToString(b)
	sessions[s] = session{id, expiry}
	return s, nil
}

func EndSession(s string) {
	delete(sessions, s)
}

func Auth(s string) (bool, uint64) {
	if v, ok := sessions[s]; ok {
		if v.expiry.After(time.Now()) {
			return true, v.id
		} else {
			delete(sessions, s)
		}
	}

	return false, math.MaxUint64
}

func AuthHttp(r *http.Request) (bool, uint64) {
	if c := Cookie(r, "session"); c != nil {
		return Auth(c.Value)
	}

	return false, math.MaxUint64
}

func SessionCookie(r *http.Request) string {
	if c := Cookie(r, "session"); c != nil {
		return c.Value
	}

	return ""
}

func GetSessions() (s string) {
	for k, v := range sessions {
		s += fmt.Sprint(k, v.id, v.expiry)
	}

	return s
}

func CleanupSessions() {
	n := 0

	for k, v := range sessions {
		if v.expiry.Before(time.Now()) {
			delete(sessions, k)
			n += 1
		}
	}

	if n > 0 {
		log.Println("[Sessions] Cleaned up", n, "expired sessions")
	}
}
