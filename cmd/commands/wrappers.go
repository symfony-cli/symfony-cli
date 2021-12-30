package commands

import (
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
)

var (
	composerWrapper = &console.Command{
		Name:   "composer",
		Usage:  "Runs Composer without memory limit",
		Hidden: console.Hide,
		Action: func(c *console.Context) error {
			return console.IncorrectUsageError{ParentError: errors.New(`This command can only be run as "symfony composer"`)}
		},
	}
	binConsoleWrapper = &console.Command{
		Name:   "console",
		Usage:  "Runs the Symfony Console (bin/console) for current project",
		Hidden: console.Hide,
		Action: func(c *console.Context) error {
			return console.IncorrectUsageError{ParentError: errors.New(`This command can only be run as "symfony console"`)}
		},
	}
	phpWrapper = &console.Command{
		Usage:  "Runs the named binary using the configured PHP version",
		Hidden: console.Hide,
		Action: func(c *console.Context) error {
			return console.IncorrectUsageError{ParentError: errors.New(`This command can only be run as "symfony php*"`)}
		},
	}
)

func init() {
	for _, name := range php.GetBinaryNames() {
		phpWrapper.Aliases = append(phpWrapper.Aliases, &console.Alias{Name: name, Hidden: console.Hide()})
	}
}
