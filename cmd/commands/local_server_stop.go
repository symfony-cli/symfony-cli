package commands

import (
	"fmt"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
	"golang.org/x/sync/errgroup"
)

var localServerStopCmd = &console.Command{
	Category: "local",
	Name:     "server:stop",
	Aliases:  []*console.Alias{{Name: "server:stop"}},
	Usage:    "Stop the local web server",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		webserver := pid.New(projectDir, nil)
		pids := append(pid.AllWorkers(projectDir), webserver)
		var g errgroup.Group
		running := 0
		for _, p := range pids {
			terminal.Printf("Stopping <comment>%s</>", p.ShortName())
			if p.IsRunning() {
				running++
				g.Go(p.Stop)
				terminal.Println("")
			} else {
				terminal.Println(": <comment>not running</>")
			}
		}

		terminal.Println("")
		if err := g.Wait(); err != nil {
			return err
		}
		if running == 0 {
			ui.Success("The web server is not running")
		} else {
			ui.Success(fmt.Sprintf("Stopped %d process(es) successfully", running))
		}
		return nil
	},
}
