// util/util.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package util

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

const ModeNotRegular = os.ModeSymlink | os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice |
	os.ModeIrregular

func If[T any](cond bool, a, b T) T {
	if cond {
		return a
	} else {
		return b
	}
}

func SliceContains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}

	return false
}

/* Return the named cookie or nil if not found or invalid. */
func Cookie(r *http.Request, name string) *http.Cookie {
	c, err := r.Cookie(name)
	if err == nil && c.Valid() == nil {
		return c
	}

	return nil
}

func ModeString(mode uint32) string {
	s := If((mode&0o40000) != 0, "d", "-")
	s += If((mode&0o400) != 0, "r", "-")
	s += If((mode&0o200) != 0, "w", "-")
	s += If((mode&0o100) != 0, "x", "-")
	s += If((mode&0o040) != 0, "r", "-")
	s += If((mode&0o020) != 0, "w", "-")
	s += If((mode&0o010) != 0, "x", "-")
	s += If((mode&0o004) != 0, "r", "-")
	s += If((mode&0o002) != 0, "w", "-")
	s += If((mode&0o001) != 0, "x", "-")
	return s
}

func DirSize(path string) (uint64, error) {
	var size int64

	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}

			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := d.Info()
		if err != nil {
			return err
		}

		if (f.Mode() & ModeNotRegular) == 0 {
			size += f.Size()
		}

		return nil
	})

	return uint64(size), err
}
