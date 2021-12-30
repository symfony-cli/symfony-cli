package commands

import (
	"os"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/terminal"
)

var localVariableExposeFromTunnelCmd = &console.Command{
	Category: "local",
	Name:     "var:expose-from-tunnel",
	Aliases:  []*console.Alias{{Name: "var:expose-from-tunnel"}},
	Usage:    "Expose tunnel service environment variables locally",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "off", Usage: "Stop exposing tunnel service environment variables"},
	},
	Action: func(c *console.Context) error {
		dir := c.String("dir")
		if dir == "" {
			var err error
			if dir, err = os.Getwd(); err != nil {
				return err
			}
		}

		tunnel := envs.Tunnel{
			Dir: dir,
		}

		if c.Bool("off") {
			terminal.Eprintln("Stop exposing tunnel service environment variables ")
			return tunnel.Expose(false)
		}

		terminal.Eprintln("Exposing tunnel service environment variables")

		if err := tunnel.Expose(true); err != nil {
			return err
		}

		return nil
	},
}
