package repo

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

func HandleRaw(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[repo/raw]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	treepath := mux.Vars(r)["path"]

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || user.Id != repo.OwnerId)) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		goit.HttpError(w, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	commit, err := gr.CommitObject(ref.Hash())
	if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	file, err := commit.File(treepath)
	if errors.Is(err, object.ErrFileNotFound) {
		goit.HttpError(w, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if rc, err := file.Blob.Reader(); err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else {
		buf := make([]byte, min(file.Size, (10*1024*1024)))
		if _, err := rc.Read(buf); err != nil && !errors.Is(err, io.EOF) {
			log.Println("[/repo/file]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(buf); err != nil {
			log.Println("[/repo/file]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}
	}
}
