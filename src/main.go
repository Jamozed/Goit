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

var protect func(http.Handler) http.Handler

func main() {
	var backup bool

	flag.BoolVar(&backup, "backup", false, "Perform a backup")
	flag.BoolVar(&util.Debug, "debug", false, "Enable debug logging")
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
		goit.Cron.Stop()
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
	h.Use(logHttp)

	protect = csrf.Protect(
		[]byte(goit.Conf.CsrfSecret), csrf.FieldName("csrf.Token"), csrf.CookieName("csrf"),
		csrf.Secure(util.If(goit.Conf.UsesHttps, true, false)),
	)

	h.Group(func(r chi.Router) {
		r.Use(protect)

		r.Get("/", goit.HandleIndex)
		r.Get("/user/login", user.HandleLogin)
		r.Post("/user/login", user.HandleLogin)
		r.Get("/user/logout", goit.HandleUserLogout)
		r.Post("/user/logout", goit.HandleUserLogout)
		r.Get("/user/sessions", user.HandleSessions)
		r.Post("/user/sessions", user.HandleSessions)
		r.Get("/user/edit", user.HandleEdit)
		r.Post("/user/edit", user.HandleEdit)
		r.Get("/repo/create", repo.HandleCreate)
		r.Post("/repo/create", repo.HandleCreate)
		r.Get("/admin", admin.HandleIndex)
		r.Get("/admin/users", admin.HandleUsers)
		r.Get("/admin/user/create", admin.HandleUserCreate)
		r.Post("/admin/user/create", admin.HandleUserCreate)
		r.Get("/admin/user/edit", admin.HandleUserEdit)
		r.Post("/admin/user/edit", admin.HandleUserEdit)
		r.Get("/admin/repos", admin.HandleRepos)
		r.Get("/admin/repo/edit", admin.HandleRepoEdit)
		r.Post("/admin/repo/edit", admin.HandleRepoEdit)

		r.Get("/static/style.css", handleStyle)
		r.Get("/static/favicon.png", handleFavicon)
		r.Get("/favicon.ico", goit.HttpNotFound)
	})

	/* TODO figure out how to use a subrouter after manually parsing the repo path */
	h.HandleFunc("/*", HandleRepo)

	/* Old repository routing, doesn't support directories */
	// h.Get("/{repo}", repo.HandleLog)
	// h.Get("/{repo}/log", repo.HandleLog)
	// h.Get("/{repo}/log/*", repo.HandleLog)
	// h.Get("/{repo}/commit/{hash}", repo.HandleCommit)
	// h.Get("/{repo}/tree", repo.HandleTree)
	// h.Get("/{repo}/tree/*", repo.HandleTree)
	// h.Get("/{repo}/file/*", repo.HandleFile)
	// h.Get("/{repo}/raw/*", repo.HandleRaw)
	// h.Get("/{repo}/download", repo.HandleDownload)
	// h.Get("/{repo}/download/*", repo.HandleDownload)
	// h.Get("/{repo}/refs", repo.HandleRefs)
	// h.Get("/{repo}/edit", repo.HandleEdit)
	// h.Post("/{repo}/edit", repo.HandleEdit)
	// h.Get("/{repo}/info/refs", goit.HandleInfoRefs)
	// h.Get("/{repo}/git-upload-pack", goit.HandleUploadPack)
	// h.Post("/{repo}/git-upload-pack", goit.HandleUploadPack)
	// h.Get("/{repo}/git-receive-pack", goit.HandleReceivePack)
	// h.Post("/{repo}/git-receive-pack", goit.HandleReceivePack)

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
		log.Fatalln("[http]", err.Error())
	}
}

func logHttp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)

		ip := r.RemoteAddr
		if fip := r.Header.Get("X-Forwarded-For"); goit.Conf.IpForwarded && fip != "" {
			ip = fip
		}

		log.Println("[http]", r.Method, r.URL.String(), "from", ip, "in", time.Since(t1))
	})
}

func handleStyle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	if _, err := w.Write([]byte(res.Style)); err != nil {
		log.Println("[style]", err.Error())
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	if goit.Favicon == nil {
		goit.HttpError(w, http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "image/png")
		if _, err := w.Write(goit.Favicon); err != nil {
			log.Println("[favicon]", err.Error())
		}
	}
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

func HandleRepo(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	repos, err := goit.GetRepos()
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	var rpath string
	for _, p := range parts {
		rpath = path.Join(rpath, p)

		for _, r := range repos {
			if rpath == r.Name {
				goto found
			}
		}
	}

	goit.HttpError(w, http.StatusNotFound)
	return

found:
	spath := strings.TrimPrefix(r.URL.Path, "/"+rpath)

	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		log.Println("[route] NULL route context")
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	rctx.URLParams.Add("repo", rpath)
	rctx.URLParams.Add("*", "")

	switch r.Method {
	case http.MethodGet:
		switch {
		case strings.HasPrefix(spath, "/log"), len(spath) == 0:
			rctx.URLParams.Add("*", strings.TrimLeft(strings.TrimPrefix(spath, "/log"), "/"))
			protect(http.HandlerFunc(repo.HandleLog)).ServeHTTP(w, r)

		case strings.HasPrefix(spath, "/commit/"):
			hash := strings.TrimPrefix(spath, "/commit/")
			if strings.Contains(hash, "/") {
				goit.HttpError(w, http.StatusNotFound)
			}

			rctx.URLParams.Add("hash", hash)
			protect(http.HandlerFunc(repo.HandleCommit)).ServeHTTP(w, r)

		case strings.HasPrefix(spath, "/tree"):
			rctx.URLParams.Add("*", strings.TrimLeft(strings.TrimPrefix(spath, "/tree"), "/"))
			protect(http.HandlerFunc(repo.HandleTree)).ServeHTTP(w, r)

		case strings.HasPrefix(spath, "/file/"):
			rctx.URLParams.Add("*", strings.TrimPrefix(spath, "/file/"))
			protect(http.HandlerFunc(repo.HandleFile)).ServeHTTP(w, r)

		case strings.HasPrefix(spath, "/raw/"):
			rctx.URLParams.Add("*", strings.TrimPrefix(spath, "/raw/"))
			protect(http.HandlerFunc(repo.HandleRaw)).ServeHTTP(w, r)

		case strings.HasPrefix(spath, "/download"):
			rctx.URLParams.Add("*", strings.TrimLeft(strings.TrimPrefix(spath, "/download"), "/"))
			protect(http.HandlerFunc(repo.HandleDownload)).ServeHTTP(w, r)

		case spath == "/refs":
			protect(http.HandlerFunc(repo.HandleRefs)).ServeHTTP(w, r)
		case spath == "/edit":
			protect(http.HandlerFunc(repo.HandleEdit)).ServeHTTP(w, r)

		case spath == "/info/refs":
			goit.HandleInfoRefs(w, r)
		case spath == "/git-upload-pack":
			goit.HandleUploadPack(w, r)
		case spath == "/git-receive-pack":
			goit.HandleReceivePack(w, r)

		default:
			goit.HttpError(w, http.StatusNotFound)
		}

	case http.MethodPost:
		switch {
		case spath == "/edit":
			protect(http.HandlerFunc(repo.HandleEdit)).ServeHTTP(w, r)

		case spath == "/git-upload-pack":
			goit.HandleUploadPack(w, r)
		case spath == "/git-receive-pack":
			goit.HandleReceivePack(w, r)
		}

	default:
		goit.HttpError(w, http.StatusNotFound)
	}
}
