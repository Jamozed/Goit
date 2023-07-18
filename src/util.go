// util.go
// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package goit

import "net/http"

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

func PanicIf(cond bool, v any) {
	if cond {
		panic(v)
	}
}

/* Return the named cookie or nil if not found. */
func Cookie(r *http.Request, name string) *http.Cookie {
	if c, err := r.Cookie(name); err != nil {
		return nil
	} else {
		return c
	}
}
