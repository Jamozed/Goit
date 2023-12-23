// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type HeaderFields struct {
	Name, Description, Url, Readme, Licence string
	Editable                                bool
}

var readmePattern = regexp.MustCompile(`(?i)^readme(?:\.?(?:md|txt))?$`)
var licencePattern = regexp.MustCompile(`(?i)^licence(?:\.?(?:md|txt))?$`)

/* Find a file that matches a regular expression in the root level of a reference. */
func findPattern(gr *git.Repository, ref *plumbing.Reference, re *regexp.Regexp) (string, error) {
	c, err := gr.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}

	t, err := c.Tree()
	if err != nil {
		return "", err
	}

	for _, e := range t.Entries {
		if re.MatchString(e.Name) {
			return e.Name, nil
		}
	}

	return "", nil
}
