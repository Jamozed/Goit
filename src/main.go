// main.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Jamozed/Goit/res"
	"github.com/Jamozed/Goit/src/admin"
	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/repo"
	"github.com/Jamozed/Goit/src/user"
	"github.com/Jamozed/Goit/src/util"
	"github.com/adrg/xdg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
)

func main() {
	var backup bool

	flag.BoolVar(&backup, "backup", false, "Perform a backup")
	flag.BoolVar(&goit.Debug, "debug", false, "Enable debug logging")
	flag.Parse()

	if backup /* IPC client */ {
		c, err := net.Dial("unix", filepath.Join(xdg.RuntimeDir, "goit-"+goit.Conf.HttpPort+".sock"))
		if err != nil {
			log.Fatalln(err.Error())
		}

		_, err = c.Write([]byte{0xBA})
		if err != nil {
			log.Fatalln(err.Error())
		}

		buf := make([]byte, 512)
		n, err := c.Read(buf)
		if err != nil {
			log.Fatalln(err.Error())
		}

		fmt.Println(string(buf[1:n]))
		c.Close()

		os.Exit(util.If(buf[0] == 0x01, -1, 0))
	}

	/* Listen for and handle SIGINT */
	stop := make(chan struct{})
	wait := &sync.WaitGroup{}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		close(stop)
		wait.Wait()
		os.Exit(0)
	}()

	/* Initialise Goit */
	if err := goit.Goit(goit.ConfPath()); err != nil {
		log.Fatalln(err.Error())
	}

	h := chi.NewRouter()
	h.NotFound(goit.HttpNotFound)
	h.Use(middleware.RedirectSlashes)

	if goit.Debug {
		h.Use(middleware.Logger)
	} else {
		h.Use(logHttp)
	}

	h.Use(csrf.Protect(
		[]byte(goit.Conf.CsrfSecret), csrf.FieldName("csrf.Token"), csrf.CookieName("csrf"),
		csrf.Secure(util.If(goit.Conf.UsesHttps, true, false)),
	))

	h.Get("/", goit.HandleIndex)
	h.Get("/user/login", user.HandleLogin)
	h.Post("/user/login", user.HandleLogin)
	h.Get("/user/logout", goit.HandleUserLogout)
	h.Post("/user/logout", goit.HandleUserLogout)
	h.Get("/user/sessions", user.HandleSessions)
	h.Post("/user/sessions", user.HandleSessions)
	h.Get("/user/edit", user.HandleEdit)
	h.Post("/user/edit", user.HandleEdit)
	h.Get("/repo/create", repo.HandleCreate)
	h.Post("/repo/create", repo.HandleCreate)
	h.Get("/admin", admin.HandleIndex)
	h.Get("/admin/users", admin.HandleUsers)
	h.Get("/admin/user/create", admin.HandleUserCreate)
	h.Post("/admin/user/create", admin.HandleUserCreate)
	h.Get("/admin/user/edit", admin.HandleUserEdit)
	h.Post("/admin/user/edit", admin.HandleUserEdit)
	h.Get("/admin/repos", admin.HandleRepos)
	h.Get("/admin/repo/edit", admin.HandleRepoEdit)
	h.Post("/admin/repo/edit", admin.HandleRepoEdit)

	h.Get("/static/style.css", handleStyle)
	h.Get("/static/favicon.png", handleFavicon)
	h.Get("/favicon.ico", goit.HttpNotFound)

	h.Get("/{repo:.+(?:\\.git)$}", redirectDotGit)
	h.Get("/{repo}", repo.HandleLog)
	h.Get("/{repo}/log", repo.HandleLog)
	h.Get("/{repo}/log/*", repo.HandleLog)
	h.Get("/{repo}/commit/{hash}", repo.HandleCommit)
	h.Get("/{repo}/tree", repo.HandleTree)
	h.Get("/{repo}/tree/*", repo.HandleTree)
	h.Get("/{repo}/file/*", repo.HandleFile)
	h.Get("/{repo}/raw/*", repo.HandleRaw)
	h.Get("/{repo}/download", repo.HandleDownload)
	h.Get("/{repo}/download/*", repo.HandleDownload)
	h.Get("/{repo}/refs", repo.HandleRefs)
	h.Get("/{repo}/edit", repo.HandleEdit)
	h.Post("/{repo}/edit", repo.HandleEdit)
	h.Get("/{repo}/info/refs", goit.HandleInfoRefs)
	h.Get("/{repo}/git-upload-pack", goit.HandleUploadPack)
	h.Post("/{repo}/git-upload-pack", goit.HandleUploadPack)
	h.Get("/{repo}/git-receive-pack", goit.HandleReceivePack)
	h.Post("/{repo}/git-receive-pack", goit.HandleReceivePack)

	/* Create a ticker to periodically cleanup expired sessions */
	tick := time.NewTicker(1 * time.Hour)
	go func() {
		for range tick.C {
			goit.CleanupSessions()
		}
	}()

	/* Listen for IPC */
	ipc, err := net.Listen("unix", filepath.Join(xdg.RuntimeDir, "goit-"+goit.Conf.HttpPort+".sock"))
	if err != nil {
		log.Fatalln("[sock]", err.Error())
	}

	go func() {
		defer ipc.Close()
		<-stop
	}()

	wait.Add(1)
	go handleIpc(stop, wait, ipc)

	/* Listen for HTTP on the specified port */
	if err := http.ListenAndServe(goit.Conf.HttpAddr+":"+goit.Conf.HttpPort, h); err != nil {
		log.Fatalln("[HTTP]", err.Error())
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

/* Handle IPC messages. */
func handleIpc(stop chan struct{}, wait *sync.WaitGroup, ipc net.Listener) {
	defer wait.Done()

	for {
		select {
		case <-stop:
			return
		default:
			c, err := ipc.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Println("[ipc]", err.Error())
				}
				continue
			}

			c.SetReadDeadline(time.Now().Add(1 * time.Second))

			buf := make([]byte, 1)
			if _, err := c.Read(buf); err != nil {
				log.Println("[ipc]", err.Error())
				continue
			}

			if buf[0] == 0xBA {
				log.Println("[backup] Starting")
				if err := goit.Backup(); err != nil {
					c.Write(append([]byte{0x01}, []byte(err.Error())...))
					log.Println("[backup]", err.Error())
				} else {
					c.Write(append([]byte{0x00}, []byte("SUCCESS")...))
					log.Println("[backup] Success")
				}
			} else {
				c.Write(append([]byte{0x01}, []byte("ILLEGAL")...))
			}

			c.Close()
		}
	}
}
