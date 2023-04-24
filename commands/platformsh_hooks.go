package commands

import (
	"errors"
	"fmt"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/terminal"
)

var platformshBeforeHooks = map[string]console.BeforeFunc{
	"cloud:environment:push": func(c *console.Context) error {
		// check that project has a DB and that server version is set properly
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		if len(platformsh.FindLocalApplications(projectDir)) > 0 {
			// not implemented yet as more complex
			return nil
		}

		dbName, dbVersion, err := readDBVersionFromPlatformServiceYAML(projectDir)
		if err != nil {
			return nil
		}
		if dbName == "" {
			// no DB
			return nil
		}

		errorTpl := fmt.Sprintf(`
The ".platform/services.yaml" file defines
a "%s" version %s database service
but %%s.

Before deploying, fix the version mismatch.
`, dbName, dbVersion)

		dotEnvVersion, err := readDBVersionFromDotEnv(projectDir)
		if err != nil {
			return nil
		}
		if dotEnvVersion != "" && dotEnvVersion != dbVersion {
			return fmt.Errorf(errorTpl, fmt.Sprintf("the \".env\" file requires version %s", dotEnvVersion))
		}

		doctrineConfigVersion, err := readDBVersionFromDoctrineConfigYAML(projectDir)
		if err != nil {
			return nil
		}
		if doctrineConfigVersion != "" && doctrineConfigVersion != dbVersion {
			return fmt.Errorf(errorTpl, fmt.Sprintf("the \"config/packages/doctrine.yaml\" file requires version %s", doctrineConfigVersion))
		}

		if dotEnvVersion == "" && doctrineConfigVersion == "" {
			return errors.New(`
The ".platform/services.yaml" file defines a "%s" database service.

When deploying, Doctrine needs to know the database version to determine the supported SQL syntax.

As the database is not available when Doctrine is warming up its cache on Platform.sh,
you need to explicitely set the database version in the ".env" or "config/packages/doctrine.yaml" file.

The easiest is to set the "serverVersion" parameter in the "config/packages/doctrine.yaml" file.
`)
		}

		return nil
	},
	"cloud:tunnel:close": func(c *console.Context) error {
		terminal.Eprintln("Stop exposing tunnel service environment variables")

		app := c.String("app")
		env := c.String("environment")
		var project *platformsh.Project
		projectID := c.String("project")
		if projectID == "" {
			projectDir, err := getProjectDir(c.String("dir"))
			if err != nil {
				return err
			}
			project, err = platformsh.ProjectFromDir(projectDir, false)
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
