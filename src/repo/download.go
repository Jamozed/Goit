// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[repo/download]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	tpath := chi.URLParam(r, "*")

	repo, err := goit.GetRepoByName(chi.URLParam(r, "repo"))
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || user.Id != repo.OwnerId)) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/download]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		goit.HttpError(w, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("[/repo/download]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	commit, err := gr.CommitObject(ref.Hash())
	if err != nil {
		log.Println("[/repo/download]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	file, err := commit.File(tpath)
	if errors.Is(err, object.ErrFileNotFound) {
		/* Possibly a directory, search file tree for prefix */
		var files []string
		var zSize uint64

		iter, err := commit.Files()
		if err != nil {
			log.Println("[/repo/download]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}

		iter.ForEach(func(f *object.File) error {
			if tpath == "" || strings.HasPrefix(f.Name, tpath+"/") {
				files = append(files, f.Name)
				zSize += uint64(f.Size)
			}

			return nil
		})

		if len(files) == 0 {
			goit.HttpError(w, http.StatusNotFound)
			return
		}

		/* Build and write ZIP of directory */
		w.Header().Set(
			"Content-Disposition", "attachment; filename="+util.If(tpath == "", repo.Name, filepath.Base(tpath))+".zip",
		)
		// w.Header().Set("Content-Length", fmt.Sprint(zSize))

		z := zip.NewWriter(w)
		for _, f := range files {
			zh := zip.FileHeader{Name: f, Method: zip.Store}

			zf, err := z.CreateHeader(&zh)
			if err != nil {
				log.Println("[/repo/download]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
			}

			if file, err := commit.File(f); err != nil {
				log.Println("[/repo/download]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else if rc, err := file.Blob.Reader(); err != nil {
				log.Println("[/repo/download]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			} else {
				if _, err := io.Copy(zf, rc); err != nil {
					log.Println("[/repo/download]", err.Error())
				}

				rc.Close()
			}
		}

		z.Close()
		return
	} else if err != nil {
		log.Println("[/repo/download]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if rc, err := file.Blob.Reader(); err != nil {
		log.Println("[/repo/download]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else {
		w.Header().Set("Content-Disposition", "attachement; filename="+filepath.Base(tpath))
		w.Header().Set("Content-Length", fmt.Sprint(file.Size))

		if _, err := io.Copy(w, rc); err != nil {
			log.Println("[/repo/download]", err.Error())
		}

		rc.Close()
	}
}
