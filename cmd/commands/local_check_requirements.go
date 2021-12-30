package commands

import (
	_ "embed"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

// To generate, run in symfony/requirements-checker
// php bin/release.php > data/check-requirements.php
//go:embed data/check-requirements.php
var phpChecker []byte

var localRequirementsCheckCmd = &console.Command{
	Category: "local",
	Name:     "check:requirements",
	Aliases:  []*console.Alias{{Name: "check:requirements"}, {Name: "check:req"}},
	Usage:    "Checks requirements for running Symfony and gives useful recommendations to optimize PHP for Symfony.",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		path := c.String("dir")
		if path == "" {
			var err error
			path, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		cacheDir := filepath.Join(util.GetHomeDir(), "cache")
		if _, err := os.Stat(cacheDir); err != nil {
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return err
			}
		}

		cachePath := filepath.Join(cacheDir, "check.php")
		defer os.Remove(cachePath)
		if err := ioutil.WriteFile(cachePath, phpChecker, 0600); err != nil {
			return err
		}

		args := []string{"php", cachePath}
		if terminal.IsVerbose() {
			args = append(args, "-v")
		}
		if c.String("dir") != "" {
			args = append(args, path)
		}
		e := &php.Executor{
			Dir:     path,
			BinName: "php",
			Args:    args,
		}
		if ret := e.Execute(false); ret != 0 {
			return console.Exit("", 1)
		}

		return nil
	},
}
