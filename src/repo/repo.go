package repo

import (
	"log"
	"net/http"
	"regexp"
	"strconv"

	goit "github.com/Jamozed/Goit/src"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type HeaderFields struct {
	Name, Description, Url, Readme, Licence string
	Editable                                bool
}

var readmePattern = regexp.MustCompile(`(?i)^readme(?:\.?(?:md|txt))?$`)
var licencePattern = regexp.MustCompile(`(?i)^licence(?:\.?(?:md|txt))?$`)

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	auth, admin, uid := goit.AuthCookieAdmin(w, r, true)

	rid, err := strconv.ParseInt(r.URL.Query().Get("repo"), 10, 64)
	if err != nil {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	repo, err := goit.GetRepo(rid)
	if err != nil {
		log.Println("[/repo/delete]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || (uid != repo.OwnerId && !admin) {
		goit.HttpError(w, http.StatusUnauthorized)
		return
	}

	if err := goit.DelRepo(repo.Name); err != nil {
		log.Println("[/repo/delete]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func findReadme(gr *git.Repository, ref *plumbing.Reference) (string, error) {
	commit, err := gr.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}

	iter, err := commit.Files()
	if err != nil {
		return "", err
	}

	var filename string
	if err := iter.ForEach(func(f *object.File) error {
		if readmePattern.MatchString(f.Name) {
			filename = f.Name
			return storer.ErrStop
		}

		return nil
	}); err != nil {
		return "", err
	}

	return filename, nil
}

func findLicence(gr *git.Repository, ref *plumbing.Reference) (string, error) {
	commit, err := gr.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}

	iter, err := commit.Files()
	if err != nil {
		return "", err
	}

	var filename string
	if err := iter.ForEach(func(f *object.File) error {
		if licencePattern.MatchString(f.Name) {
			filename = f.Name
			return storer.ErrStop
		}

		return nil
	}); err != nil {
		return "", err
	}

	return filename, nil
}
