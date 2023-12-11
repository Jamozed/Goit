// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func HandleFile(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	tpath := chi.URLParam(r, "*")

	repo, err := goit.GetRepoByName(chi.URLParam(r, "repo"))
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || repo.OwnerId != user.Id)) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	data := struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Path, LineC, Size, Mode       string
		Lines                         []string
		Body                          string
		Editable                      bool
		HtmlPath                      template.HTML
	}{
		Title: repo.Name + " - File", Name: repo.Name, Description: repo.Description,
		Url:      util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable: (auth && repo.OwnerId == user.Id),
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

	if readme, _ := findReadme(gr, ref); readme != "" {
		data.Readme = path.Join("/", repo.Name, "file", readme)
	}
	if licence, _ := findLicence(gr, ref); licence != "" {
		data.Licence = path.Join("/", repo.Name, "file", licence)
	}

	commit, err := gr.CommitObject(ref.Hash())
	if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	file, err := commit.File(tpath)
	if errors.Is(err, object.ErrFileNotFound) {
		goit.HttpError(w, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	data.Mode = util.ModeString(uint32(file.Mode))
	data.Path = file.Name
	data.Size = humanize.IBytes(uint64(file.Size))

	parts := strings.Split(file.Name, "/")
	htmlPath := "<b style=\"padding-left: 0.4rem;\"><a href=\"/" + repo.Name + "/tree\">" + repo.Name + "</a></b>/"
	dirPath := ""

	for i := 0; i < len(parts)-1; i += 1 {
		dirPath = path.Join(dirPath, parts[i])
		htmlPath += "<a href=\"/" + repo.Name + "/tree/" + dirPath + "\">" + parts[i] + "</a>/"
	}
	htmlPath += parts[len(parts)-1]

	data.HtmlPath = template.HTML(htmlPath)

	if rc, err := file.Blob.Reader(); err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else {
		buf := make([]byte, min(file.Size, 512))

		if _, err := rc.Read(buf); err != nil {
			log.Println("[/repo/file]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}

		if strings.HasPrefix(http.DetectContentType(buf), "text") {
			buf2 := make([]byte, min(file.Size-int64(len(buf)), (10*1024*1024)-int64(len(buf))))
			if _, err := rc.Read(buf2); err != nil && !errors.Is(err, io.EOF) {
				log.Println("[/repo/file]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			}

			data.Body = string(append(buf, buf2...))
			data.Lines = strings.Split(data.Body, "\n")
		}

		rc.Close()
	}

	data.LineC = fmt.Sprint(len(data.Lines), " lines")

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/file", data); err != nil {
		log.Println("[/repo/file]", err.Error())
	}
}
