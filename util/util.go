package util

import (
	"os/user"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

func GetHomeDir() string {
	return filepath.Join(getUserHomeDir(), ".symfony5")
}

func getUserHomeDir() string {
	if InCloud() {
		u, err := user.Current()
		if err != nil {
			return "/tmp"
		}
		return "/tmp/" + u.Username
	}

	if homeDir, err := homedir.Dir(); err == nil {
		return homeDir
	}

	return "."
}
