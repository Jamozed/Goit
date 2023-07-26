package repo

import (
	"errors"
	"log"
	"net/http"
	"path"
	"sort"
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
	treepath := mux.Vars(r)["path"]

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && repo.OwnerId != uid) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct {
		Mode, Name, Path, Size string
		B                      bool
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

	tree, err := commit.Tree()
	if err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if treepath != "" {
		data.Files = append(data.Files, row{Mode: "d---------", Name: "..", Path: path.Dir(treepath)})

		tree, err = tree.Tree(treepath)
		if errors.Is(err, object.ErrDirectoryNotFound) {
			goit.HttpError(w, http.StatusNotFound)
			return
		} else if err != nil {
			log.Println("[/repo/tree]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}
	}

	sort.SliceStable(tree.Entries, func(i, j int) bool {
		if tree.Entries[i].Mode&0o40000 != 0 && tree.Entries[j].Mode&0o40000 == 0 {
			return true
		}

		return tree.Entries[i].Name < tree.Entries[j].Name
	})

	for _, v := range tree.Entries {
		size := ""

		if v.Mode&0o40000 == 0 {
			file, err := tree.File(v.Name)
			if err != nil {
				log.Println("[/repo/tree]", err.Error())
				goit.HttpError(w, http.StatusInternalServerError)
				return
			}

			size = humanize.IBytes(uint64(file.Size))
		}

		data.Files = append(data.Files, row{
			Mode: util.ModeString(uint32(v.Mode)), Name: v.Name, Path: path.Join(treepath, v.Name), Size: size,
			B: util.If(strings.HasSuffix(size, " B"), true, false),
		})
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/tree", data); err != nil {
		log.Println("[/repo/tree]", err.Error())
	}
}
