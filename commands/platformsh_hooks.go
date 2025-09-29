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
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/upsun"
	"github.com/symfony-cli/terminal"
)

var upsunBeforeHooks = map[string]console.BeforeFunc{
	"environment:push": func(c *console.Context) error {
		// check that project has a DB and that server version is set properly
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		return checkDoctrineServerVersionSetting(projectDir, zerolog.Nop())
	},
	"tunnel:close": func(c *console.Context) error {
		terminal.Eprintln("Stop exposing tunnel service environment variables")

		app := c.String("app")
		env := c.String("environment")
		var project *upsun.Project
		projectID := c.String("project")
		if projectID == "" {
			projectDir, err := getProjectDir(c.String("dir"))
			if err != nil {
				return err
			}
			project, err = upsun.ProjectFromDir(projectDir, false)
			if err != nil {
				return err
			}
			if app != "" {
				project.App = app
			}
			if env != "" {
				project.Env = env
			}
		} else {
			project = &upsun.Project{
				ID:  projectID,
				App: app,
				Env: env,
			}
		}

		tunnel := envs.Tunnel{Project: project}
		return tunnel.Expose(false)
	},
}
