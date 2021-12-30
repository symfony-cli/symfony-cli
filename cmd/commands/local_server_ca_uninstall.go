package commands

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/symfony-cli/cert"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localServerCAUninstallCmd = &console.Command{
	Category: "local",
	Name:     "server:ca:uninstall",
	Aliases:  []*console.Alias{{Name: "server:ca:uninstall"}},
	Usage:    "Uninstall the local Certificate Authority",
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		certsDir := filepath.Join(util.GetHomeDir(), "certs")
		ca, err := cert.NewCA(certsDir)
		if err != nil {
			return nil
		}
		if !ca.HasCA() {
			ui.Success("The local Certificate Authority is not installed yet")
			return nil
		}
		if err = ca.LoadCA(); err != nil {
			return errors.Wrap(err, "failed to load the local Certificate Authority")
		}
		ca.Uninstall()
		os.RemoveAll(certsDir)
		ui.Success("The local Certificate Authority has been uninstalled")
		return nil
	},
}
