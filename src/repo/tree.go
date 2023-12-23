// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

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
	"github.com/go-chi/chi/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func HandleTree(w http.ResponseWriter, r *http.Request) {
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

	type row struct {
		Mode, Name, Path, RawPath, Size string
		IsFile, B                       bool
	}
	data := struct {
		HeaderFields
		Title, Path, Size string
		Files             []row
		HtmlPath          template.HTML
	}{
		Title:        repo.Name + " - Tree",
		HeaderFields: GetHeaderFields(auth, user, repo, r.Host),
	}

	parts := strings.Split(tpath, "/")
	htmlPath := "<b style=\"padding-left: 0.4rem;\"><a href=\"/" + repo.Name + "/tree\">" + repo.Name + "</a></b>/"
	dirPath := ""

	for i := 0; i < len(parts)-1; i += 1 {
		dirPath = path.Join(dirPath, parts[i])
		htmlPath += "<a href=\"/" + repo.Name + "/tree/" + dirPath + "\">" + parts[i] + "</a>/"
	}
	htmlPath += parts[len(parts)-1]

	data.Path = tpath
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
		if readme, _ := findPattern(gr, ref, readmePattern); readme != "" {
			data.Readme = path.Join("/", repo.Name, "file", readme)
		}
		if licence, _ := findPattern(gr, ref, licencePattern); licence != "" {
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

		if tpath != "" {
			data.Files = append(data.Files, row{
				Mode: "d---------", Name: "..", Path: path.Join("tree", path.Dir(tpath)),
			})

			tree, err = tree.Tree(tpath)
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
				fpath = path.Join("file", tpath, v.Name)
				rpath = path.Join(tpath, v.Name)
				isFile = true

				goit.SizesLock.RLock()
				sz, ok := goit.Sizes[v.Hash]
				goit.SizesLock.RUnlock()

				if !ok {
					file, err := tree.File(v.Name)
					if err != nil {
						log.Println("[/repo/tree]", err.Error())
						goit.HttpError(w, http.StatusInternalServerError)
						return
					}

					sz = uint64(file.Size)

					goit.SizesLock.Lock()
					goit.Sizes[v.Hash] = sz
					goit.SizesLock.Unlock()
				}

				size = humanize.IBytes(sz)
				totalSize += sz
			} else {
				fpath = path.Join("tree", tpath, v.Name)
				rpath = path.Join(tpath, v.Name)

				goit.SizesLock.RLock()
				sz, ok := goit.Sizes[v.Hash]
				goit.SizesLock.RUnlock()

				if !ok {
					dirt, err := tree.Tree(v.Name)
					if err != nil {
						log.Println("[/repo/tree]", err.Error())
						goit.HttpError(w, http.StatusInternalServerError)
						return
					}

					var dirSize uint64
					if err := dirt.Files().ForEach(func(f *object.File) error {
						dirSize += uint64(f.Size)
						return nil
					}); err != nil {
						log.Println("[/repo/tree]", err.Error())
						goit.HttpError(w, http.StatusInternalServerError)
						return
					}

					sz = dirSize

					goit.SizesLock.Lock()
					goit.Sizes[v.Hash] = sz
					goit.SizesLock.Unlock()
				}

				size = humanize.IBytes(sz)
				totalSize += sz
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
