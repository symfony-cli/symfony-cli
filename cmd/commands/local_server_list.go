package commands

import (
	"sort"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/projects"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/terminal"
)

var localServerListCmd = &console.Command{
	Category: "local",
	Name:     "server:list",
	Aliases:  []*console.Alias{{Name: "server:list"}},
	Usage:    "List all configured local web servers",
	Action: func(c *console.Context) error {
		return printConfiguredServers()
	},
}

func printConfiguredServers() error {
	table := tablewriter.NewWriter(terminal.Stdout)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{terminal.Format("<header>Directory</>"), terminal.Format("<header>Port</>"), terminal.Format("<header>Domains</>")})

	proxyProjects, err := proxy.ToConfiguredProjects()
	if err != nil {
		return errors.WithStack(err)
	}
	runningProjects, err := pid.ToConfiguredProjects()
	if err != nil {
		return errors.WithStack(err)
	}
	projects, err := projects.GetConfiguredAndRunning(proxyProjects, runningProjects)
	if err != nil {
		return errors.WithStack(err)
	}
	projectDirs := []string{}
	for dir := range projects {
		projectDirs = append(projectDirs, dir)
	}
	sort.Strings(projectDirs)
	for _, dir := range projectDirs {
		project := projects[dir]
		domain := ""
		if len(project.Domains) > 0 {
			domain = terminal.Formatf("<href=%s://%s>%s</>", project.Scheme, project.Domains[0], project.Domains[0])
		}
		port := "Not running"
		if project.Port > 0 {
			port = terminal.Formatf("<href=%s://127.0.0.1:%d>%d</>", project.Scheme, project.Port, project.Port)
		}
		table.Append([]string{dir, port, domain})
		if len(project.Domains) > 1 {
			for i, domain := range project.Domains {
				if i == 0 {
					continue
				}
				table.Append([]string{"", "", terminal.Formatf("<href=%s://%s>%s</>", project.Scheme, domain, domain)})
			}
		}
	}
	table.Render()
	return nil
}
