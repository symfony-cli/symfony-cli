/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/git"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/terminal"
)

var projectInitCmd = &console.Command{
	Category: "project",
	Name:     "init",
	Aliases:  []*console.Alias{{Name: "init"}},
	Usage:    "Initialize a new project using templates",
	Description: `Initialize a new project using templates.
Templates used by this tool are fetched from ` + templatesGitRepository + `.
`,
	Flags: []console.Flag{
		dirFlag,
		&console.StringFlag{Name: "template", Usage: "Project template to use", DefaultText: "autodetermined"},
		&console.StringFlag{Name: "title", Usage: "Project title", DefaultText: "autodetermined based on directory name"},
		&console.StringFlag{Name: "slug", DefaultValue: "app", Usage: "Project slug"},
		&console.StringFlag{Name: "php", Usage: "PHP version to use"},
		&console.BoolFlag{Name: "upsun", Usage: "Initialize Upsun"},
		// FIXME: services should also be used to configure Docker? Instead of Flex?
		// FIXME: services can also be guessed via the existing Docker Compose file?
		&console.StringSliceFlag{Name: "service", Usage: "Configure some services", Hidden: true},
		&console.BoolFlag{Name: "dump", Usage: "Dump file content instead of writing them on disk"},
		&console.BoolFlag{Name: "force", Usage: "Force the overwrite of the files even if they already exists", Hidden: true},
	},
	Before: CheckGitIsAvailable,
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)

		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		minorPHPVersion, err := forcePHPVersion(c.String("php"), projectDir)
		if err != nil {
			return err
		}

		if buf, err := gitInit(projectDir); err != nil {
			fmt.Print(buf.String())
			return err
		}
		slug := c.String("slug")
		if slug == "" {
			slug = "app"
		}

		cloudServices, err := parseCloudServices(projectDir, c.StringSlice("service"))
		if err != nil {
			return err
		}

		brand := platformsh.PlatformshBrand
		if c.Bool("upsun") {
			brand = platformsh.UpsunBrand
		}
		createdFiles, err := createRequiredFilesProject(brand, projectDir, slug, c.String("template"), minorPHPVersion, cloudServices, c.Bool("dump"), c.Bool("force"))
		if err != nil {
			return err
		}

		if c.Bool("dump") {
			return nil
		}

		terminal.Println("\n<info>Project configured</>")
		terminal.Println("")

		if len(createdFiles) > 0 {
			terminal.Println("The following files were created automatically:")
			for _, file := range createdFiles {
				terminal.Println("", file)
			}
			terminal.Println("")

			ui.Section("Next Steps")

			terminal.Println(" * Adapt the generated files if needed")
			terminal.Printf(" * Commit them: <info>git add %s && git commit -m\"Add %s configuration\"</>\n", strings.Join(createdFiles, " "), brand)
			terminal.Printf(" * Deploy: <info>%s deploy</>\n", c.App.HelpName)
		} else {
			terminal.Printf("Deploy the project via <info>%s deploy</>.\n", c.App.HelpName)
		}

		return nil
	},
}

func gitInit(cwd string) (*bytes.Buffer, error) {
	if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil || !os.IsNotExist(err) {
		return nil, nil
	}

	// project:init is only used in a Cloud context, so we can safely force the
	// branch to be "main"
	return git.Init(cwd, true, false)
}
