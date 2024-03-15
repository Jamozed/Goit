// Copyright (C) 2024, Jakob Wakeling
// All rights reserved.

package goit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type config struct {
	DataPath    string `json:"data_path"`
	LogsPath    string `json:"logs_path"`
	RuntimePath string `json:"runtime_path"`
	HttpAddr    string `json:"http_addr"`
	HttpPort    string `json:"http_port"`
	GitPath     string `json:"git_path"`
	IpSessions  bool   `json:"ip_sessions"`
	UsesHttps   bool   `json:"uses_https"`
	IpForwarded bool   `json:"ip_forwarded"`
	CsrfSecret  string `json:"csrf_secret"`
}

func loadConfig() (config, error) {
	conf := config{
		DataPath:    dataPath(),
		LogsPath:    logsPath(),
		RuntimePath: runtimePath(),
		HttpAddr:    "",
		HttpPort:    "8080",
		GitPath:     "git",
		IpSessions:  true,
		UsesHttps:   false,
		IpForwarded: false,
		CsrfSecret:  "1234567890abcdef1234567890abcdef",
	}

	/* Load config file(s) */
	configs := []string{
		filepath.Join("/etc", "goit", "goit.conf"),
	}

	if os.Getuid() != 0 {
		configs = append(configs, filepath.Join(userConfigBase(), "goit", "goit.conf"))
	}

	for _, file := range configs {
		if data, err := os.ReadFile(file); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return config{}, err
			}
		} else if data != nil {
			if err := json.Unmarshal(data, &conf); err != nil {
				return config{}, err
			}
		}
	}

	/* Check required config values */
	if conf.DataPath == "" {
		return config{}, errors.New("data path unset")
	}

	return conf, nil
}

func userConfigBase() string {
	if path := os.Getenv("XDG_CONFIG_HOME"); path != "" {
		return path
	}

	if path := os.Getenv("HOME"); path != "" {
		return filepath.Join(path, ".config")
	}

	return ""
}

func dataPath() string {
	if os.Getuid() == 0 {
		return "/var/lib/goit"
	}

	if path := os.Getenv("XDG_DATA_HOME"); path != "" {
		return filepath.Join(path, "goit")
	}

	if path := os.Getenv("HOME"); path != "" {
		return filepath.Join(path, ".local", "share", "goit")
	}

	return ""
}

func logsPath() string {
	if os.Getuid() == 0 {
		return "/var/log/goit"
	}

	if path := os.Getenv("XDG_STATE_HOME"); path != "" {
		return filepath.Join(path, "goit")
	}

	if path := os.Getenv("HOME"); path != "" {
		return filepath.Join(path, ".local", "state", "goit")
	}

	return ""
}

func runtimePath() string {
	if os.Getuid() == 0 {
		return "/run"
	}

	if path := os.Getenv("XDG_RUNTIME_DIR"); path != "" {
		return filepath.Join(path)
	}

	return ""
}
