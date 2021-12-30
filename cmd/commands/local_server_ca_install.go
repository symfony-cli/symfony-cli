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

var localServerCAInstallCmd = &console.Command{
	Category: "local",
	Name:     "server:ca:install",
	Aliases:  []*console.Alias{{Name: "server:ca:install"}},
	Usage:    "Create a local Certificate Authority for serving HTTPS",
	Flags: []console.Flag{
		&console.BoolFlag{Name: "renew", Usage: "Force generating a new CA"},
		&console.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Force reinstalling current CA"},
	},
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		homeDir := util.GetHomeDir()
		certsDir := filepath.Join(homeDir, "certs")
		ca, err := cert.NewCA(certsDir)
		if err != nil {
			return err
		}
		newCA := false
		renew := c.Bool("renew")

	retry:
		if !ca.HasCA() {
			if err := ca.CreateCA(); err != nil {
				return errors.Wrap(err, "failed to generate the local Certificate Authority")
			}
			newCA = true
		}
		if err = ca.LoadCA(); err != nil {
			return errors.Wrap(err, "failed to load the local Certificate Authority")
		}
		if renew && !newCA {
			ca.Uninstall()
			os.RemoveAll(certsDir)
			renew = false

			goto retry
		}
		if err = ca.Install(c.Bool("force")); err != nil {
			return errors.Wrap(err, "failed to install the local Certificate Authority")
		}
		p12 := filepath.Join(homeDir, "certs", "default.p12")
		if _, err := os.Stat(p12); os.IsNotExist(err) {
			terminal.Println("Generating a default certificate for HTTPS support")
			err := ca.MakeCert(p12, []string{"localhost", "127.0.0.1", "::1"})
			if err != nil {
				return errors.Wrap(err, "failed to generate a default certificate for localhost")
			}
		}

		ui.Success("The local Certificate Authority is installed and trusted")
		return nil
	},
}
