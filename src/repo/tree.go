package repo

import (
	"log"
	"net/http"
	"strings"

	goit "github.com/Jamozed/Goit/src"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

func HandleTree(w http.ResponseWriter, r *http.Request) {
	_, uid := goit.AuthCookie(w, r, true)

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && repo.OwnerId != uid) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct {
		Mode, Name, Size string
		B                bool
	}
	data := struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Files                         []row
	}{
		Title: repo.Name + " - Tree", Name: repo.Name, Description: repo.Description,
		Url: util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
	}

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name))
	if err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	ref, err := gr.Head()
	if err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	commit, err := gr.CommitObject(ref.Hash())
	if err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	iter, err := commit.Tree()
	if err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := iter.Files().ForEach(func(f *object.File) error {
		size := humanize.IBytes(uint64(f.Size))
		data.Files = append(data.Files, row{
			Mode: util.ModeString(uint32(f.Mode)), Name: f.Name, Size: size,
			B: util.If(strings.HasSuffix(size, " B"), true, false),
		})

		return nil
	}); err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/tree", data); err != nil {
		log.Println("[/repo/tree]", err.Error())
	}
}
