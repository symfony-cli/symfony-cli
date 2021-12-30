package commands

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
)

var localServerProdCmd = &console.Command{
	Category: "local",
	Name:     "server:prod",
	Aliases:  []*console.Alias{{Name: "server:prod"}},
	Usage:    "Switch a project to use Symfony's production environment",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "off", Usage: "Disable prod mode"},
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		beacon := filepath.Join(projectDir, ".prod")
		if c.Bool("off") {
			return errors.WithStack(os.Remove(beacon))
		}
		if f, err := os.OpenFile(beacon, os.O_RDONLY|os.O_CREATE, 0666); err != nil {
			return errors.WithStack(err)
		} else {
			f.Close()
		}
		return nil
	},
}
