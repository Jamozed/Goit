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

func ModeString(mode uint32) string {
	s := "-"
	s += If((mode&0b100000000) != 0, "r", "-")
	s += If((mode&0b010000000) != 0, "w", "-")
	s += If((mode&0b001000000) != 0, "x", "-")
	s += If((mode&0b000100000) != 0, "r", "-")
	s += If((mode&0b000010000) != 0, "w", "-")
	s += If((mode&0b000001000) != 0, "x", "-")
	s += If((mode&0b000000100) != 0, "r", "-")
	s += If((mode&0b000000010) != 0, "w", "-")
	s += If((mode&0b000000001) != 0, "x", "-")
	return s
}
