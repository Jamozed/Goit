// util/util.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package util

import (
	"net/http"
)

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
