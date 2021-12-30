package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localPhpRefreshCmd = &console.Command{
	Category: "local",
	Name:     "php:refresh",
	Usage:    "Auto-discover the list of available PHP version",
	Action: func(c *console.Context) error {
		phpstore.New(util.GetHomeDir(), true, nil)
		terminal.Println("<info>Available PHP versions refreshed!</>")
		return nil
	},
}
