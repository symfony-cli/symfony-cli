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
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/reexec"
	"github.com/symfony-cli/symfony-cli/updater"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var (
	psh     *platformshCLI
	pshOnce sync.Once

	dirFlag         = &console.StringFlag{Name: "dir", Usage: "Project directory"}
	projectFlag     = &console.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "The project ID or URL"}
	environmentFlag = &console.StringFlag{Name: "environment", Aliases: []string{"e"}, Usage: "The environment ID"}
)

func CommonCommands() []*console.Command {
	adminCommands := []*console.Command{
		// common cloud commands
		// FIXME: this command should be renamed (it's a special command as it's local but creates cloud files)
		projectInitCmd,
		// FIXME: this command should be renamed (it works locally AND in the cloud)
		variableExportCmd,
		// commands that we override to provide more features
		//environmentSSHCmd,
	}
	localCommands := []*console.Command{
		binConsoleWrapper,
		composerWrapper,
		phpWrapper,
		bookCheckReqsCmd,
		bookCheckoutCmd,
		cloudEnvDebugCmd,
		localNewCmd,
		localPhpListCmd,
		localPhpRefreshCmd,
		localProxyAttachDomainCmd,
		localProxyDetachDomainCmd,
		localProxyStartCmd,
		localProxyStatusCmd,
		localProxyUrlCmd,
		localProxyStopCmd,
		localRequirementsCheckCmd,
		localRunCmd,
		localServerCAInstallCmd,
		localServerCAUninstallCmd,
		localServerListCmd,
		localServerLogCmd,
		localServerProdCmd,
		localServerStartCmd,
		localServerStatusCmd,
		localServerStopCmd,
		localVariableExposeFromTunnelCmd,
		localSecurityCheckCmd,
		projectLocalMailCatcherOpenCmd,
		projectLocalRabbitMQManagementOpenCmd,
		projectLocalServiceOpenCmd,
		projectLocalOpenCmd,
	}
	return append(localCommands, adminCommands...)
}

func CheckGitIsAvailable(c *console.Context) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("Git is a requirement and it cannot be found, please install it first.")
	}

	return nil
}

func init() {
	initCLI()
}

func GetPSH() (*platformshCLI, error) {
	var err error
	pshOnce.Do(func() {
		psh, err = NewPlatformShCLI()
		if err != nil {
			err = errors.Wrap(err, "Unable to setup Platform.sh CLI")
		}
	})
	return psh, err
}

func InitAppFunc(c *console.Context) error {
	if c.App.Channel == "stable" {
		// do not run auto-update in the cloud, CI or background jobs
		if !util.InCloud() && terminal.Stdin.IsInteractive() && !reexec.IsChild() {
			debug := false
			if os.Getenv("SC_DEBUG") == "1" {
				debug = true
			}
			updater := updater.NewUpdater(filepath.Join(util.GetHomeDir(), "update"), c.App.ErrWriter, debug)
			updater.CheckForNewVersion(c.App.Version)
		}
	}

	return nil
}

// WelcomeAction displays a message when no command
func WelcomeAction(c *console.Context) error {
	console.ShowVersion(c)
	terminal.Println(c.App.Usage)
	terminal.Println("")
	terminal.Println("These are common commands used in various situations:")
	terminal.Println("")
	terminal.Println("<comment>Work on a project locally</>")
	terminal.Println("")
	displayCommandsHelp(c, []*console.Command{
		localNewCmd,
		localServerStartCmd,
		localServerStopCmd,
		localSecurityCheckCmd,
		composerWrapper,
		binConsoleWrapper,
		phpWrapper,
	})
	terminal.Println("")
	terminal.Println("<comment>Manage a project on Cloud</>")
	terminal.Println("")
	psh, err := GetPSH()
	if err != nil {
		return errors.WithStack(err)
	}
	displayCommandsHelp(c, append([]*console.Command{projectInitCmd}, psh.PSHMainCommands()...))
	terminal.Println("")
	terminal.Printf("Show all commands with <info>%s help</>,\n", c.App.HelpName)
	terminal.Printf("Get help for a specific command with <info>%s help COMMAND</>.\n", c.App.HelpName)
	return nil
}

func displayCommandsHelp(c *console.Context, cmds []*console.Command) {
	console.HelpPrinter(c.App.Writer, `{{range .}}  <info>{{.PreferredName}}</>{{"\t"}}{{.Usage}}{{"\n"}}{{end}}`, cmds)
}

func initCLI() {
	formatter := terminal.Stdout.GetFormatter()
	formatter.SetStyle("sc", terminal.NewFormatterStyle("cyan", "", nil))
	formatter.SetStyle("bold", terminal.NewFormatterStyle("", "", []string{"bold"}))
	formatter.SetStyle("failure", terminal.NewFormatterStyle("red", "", nil))
	formatter.AddAlias("notes", "warning")

	console.AppHelpTemplate += `
<comment>Available wrappers:</>
Runs PHP (version depends on project's configuration).
Environment variables to use Platform.sh relationships or Docker services are automatically defined.

{{with .Command "composer"}}  <info>{{.PreferredName}}</>{{"\t"}}{{.Usage}}{{end}}
{{with .Command "console"}}  <info>{{.PreferredName}}</>{{"\t"}}{{.Usage}}{{end}}
{{with .Command "php"}}  <info>{{.PreferredName}}</>{{"\t"}}{{.Usage}}{{end}}

`
}

func getProjectDir(dir string) (string, error) {

	var err error
	if dir, err = filepath.Abs(dir); err != nil {
		return "", errors.WithStack(err)
	}

	link, e := filepath.EvalSymlinks(dir)

	return link, errors.WithStack(e)
}
