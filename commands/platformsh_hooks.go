package commands

import (
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/terminal"
)

var platformshBeforeHooks = map[string]console.BeforeFunc{
	"cloud:tunnel:close": func(c *console.Context) error {
		terminal.Eprintln("Stop exposing tunnel service environment variables")

		app := c.String("app")
		env := c.String("environment")
		var project *platformsh.Project
		projectID := c.String("project")
		if projectID == "" {
			projectDir, err := getProjectDir(c.String("dir"))
			if err != nil {
				return errors.WithStack(err)
			}
			project, err = platformsh.ProjectFromDir(projectDir, false)
			if err != nil {
				return errors.WithStack(err)
			}
			if app != "" {
				project.App = app
			}
			if env != "" {
				project.Env = env
			}
		} else {
			project = &platformsh.Project{
				ID:  projectID,
				App: app,
				Env: env,
			}
		}

		tunnel := envs.Tunnel{Project: project}
		return tunnel.Expose(false)
	},
}
