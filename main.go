// main.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Jamozed/Goit/res"
	goit "github.com/Jamozed/Goit/src"
	"github.com/gorilla/mux"
)

func main() {
	g, err := goit.InitGoit()
	if err != nil {
		log.Fatalln("[InitGoit]", err.Error())
	} else {
		defer g.Close()
	}

	mx := mux.NewRouter()
	mx.StrictSlash(true)

	mx.Path("/").HandlerFunc(g.HandleIndex)
	mx.Path("/user/login").Methods("GET", "POST").HandlerFunc(g.HandleUserLogin)
	mx.Path("/user/logout").Methods("GET", "POST").HandlerFunc(g.HandleUserLogout)
	// mx.Path("/user/settings").Methods("GET").HandlerFunc()
	mx.Path("/repo/create").Methods("GET", "POST").HandlerFunc(g.HandleRepoCreate)
	// mx.Path("/repo/delete").Methods("POST").HandlerFunc()
	// mx.Path("/admin/settings").Methods("GET").HandlerFunc()
	mx.Path("/admin/user").Methods("GET").HandlerFunc(g.HandleAdminUserIndex)
	// mx.Path("/admin/repos").Methods("GET").HandlerFunc()
	mx.Path("/admin/user/create").Methods("GET", "POST").HandlerFunc(g.HandleAdminUserCreate)
	// mx.Path("/admin/user/edit").Methods("GET", "POST").HandlerFunc()

	rm := mx.Path("/{repo}/").Subrouter()
	// rm.Path("/").Methods("GET").HandlerFunc()
	// rm.Path("/log").Methods("GET").HandlerFunc()
	// rm.Path("/tree").Methods("GET").HandlerFunc()
	// rm.Path("/refs").Methods("GET").HandlerFunc()

	mx.Path("/static/style.css").Methods(http.MethodGet).HandlerFunc(handleStyle)

	mx.PathPrefix("/").HandlerFunc(http.NotFound)
	rm.PathPrefix("/").HandlerFunc(http.NotFound)

	/* Create a ticker to periodically cleanup expired sessions */
	tick := time.NewTicker(1 * time.Hour)
	go func() {
		for range tick.C {
			goit.CleanupSessions()
		}
	}()

	if err := http.ListenAndServe(":8080", logHttp(mx)); err != nil {
		log.Fatalln("[HTTP]", err)
	}
}

func logHttp(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("[HTTP]", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func handleStyle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/css")
	if _, err := w.Write([]byte(res.Style)); err != nil {
		log.Println("[handleStyle]", err.Error())
	}
}
