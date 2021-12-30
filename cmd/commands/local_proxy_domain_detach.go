package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localProxyDetachDomainCmd = &console.Command{
	Category: "local",
	Name:     "proxy:domain:detach",
	Aliases:  []*console.Alias{{Name: "proxy:domain:detach"}, {Name: "proxy:domain:remove", Hidden: true}},
	Usage:    "Detach domains from the proxy",
	Args: []*console.Arg{
		{Name: "domains", Optional: true, Description: "The domains to detach", Slice: true},
	},
	Action: func(c *console.Context) error {
		homeDir := util.GetHomeDir()
		config, err := proxy.Load(homeDir)
		if err != nil {
			return err
		}
		domains := c.Args().Tail()
		if err := config.RemoveDirDomains(domains); err != nil {
			return err
		}
		terminal.Println("<info>The following domains are not defined anymore on the proxy:</>")
		for _, domain := range domains {
			terminal.Printfln(" * http://%s", domain)
		}
		return nil
	},
}
