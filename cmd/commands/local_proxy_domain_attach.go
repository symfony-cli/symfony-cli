package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localProxyAttachDomainCmd = &console.Command{
	Category: "local",
	Name:     "proxy:domain:attach",
	Aliases:  []*console.Alias{{Name: "proxy:domain:attach"}, {Name: "proxy:domain:add", Hidden: true}},
	Usage:    "Attach a local domain for the proxy",
	Flags: []console.Flag{
		dirFlag,
	},
	Args: []*console.Arg{
		{Name: "domains", Optional: true, Description: "The project's domains", Slice: true},
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		homeDir := util.GetHomeDir()
		config, err := proxy.Load(homeDir)
		if err != nil {
			return err
		}
		if err := config.AddDirDomains(projectDir, c.Args().Tail()); err != nil {
			return err
		}
		terminal.Println("<info>The proxy is now configured with the following domains for this directory:</>")
		for _, domain := range config.GetDomains(projectDir) {
			terminal.Printfln(" * http://%s", domain)
		}
		return nil
	},
}
