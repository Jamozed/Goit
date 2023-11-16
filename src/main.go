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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Jamozed/Goit/res"
	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/repo"
	"github.com/Jamozed/Goit/src/user"
	"github.com/Jamozed/Goit/src/util"
	"github.com/adrg/xdg"
	"github.com/gorilla/mux"
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

	log.Println("Starting Goit", res.Version)

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

	if err := goit.Goit(goit.ConfPath()); err != nil {
		log.Fatalln(err.Error())
	}

	h := mux.NewRouter()
	h.StrictSlash(true)

	h.Path("/").HandlerFunc(goit.HandleIndex)
	h.Path("/user/login").Methods("GET", "POST").HandlerFunc(user.HandleLogin)
	h.Path("/user/logout").Methods("GET", "POST").HandlerFunc(goit.HandleUserLogout)
	h.Path("/user/sessions").Methods("GET", "POST").HandlerFunc(user.HandleSessions)
	h.Path("/user/edit").Methods("GET", "POST").HandlerFunc(user.HandleEdit)
	h.Path("/repo/create").Methods("GET", "POST").HandlerFunc(repo.HandleCreate)
	h.Path("/admin").Methods("GET").HandlerFunc(goit.HandleAdminIndex)
	h.Path("/admin/users").Methods("GET").HandlerFunc(goit.HandleAdminUsers)
	h.Path("/admin/user/create").Methods("GET", "POST").HandlerFunc(goit.HandleAdminUserCreate)
	h.Path("/admin/user/edit").Methods("GET", "POST").HandlerFunc(goit.HandleAdminUserEdit)
	h.Path("/admin/repos").Methods("GET").HandlerFunc(goit.HandleAdminRepos)
	h.Path("/admin/repo/edit").Methods("GET", "POST").HandlerFunc(goit.HandleAdminRepoEdit)

	h.Path("/{repo:.+(?:\\.git)$}").Methods("GET").HandlerFunc(redirectDotGit)
	h.Path("/{repo}").Methods("GET").HandlerFunc(repo.HandleLog)
	h.Path("/{repo}/log").Methods("GET").HandlerFunc(repo.HandleLog)
	h.Path("/{repo}/log/{path:.*}").Methods("GET").HandlerFunc(repo.HandleLog)
	h.Path("/{repo}/commit/{hash}").Methods("GET").HandlerFunc(repo.HandleCommit)
	h.Path("/{repo}/tree").Methods("GET").HandlerFunc(repo.HandleTree)
	h.Path("/{repo}/tree/{path:.*}").Methods("GET").HandlerFunc(repo.HandleTree)
	h.Path("/{repo}/file/{path:.*}").Methods("GET").HandlerFunc(repo.HandleFile)
	h.Path("/{repo}/raw/{path:.*}").Methods("GET").HandlerFunc(repo.HandleRaw)
	h.Path("/{repo}/download/{path:.*}").Methods("GET").HandlerFunc(repo.HandleDownload)
	h.Path("/{repo}/refs").Methods("GET").HandlerFunc(repo.HandleRefs)
	h.Path("/{repo}/edit").Methods("GET", "POST").HandlerFunc(repo.HandleEdit)
	h.Path("/{repo}/info/refs").Methods("GET").HandlerFunc(goit.HandleInfoRefs)
	h.Path("/{repo}/git-upload-pack").Methods("POST").HandlerFunc(goit.HandleUploadPack)
	h.Path("/{repo}/git-receive-pack").Methods("POST").HandlerFunc(goit.HandleReceivePack)

	h.Path("/static/style.css").Methods("GET").HandlerFunc(handleStyle)
	h.Path("/static/favicon.png").Methods("GET").HandlerFunc(handleFavicon)

	h.PathPrefix("/").HandlerFunc(goit.HttpNotFound)

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
	if err := http.ListenAndServe(goit.Conf.HttpAddr+":"+goit.Conf.HttpPort, logHttp(h)); err != nil {
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
