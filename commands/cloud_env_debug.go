package commands

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/terminal"
)

var cloudEnvDebugCmd = &console.Command{
	Category: "cloud:environment",
	Name:     "debug",
	Aliases:  []*console.Alias{{Name: "environment:debug"}},
	Usage:    "Debug an environment by switching Symfony to the debug mode temporarily",
	Flags: []console.Flag{
		projectFlag,
		environmentFlag,
		&console.BoolFlag{Name: "off", Usage: "Disable debug mode"},
		&console.BoolFlag{Name: "debug", Usage: "Display commands output"},
	},
	Action: func(c *console.Context) error {
		spinner := terminal.NewSpinner(terminal.Stderr)
		spinner.Start()
		defer spinner.Stop()

		psh, err := platformsh.Get()
		if err != nil {
			return err
		}
		prefix := platformsh.GuessCloudFromCommandName(c.Command.UserName).CommandPrefix

		projectID := c.String("project")
		if projectID == "" {
			out, ok := psh.RunInteractive(terminal.Logger, "", []string{prefix + "project:info", "id", "-y"}, c.Bool("debug"), nil)
			if !ok {
				return errors.New("Unable to detect the project")
			}
			projectID = strings.TrimSpace(out.String())
		}

		out, ok := psh.RunInteractive(terminal.Logger, "", []string{prefix + "project:info", "default_branch", "--project=" + projectID, "-y"}, c.Bool("debug"), nil)
		if !ok {
			return errors.New("Unable to detect the default branch name")
		}
		defaultEnvName := strings.TrimSpace(out.String())
		envName := c.String("environment")
		if envName == "" {
			if out, ok := psh.RunInteractive(terminal.Logger, "", []string{prefix + "env:info", "id", "--project=" + projectID, "-y"}, false, nil); ok {
				envName = strings.TrimSpace(out.String())
			} else {
				envName = defaultEnvName
			}
		}

		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)

		defaultArgs := []string{"--level=env", "--project=" + projectID, "-y"}
		if c.String("environment") != "" {
			defaultArgs = append(defaultArgs, c.String("environment"))
		}

		if c.Bool("off") {
			terminal.Println("Deleting APP_ENV and APP_DEBUG (can take some time, --debug to tail commands)")
			if out, ok := psh.RunInteractive(terminal.Logger, "", append(defaultArgs, prefix+"var:delete", "env:APP_ENV"), c.Bool("debug"), nil); !ok {
				if !strings.Contains(out.String(), "Variable not found") {
					return errors.New("An error occurred while removing APP_ENV")
				}
			}
			if out, ok := psh.RunInteractive(terminal.Logger, "", append(defaultArgs, prefix+"var:delete", "env:APP_DEBUG"), c.Bool("debug"), nil); !ok {
				if !strings.Contains(out.String(), "Variable not found") {
					return errors.New("An error occurred while removing APP_DEBUG")
				}
			}
			ui.Success(fmt.Sprintf("The \"%s\" environment has been switched back to production mode.", envName))
			return nil
		}

		out, ok = psh.RunInteractive(terminal.Logger, "", []string{prefix + "project:info", "default_domain", "--project=" + projectID, "-y"}, c.Bool("debug"), nil)
		if !ok {
			return errors.New("Unable to detect the default domain")
		}
		defaultDomain := strings.TrimSpace(out.String())
		if defaultDomain != "" {
			if defaultEnvName == envName {
				return errors.Errorf("You cannot use the cloud:environment:debug command on the production environment (%s branch) of a production project", defaultEnvName)
			}
		}

		terminal.Println("Setting APP_ENV and APP_DEBUG to dev/debug (can take some time, --debug to tail commands)")
		if out, ok := psh.RunInteractive(terminal.Logger, "", append(defaultArgs, prefix+"var:create", "--name=env:APP_ENV", "--value=dev"), c.Bool("debug"), nil); !ok {
			if !strings.Contains(out.String(), "already exists on the environment") {
				return errors.New("An error occurred while adding APP_ENV")
			}
			if out, ok := psh.RunInteractive(terminal.Logger, "", append(defaultArgs, prefix+"var:update", "--value=dev", "env:APP_ENV"), c.Bool("debug"), nil); !ok {
				if !strings.Contains(out.String(), "No changes were provided") {
					return errors.New("An error occurred while adding APP_ENV")
				}
			}
		}
		if out, ok := psh.RunInteractive(terminal.Logger, "", append(defaultArgs, prefix+"var:create", "--name=env:APP_DEBUG", "--value=1"), c.Bool("debug"), nil); !ok {
			if !strings.Contains(out.String(), "already exists on the environment") {
				return errors.New("An error occurred while adding APP_DEBUG")
			}
			if out, ok := psh.RunInteractive(terminal.Logger, "", append(defaultArgs, prefix+"var:update", "--value=1", "env:APP_DEBUG"), c.Bool("debug"), nil); !ok {
				if !strings.Contains(out.String(), "No changes were provided") {
					return errors.New("An error occurred while adding APP_DEBUG")
				}
			}
		}

		spinner.Stop()
		ui.Success(fmt.Sprintf("The \"%s\" environment is now in debug mode\n     Switch back via %s env:debug --off", envName, c.App.HelpName))
		return nil
	},
}
