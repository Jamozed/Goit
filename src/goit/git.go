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
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type gitCommand struct {
	prog string
	args []string
	Dir  string
	env  []string
}

func HandleInfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.FormValue("service")

	repo := gitHttpBase(w, r, service)
	if repo == nil {
		return
	}

	c := NewGitCommand(strings.TrimPrefix(service, "git-"), "--stateless-rpc", "--advertise-refs", ".")
	c.AddEnv(os.Environ()...)
	c.AddEnv("GIT_PROTOCOL=version=2")
	c.Dir = RepoPath(repo.Name, true)

	refs, _, err := c.Run(nil, nil)
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
	reponame := chi.URLParam(r, "repo")

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
	repo, err := GetRepoByName(reponame)
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

	c := NewGitCommand(strings.TrimPrefix(service, "git-"), "--stateless-rpc", ".")
	c.AddEnv(os.Environ()...)
	c.Dir = RepoPath(repo.Name, true)

	if p := r.Header.Get("Git-Protocol"); p == "version=2" {
		c.AddEnv("GIT_PROTOCOL=version=2")
	}

	w.Header().Add("Content-Type", "application/x-"+service+"-result")
	w.WriteHeader(http.StatusOK)

	if _, _, err := c.Run(body, w); err != nil {
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

func NewGitCommand(args ...string) *gitCommand {
	return &gitCommand{prog: "git", args: args}
}

func (C *gitCommand) AddEnv(env ...string) {
	C.env = append(C.env, env...)
}

func (C *gitCommand) Run(in io.Reader, out io.Writer) ([]byte, []byte, error) {
	c := exec.Command(C.prog, C.args...)
	c.Dir = C.Dir
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

type DiffStat struct {
	Name, Prev string
	Status     string
	Addition   int
	Deletion   int
	IsBinary   bool
}

var diffs = map[plumbing.Hash][]DiffStat{}
var diffsLock sync.RWMutex

var Sizes = map[plumbing.Hash]uint64{}
var SizesLock sync.RWMutex

func DiffStats(c *object.Commit) ([]DiffStat, error) {
	diffsLock.RLock()
	if stats, ok := diffs[c.Hash]; ok {
		diffsLock.RUnlock()
		return stats, nil
	}
	diffsLock.RUnlock()

	from, err := c.Tree()
	if err != nil {
		return nil, err
	}

	to := &object.Tree{}
	if c.NumParents() != 0 {
		parent, err := c.Parents().Next()
		if err != nil {
			return nil, err
		}

		to, err = parent.Tree()
		if err != nil {
			return nil, err
		}
	}

	patch, err := to.Patch(from)
	if err != nil {
		return nil, err
	}

	var stats []DiffStat
	for _, fp := range patch.FilePatches() {
		var stat DiffStat

		if len(fp.Chunks()) == 0 {
			if !fp.IsBinary() {
				continue
			}

			stat.IsBinary = true
		}

		from, to := fp.Files()
		if from == nil && to == nil {
			continue
		} else if from == nil /* Added */ {
			stat.Name = to.Path()
			stat.Status = "A"
		} else if to == nil /* Deleted */ {
			stat.Name = from.Path()
			stat.Status = "D"
		} else if from.Path() != to.Path() /* Renamed */ {
			stat.Name = to.Path()
			stat.Prev = from.Path()
			stat.Status = "R"
		} else {
			stat.Name = from.Path()
			stat.Status = "M"
		}

		for _, chunk := range fp.Chunks() {
			s := chunk.Content()
			if len(s) == 0 {
				continue
			}

			switch chunk.Type() {
			case diff.Add:
				stat.Addition += strings.Count(s, "\n")
				if s[len(s)-1] != '\n' {
					stat.Addition++
				}
			case diff.Delete:
				stat.Deletion += strings.Count(s, "\n")
				if s[len(s)-1] != '\n' {
					stat.Deletion++
				}
			}
		}

		stats = append(stats, stat)
	}

	diffsLock.Lock()
	diffs[c.Hash] = stats
	diffsLock.Unlock()

	return stats, nil
}

type countPair struct {
	hash  plumbing.Hash
	count uint64
}

var counts = map[string]countPair{}
var countsLock sync.RWMutex

func CommitCount(repo, branch string, hash plumbing.Hash) (uint64, error) {
	countsLock.RLock()
	if count, ok := counts[repo+"/"+branch]; ok && count.hash == hash {
		countsLock.RUnlock()
		return count.count, nil
	}
	countsLock.RUnlock()

	c := NewGitCommand("rev-list", "--count", branch)
	c.Dir = RepoPath(repo, true)
	out, _, err := c.Run(nil, nil)
	if err != nil {
		return 0, err
	}

	count, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, err
	}

	countsLock.Lock()
	counts[repo+"/"+branch] = countPair{hash, count}
	countsLock.Unlock()

	return count, nil
}
