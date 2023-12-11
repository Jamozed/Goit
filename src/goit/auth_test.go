// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit_test

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Jamozed/Goit/src/goit"
)

func TestNewSession(t *testing.T) {
	goit.Sessions = map[int64][]goit.Session{}
	goit.SessionsMutex = sync.RWMutex{}

	var uid int64 = 1
	var session = goit.Session{Ip: "127.0.0.1", Expiry: time.Unix(0, 0)}

	s, err := goit.NewSession(uid, session.Ip, session.Expiry)
	if err != nil {
		t.Fatal(err.Error())
	}

	if goit.Sessions[uid] == nil {
		t.Fatal("UID slice not added to the sessions map")
	}
	if len(goit.Sessions[uid]) != 1 {
		t.Fatal("Incorrect number of sessions added to the sessions map")
	}
	if s != goit.Sessions[uid][0] {
		t.Fatal("Added and returned sessions do not match")
	}
	if s.Ip != session.Ip {
		t.Fatal("Added session IP is incorrect")
	}
	if s.Expiry != session.Expiry {
		t.Fatal("Added session expiry is incorrect")
	}
	if !s.Seen.Before(time.Now()) {
		t.Fatal("Session seen time is in the future")
	}
	if len(s.Token) != 32 {
		t.Fatal("Session token length is incorrect")
	}
	if goit.SessionsMutex.TryLock() == false {
		t.Fatal("Sessions mutex was not unlocked")
	}
}

func TestHash(t *testing.T) {
	var pass = "password"
	var salt = make([]byte, 16)
	var hash = []byte{
		0x00, 0xB1, 0xEE, 0xD9, 0xBE, 0xE6, 0xDC, 0x06, 0x41, 0xA5, 0x07, 0x71,
		0x7D, 0xB7, 0x6B, 0x65, 0x20, 0xEC, 0x87, 0x6E, 0xCE, 0x6C, 0xD1, 0x09,
		0x25, 0xE4, 0x38, 0x75, 0xB5, 0x43, 0x57, 0x5E,
	}

	if !slices.Equal(goit.Hash(pass, salt), hash) {
		fmt.Printf("%x", goit.Hash(pass, salt))
		t.Fatal("Hash output is incorrect")
	}
}
