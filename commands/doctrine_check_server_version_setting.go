package commands

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/terminal"
)

var doctrineCheckServerVersionSettingCmd = &console.Command{
	Name:   "doctrine:check-server-version-setting",
	Usage:  "Check if Doctrine server version is configured explicitly",
	Hidden: console.Hide,
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		logger := terminal.Logger.Output(zerolog.ConsoleWriter{Out: terminal.Stderr}).With().Timestamp().Logger()
		if err := checkDoctrineServerVersionSetting(projectDir, logger); err != nil {
			return err
		}

		fmt.Println("✅ Doctrine server version is set properly.")
		return nil
	},
}
