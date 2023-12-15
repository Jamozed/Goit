// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package util

import "log"

var Debug = false

func Debugln(v ...any) {
	if Debug {
		var a = []any{"\033[34m[DEBUG]\033[0m"}
		a = append(a, v...)
		log.Println(a...)
	}
}
