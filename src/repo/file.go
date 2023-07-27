package repo

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	goit "github.com/Jamozed/Goit/src"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

func HandleFile(w http.ResponseWriter, r *http.Request) {
	_, uid := goit.AuthCookie(w, r, true)

	treepath := mux.Vars(r)["path"]
	// if treepath == "" {
	// 	goit.HttpError(w, http.StatusNotFound)
	// 	return
	// }

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && repo.OwnerId != uid) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	data := struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Mode, File, Size              string
		Lines                         []string
	}{
		Title: repo.Name + " - File", Name: repo.Name, Description: repo.Description,
		Url: util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name))
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

	data.Mode = util.ModeString(uint32(file.Mode))
	data.File = file.Name
	data.Size = humanize.IBytes(uint64(file.Size))

	if rc, err := file.Blob.Reader(); err != nil {
		log.Println("[/repo/file]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else {
		buf := make([]byte, util.Min(file.Size, 512))

		if _, err := rc.Read(buf); err != nil {
			log.Println("[/repo/file]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}

		if strings.HasPrefix(http.DetectContentType(buf), "text") {
			buf2 := make([]byte, util.Min(file.Size-int64(len(buf)), (10*1024*1024)-int64(len(buf))))
			if _, err := rc.Read(buf2); err != nil && !errors.Is(err, io.EOF) {
				log.Println("[/repo/file]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			}

			data.Lines = strings.Split(string(append(buf, buf2...)), "\n")
		}
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/file", data); err != nil {
		log.Println("[/repo/file]", err.Error())
	}
}
