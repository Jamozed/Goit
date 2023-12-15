// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package user

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
)

func HandleSessions(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	if !auth {
		goit.HttpError(w, http.StatusUnauthorized)
		return
	}

	_, ss := goit.GetSessionCookie(r)

	revoke, err := strconv.ParseInt(r.FormValue("revoke"), 10, 64)
	if err != nil {
		revoke = -1
	}

	type row struct{ Index, Ip, Seen, Expiry, Current string }
	var data = struct {
		Title    string
		Sessions []row
	}{Title: "User - Sessions"}

	goit.SessionsMutex.RLock()
	util.Debugln("[goit.HandleSessions] SessionsMutex rlock")

	if revoke >= 0 && revoke < int64(len(goit.Sessions[user.Id])) {
		var token = goit.Sessions[user.Id][revoke].Token
		var current = token == ss.Token

		goit.SessionsMutex.RUnlock()
		util.Debugln("[goit.HandleSessions] SessionsMutex runlock")

		goit.EndSession(user.Id, token)

		if current {
			goit.EndSessionCookie(w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		http.Redirect(w, r, "/user/sessions", http.StatusFound)
		return
	}

	for i, v := range goit.Sessions[user.Id] {
		data.Sessions = append(data.Sessions, row{
			Index: fmt.Sprint(i), Ip: v.Ip, Seen: v.Seen.Format(time.DateTime), Expiry: v.Expiry.Format(time.DateTime),
			Current: util.If(v.Token == ss.Token, "(current)", ""),
		})
	}

	goit.SessionsMutex.RUnlock()
	util.Debugln("[goit.HandleSessions] SessionsMutex runlock")

	if err := goit.Tmpl.ExecuteTemplate(w, "user/sessions", data); err != nil {
		log.Println("[/user/login]", err.Error())
	}
}
