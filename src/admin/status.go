// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package admin

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/Jamozed/Goit/res"
	"github.com/Jamozed/Goit/src/goit"
	"github.com/Jamozed/Goit/src/util"
	"github.com/dustin/go-humanize"
)

func HandleStatus(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)

	data := struct {
		Title, Version, Uptime string
		Goroutines             int
		Memory, Stack, Heap    string
	}{
		Title:      "Admin - Status",
		Version:    res.Version,
		Uptime:     formatUptime(time.Since(goit.StartTime)),
		Goroutines: runtime.NumGoroutine(),
		Memory:     humanize.Bytes(mem.Sys),
		Stack:      humanize.Bytes(mem.StackInuse),
		Heap:       humanize.Bytes(mem.HeapInuse),
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/status", data); err != nil {
		log.Println("[/admin/status]", err.Error())
	}
}

func formatUptime(uptime time.Duration) string {
	const (
		day  = time.Hour * 24
		week = day * 7
	)

	weeks := int64(uptime / week)
	uptime -= time.Duration(weeks) * week
	days := int64(uptime / day)
	uptime -= time.Duration(days) * day
	hours := int64(uptime / time.Hour)
	uptime -= time.Duration(hours) * time.Hour
	minutes := int64(uptime / time.Minute)
	uptime -= time.Duration(minutes) * time.Minute
	seconds := int64(uptime / time.Second)

	var parts []string
	if weeks > 0 {
		parts = append(parts, fmt.Sprintf(util.If(weeks == 1, "%d week", "%d weeks"), weeks))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf(util.If(days == 1, "%d day", "%d days"), days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf(util.If(days == 1, "%d hour", "%d hours"), hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf(util.If(days == 1, "%d minute", "%d minutes"), minutes))
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf(util.If(days == 1, "%d second", "%d seconds"), seconds))
	}

	if len(parts) == 0 {
		return "0 seconds"
	}

	return strings.Join(parts, ", ")
}
