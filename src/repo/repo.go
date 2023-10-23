package repo

import (
	"regexp"

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
