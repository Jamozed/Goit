package repo

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

func HandleTree(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	treepath := mux.Vars(r)["path"]

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (repo.IsPrivate && (!auth || repo.OwnerId != user.Id)) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	type row struct {
		Mode, Name, Path, RawPath, Size string
		IsFile, B                       bool
	}
	data := struct {
		Title, Name, Description, Url string
		Readme, Licence               string
		Path, Size                    string
		Files                         []row
		Editable                      bool
		HtmlPath                      template.HTML
	}{
		Title: repo.Name + " - Tree", Name: repo.Name, Description: repo.Description,
		Url:      util.If(goit.Conf.UsesHttps, "https://", "http://") + r.Host + "/" + repo.Name,
		Editable: (auth && repo.OwnerId == user.Id),
	}

	parts := strings.Split(treepath, "/")
	htmlPath := "<b style=\"padding-left: 0.4rem;\"><a href=\"/" + repo.Name + "/tree\">" + repo.Name + "</a></b>/"
	dirPath := ""

	for i := 0; i < len(parts)-1; i += 1 {
		dirPath = path.Join(dirPath, parts[i])
		htmlPath += "<a href=\"/" + repo.Name + "/tree/" + dirPath + "\">" + parts[i] + "</a>/"
	}
	htmlPath += parts[len(parts)-1]

	data.HtmlPath = template.HTML(htmlPath)

	gr, err := git.PlainOpen(goit.RepoPath(repo.Name, true))
	if err != nil {
		log.Println("[/repo/tree]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if ref, err := gr.Head(); err != nil {
		if !errors.Is(err, plumbing.ErrReferenceNotFound) {
			log.Println("[/repo/tree]", err.Error())
			goit.HttpError(w, http.StatusInternalServerError)
			return
		}
	} else {
		if readme, _ := findReadme(gr, ref); readme != "" {
			data.Readme = path.Join("/", repo.Name, "file", readme)
		}
		if licence, _ := findLicence(gr, ref); licence != "" {
			data.Licence = path.Join("/", repo.Name, "file", licence)
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
			data.Files = append(data.Files, row{
				Mode: "d---------", Name: "..", Path: path.Join("tree", path.Dir(treepath)),
			})

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

		var totalSize uint64

		for _, v := range tree.Entries {
			var fpath, rpath, size string
			var isFile bool

			if v.Mode&0o40000 == 0 {
				file, err := tree.File(v.Name)
				if err != nil {
					log.Println("[/repo/tree]", err.Error())
					goit.HttpError(w, http.StatusInternalServerError)
					return
				}

				fpath = path.Join("file", treepath, v.Name)
				rpath = path.Join(treepath, v.Name)
				size = humanize.IBytes(uint64(file.Size))

				isFile = true

				totalSize += uint64(file.Size)
			} else {
				var dirSize uint64

				dirt, err := tree.Tree(v.Name)
				if err != nil {
					log.Println("[/repo/tree]", err.Error())
					goit.HttpError(w, http.StatusInternalServerError)
					return
				}

				if err := dirt.Files().ForEach(func(f *object.File) error {
					dirSize += uint64(f.Size)
					return nil
				}); err != nil {
					log.Println("[/repo/tree]", err.Error())
					goit.HttpError(w, http.StatusInternalServerError)
					return
				}

				fpath = path.Join("tree", treepath, v.Name)
				rpath = path.Join(treepath, v.Name)
				size = humanize.IBytes(dirSize)

				totalSize += dirSize
			}

			data.Files = append(data.Files, row{
				Mode: util.ModeString(uint32(v.Mode)), Name: v.Name, Path: fpath, RawPath: rpath, Size: size,
				IsFile: isFile, B: util.If(strings.HasSuffix(size, " B"), true, false),
			})
		}

		data.Size = humanize.IBytes(totalSize)
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "repo/tree", data); err != nil {
		log.Println("[/repo/tree]", err.Error())
	}
}
