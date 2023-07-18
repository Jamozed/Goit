// git.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type GitCommand struct {
	prog string
	args []string
	dir  string
	env  []string
}

func HandleInfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.FormValue("service")
	repo := httpBase(w, r, service)
	if repo == nil {
		return
	}

	c := NewCommand(strings.TrimPrefix(service, "git-"), "--stateless-rpc", "--advertise-refs", ".")
	c.AddEnv(os.Environ()...)
	c.AddEnv("GIT_PROTOCOL=version=2")
	c.dir = "./" + repo.Name + ".git"

	refs, _, err := c.Run(nil, nil)
	if err != nil {
		log.Println("[Git]", err.Error())
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/x-"+service+"-advertisement")
	w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.WriteHeader(http.StatusOK)

	w.Write(pktLine("# service=" + service + "\n"))
	w.Write(pktFlush())
	w.Write(refs)
}

func HandleUploadPack(w http.ResponseWriter, r *http.Request) {
	repo := httpBase(w, r, "git-upload-pack")
	if repo == nil {
		return
	}

	serviceRPC(w, r, "git-upload-pack", repo)
}

func HandleReceivePack(w http.ResponseWriter, r *http.Request) {
	repo := httpBase(w, r, "git-receive-pack")
	if repo == nil {
		return
	}

	serviceRPC(w, r, "git-receive-pack", repo)
}

func httpBase(w http.ResponseWriter, r *http.Request, service string) *Repo {
	reponame := mux.Vars(r)["repo"]

	var isPull bool
	switch service {
	case "git-upload-pack":
		isPull = true
	case "git-receive-pack":
		isPull = false
	default:
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return nil
	}

	if r.Header.Get("Git-Protocol") != "version=2" {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return nil
	}

	repo, err := GetRepoByName(db, reponame)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return nil
	} else if repo == nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return nil
	}

	/* Require authentication other than for public pull */
	if repo.IsPrivate || !isPull {
		/* TODO authentcate */
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		return nil
	}

	return repo
}

func serviceRPC(w http.ResponseWriter, r *http.Request, service string, repo *Repo) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Println("[GitRPC]", err.Error())
		}
	}()

	if r.Header.Get("Content-Type") != "application/x-"+service+"-request" {
		log.Println("[GitRPC]", "Content-Type mismatch")
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		return
	}

	body := r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		if b, err := gzip.NewReader(r.Body); err != nil {
			log.Println("[GitRPC]", err.Error())
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			body = b
		}
	}

	c := NewCommand(strings.TrimPrefix(service, "git-"), "--stateless-rpc", ".")
	c.AddEnv(os.Environ()...)
	c.AddEnv("GIT_PROTOCOL=version=2")
	c.dir = "./" + repo.Name + ".git"

	w.Header().Add("Content-Type", "application/x-"+service+"-result")
	w.WriteHeader(http.StatusOK)

	if _, _, err := c.Run(body, w); err != nil {
		log.Println("[GitRPC]", err.Error())
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func pktLine(str string) []byte {
	PanicIf(len(str) > 65516, "pktLine: Payload exceeds maximum length")
	s := strconv.FormatUint(uint64(len(str)+4), 16)
	s = strings.Repeat("0", 4-len(s)%4) + s
	return []byte(s + str)
}

func pktFlush() []byte { return []byte("0000") }

func NewCommand(args ...string) *GitCommand {
	return &GitCommand{prog: "git", args: args}
}

func (C *GitCommand) AddArgs(args ...string) {
	C.args = append(C.args, args...)
}

func (C *GitCommand) AddEnv(env ...string) {
	C.env = append(C.env, env...)
}

func (C *GitCommand) Run(in io.Reader, out io.Writer) ([]byte, []byte, error) {
	c := exec.Command(C.prog, C.args...)
	c.Dir = C.dir
	c.Env = C.env
	c.Stdin = in

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	c.Stdout = stdout
	c.Stderr = os.Stderr

	if out != nil {
		c.Stdout = out
	}

	if err := c.Run(); err != nil {
		return nil, stderr.Bytes(), err
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}
