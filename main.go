// main.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jamozed/Goit/res"
	goit "github.com/Jamozed/Goit/src"
	"github.com/gorilla/mux"
)

func main() {
	if err := goit.Goit(goit.ConfPath()); err != nil {
		log.Fatalln(err.Error())
	}

	h := mux.NewRouter()
	h.StrictSlash(true)

	h.Path("/").HandlerFunc(goit.HandleIndex)
	h.Path("/user/login").Methods("GET", "POST").HandlerFunc(goit.HandleUserLogin)
	h.Path("/user/logout").Methods("GET", "POST").HandlerFunc(goit.HandleUserLogout)
	// h.Path("/user/settings").Methods("GET").HandlerFunc()
	h.Path("/repo/create").Methods("GET", "POST").HandlerFunc(goit.HandleRepoCreate)
	// h.Path("/repo/delete").Methods("POST").HandlerFunc()
	// h.Path("/admin/settings").Methods("GET").HandlerFunc()
	h.Path("/admin/users").Methods("GET").HandlerFunc(goit.HandleAdminUsers)
	h.Path("/admin/user/create").Methods("GET", "POST").HandlerFunc(goit.HandleAdminUserCreate)
	h.Path("/admin/user/edit").Methods("GET", "POST").HandlerFunc(goit.HandleAdminUserEdit)
	h.Path("/admin/repos").Methods("GET").HandlerFunc(goit.HandleAdminRepos)

	h.Path("/{repo:.+(?:\\.git)$}").Methods(http.MethodGet).HandlerFunc(redirectDotGit)
	h.Path("/{repo}").Methods(http.MethodGet).HandlerFunc(goit.HandleRepoLog)
	h.Path("/{repo}/log").Methods(http.MethodGet).HandlerFunc(goit.HandleRepoLog)
	h.Path("/{repo}/tree").Methods(http.MethodGet).HandlerFunc(goit.HandleRepoTree)
	h.Path("/{repo}/refs").Methods(http.MethodGet).HandlerFunc(goit.HandleRepoRefs)
	h.Path("/{repo}/info/refs").Methods(http.MethodGet).HandlerFunc(goit.HandleInfoRefs)
	h.Path("/{repo}/git-upload-pack").Methods(http.MethodPost).HandlerFunc(goit.HandleUploadPack)
	h.Path("/{repo}/git-receive-pack").Methods(http.MethodPost).HandlerFunc(goit.HandleReceivePack)

	h.Path("/static/style.css").Methods(http.MethodGet).HandlerFunc(handleStyle)
	h.Path("/static/favicon.png").Methods(http.MethodGet).HandlerFunc(handleFavicon)

	h.PathPrefix("/").HandlerFunc(goit.HttpNotFound)

	/* Create a ticker to periodically cleanup expired sessions */
	tick := time.NewTicker(1 * time.Hour)
	go func() {
		for range tick.C {
			goit.CleanupSessions()
		}
	}()

	if err := http.ListenAndServe(goit.Conf.HttpAddr+":"+goit.Conf.HttpPort, logHttp(h)); err != nil {
		log.Fatalln("[HTTP]", err)
	}
}

func logHttp(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("[HTTP]", r.RemoteAddr, r.Method, r.URL.String())
		// log.Println("[HTTP]", r.Header)
		handler.ServeHTTP(w, r)
	})
}

func handleStyle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	if _, err := w.Write([]byte(res.Style)); err != nil {
		log.Println("[Style]", err.Error())
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	if goit.Favicon == nil {
		goit.HttpError(w, http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "image/png")
		if _, err := w.Write(goit.Favicon); err != nil {
			log.Println("[Favicon]", err.Error())
		}
	}
}

func redirectDotGit(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, ".git"), http.StatusMovedPermanently)
}
