package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

var localProxyStatusCmd = &console.Command{
	Category: "local",
	Name:     "proxy:status",
	Aliases:  []*console.Alias{{Name: "proxy:status"}},
	Usage:    "Get the local proxy server status",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		terminal.Println("<info>Local Proxy Server</>")

		pidFile := pid.New("__proxy__", nil)
		if !pidFile.IsRunning() {
			terminal.Println("    <error>Not Running</>")
			return nil
		}

		terminal.Printfln("    Listening on <href=%s://127.0.0.1:%d>%s://127.0.0.1:%d</>", pidFile.Scheme, pidFile.Port, pidFile.Scheme, pidFile.Port)

		terminal.Println()
		terminal.Println("<info>Configured Web Servers</>")
		return printConfiguredServers()
	},
}
