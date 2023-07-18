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

	h := mux.NewRouter()
	h.StrictSlash(true)

	h.Path("/").HandlerFunc(g.HandleIndex)
	h.Path("/user/login").Methods("GET", "POST").HandlerFunc(g.HandleUserLogin)
	h.Path("/user/logout").Methods("GET", "POST").HandlerFunc(g.HandleUserLogout)
	// h.Path("/user/settings").Methods("GET").HandlerFunc()
	h.Path("/repo/create").Methods("GET", "POST").HandlerFunc(g.HandleRepoCreate)
	// h.Path("/repo/delete").Methods("POST").HandlerFunc()
	// h.Path("/admin/settings").Methods("GET").HandlerFunc()
	h.Path("/admin/user").Methods("GET").HandlerFunc(g.HandleAdminUserIndex)
	// h.Path("/admin/repos").Methods("GET").HandlerFunc()
	h.Path("/admin/user/create").Methods("GET", "POST").HandlerFunc(g.HandleAdminUserCreate)
	// h.Path("/admin/user/edit").Methods("GET", "POST").HandlerFunc()

	h.Path("/{repo}/").Methods(http.MethodGet).HandlerFunc(g.HandleRepoLog)
	h.Path("/{repo}/log").Methods(http.MethodGet).HandlerFunc(g.HandleRepoLog)
	// h.Path("/{repo}/tree").Methods(http.MethodGet).HandlerFunc(g.HandleRepoTree)
	// h.Path("/{repo}/refs").Methods(http.MethodGet).HandlerFunc(g.HandleRepoRefs)
	h.Path("/{repo}/info/refs").Methods(http.MethodGet).HandlerFunc(goit.HandleInfoRefs)
	h.Path("/{repo}/git-upload-pack").Methods(http.MethodPost).HandlerFunc(goit.HandleUploadPack)
	h.Path("/{repo}/git-receive-pack").Methods(http.MethodPost).HandlerFunc(goit.HandleReceivePack)

	h.Path("/static/style.css").Methods(http.MethodGet).HandlerFunc(handleStyle)

	h.PathPrefix("/").HandlerFunc(http.NotFound)

	/* Create a ticker to periodically cleanup expired sessions */
	tick := time.NewTicker(1 * time.Hour)
	go func() {
		for range tick.C {
			goit.CleanupSessions()
		}
	}()

	if err := http.ListenAndServe(":8080", logHttp(h)); err != nil {
		log.Fatalln("[HTTP]", err)
	}
}

func logHttp(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("[HTTP]", r.RemoteAddr, r.Method, r.URL.String())
		handler.ServeHTTP(w, r)
	})
}

func handleStyle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/css")
	if _, err := w.Write([]byte(res.Style)); err != nil {
		log.Println("[handleStyle]", err.Error())
	}
}
