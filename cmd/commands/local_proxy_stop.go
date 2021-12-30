package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

var localProxyStopCmd = &console.Command{
	Category: "local",
	Name:     "proxy:stop",
	Aliases:  []*console.Alias{{Name: "proxy:stop"}},
	Usage:    "Stop the local proxy server",
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		p := pid.New("__proxy__", nil)
		if !p.IsRunning() {
			ui.Success("The proxy server is not running")
			return nil
		}
		if err := p.Stop(); err != nil {
			return err
		}
		ui.Success("Stopped the proxy server successfully")
		return nil
	},
}
