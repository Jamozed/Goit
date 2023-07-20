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

type gitCommand struct {
	prog string
	args []string
	dir  string
	env  []string
}

func HandleInfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.FormValue("service")

	repo := gitHttpBase(w, r, service)
	if repo == nil {
		return
	}

	c := newCommand(strings.TrimPrefix(service, "git-"), "--stateless-rpc", "--advertise-refs", ".")
	c.addEnv(os.Environ()...)
	c.addEnv("GIT_PROTOCOL=version=2")
	c.dir = GetRepoPath(repo.Name)

	refs, _, err := c.run(nil, nil)
	if err != nil {
		log.Println("[Git HTTP]", err.Error())
		HttpError(w, http.StatusInternalServerError)
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
	const service = "git-upload-pack"

	repo := gitHttpBase(w, r, service)
	if repo == nil {
		return
	}

	gitHttpRpc(w, r, service, repo)
}

func HandleReceivePack(w http.ResponseWriter, r *http.Request) {
	const service = "git-receive-pack"

	repo := gitHttpBase(w, r, service)
	if repo == nil {
		return
	}

	gitHttpRpc(w, r, service, repo)
}

func gitHttpBase(w http.ResponseWriter, r *http.Request, service string) *Repo {
	reponame := mux.Vars(r)["repo"]

	/* Check that the Git service and protocol version are supported */
	if service != "git-upload-pack" && service != "git-receive-pack" {
		w.WriteHeader(http.StatusForbidden)
		return nil
	}
	if service == "git-upload-pack" && r.Header.Get("Git-Protocol") != "version=2" {
		w.WriteHeader(http.StatusForbidden)
		return nil
	}

	/* Load the repository from the database */
	repo, err := GetRepoByName(db, reponame)
	if err != nil {
		log.Println("[Git HTTP]", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	/* Require authentication other than for public pull */
	if repo == nil || repo.IsPrivate || service == "git-receive-pack" {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"git\"")
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}

		user, err := GetUserByName(username)
		if err != nil {
			log.Println("[Git HTTP]", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		}

		/* If the user doesn't exist or has invalid credentials */
		if user == nil || !bytes.Equal(Hash(password, user.Salt), user.Pass) {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"git\"")
			w.WriteHeader(http.StatusUnauthorized)
			return nil
		}

		/* If the repo doesn't exist or isn't owned by the user */
		if repo == nil || user.Id != repo.OwnerId {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}
	}

	if repo == nil {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	return repo
}

func gitHttpRpc(w http.ResponseWriter, r *http.Request, service string, repo *Repo) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Println("[Git RPC]", err.Error())
		}
	}()

	if r.Header.Get("Content-Type") != "application/x-"+service+"-request" {
		log.Println("[Git RPC]", "Content-Type mismatch")
		HttpError(w, http.StatusUnauthorized)
		return
	}

	body := r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		if b, err := gzip.NewReader(r.Body); err != nil {
			log.Println("[Git RPC]", err.Error())
			HttpError(w, http.StatusInternalServerError)
			return
		} else {
			body = b
		}
	}

	c := newCommand(strings.TrimPrefix(service, "git-"), "--stateless-rpc", ".")
	c.addEnv(os.Environ()...)
	c.dir = GetRepoPath(repo.Name)

	if p := r.Header.Get("Git-Protocol"); p == "version=2" {
		c.addEnv("GIT_PROTOCOL=version=2")
	}

	w.Header().Add("Content-Type", "application/x-"+service+"-result")
	w.WriteHeader(http.StatusOK)

	if _, _, err := c.run(body, w); err != nil {
		log.Println("[Git RPC]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}
}

func pktLine(str string) []byte {
	s := strconv.FormatUint(uint64(len(str)+4), 16)
	s = strings.Repeat("0", 4-len(s)%4) + s
	return []byte(s + str)
}

func pktFlush() []byte { return []byte("0000") }

func newCommand(args ...string) *gitCommand {
	return &gitCommand{prog: "git", args: args}
}

func (C *gitCommand) addEnv(env ...string) {
	C.env = append(C.env, env...)
}

func (C *gitCommand) run(in io.Reader, out io.Writer) ([]byte, []byte, error) {
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
