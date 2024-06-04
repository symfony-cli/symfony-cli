package commands

import (
	"github.com/symfony-cli/console"
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

		return checkDoctrineServerVersionSetting(projectDir)
	},
}
