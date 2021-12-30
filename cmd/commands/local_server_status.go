package commands

import (
	"fmt"
	"strings"

	"github.com/symfony-cli/console"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localServerStatusCmd = &console.Command{
	Category: "local",
	Name:     "server:status",
	Aliases:  []*console.Alias{{Name: "server:status"}},
	Usage:    "Get the local web server status",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		return printWebServerStatus(projectDir)
	},
}

func printWebServerStatus(projectDir string) error {
	pidFile := pid.New(projectDir, nil)
	workers := pid.AllWorkers(projectDir)

	// web server
	terminal.Println("<info>Local Web Server</>")
	if !pidFile.IsRunning() {
		terminal.Println("    <error>Not Running</>")
	} else {
		terminal.Printfln("    Listening on <href=%s://127.0.0.1:%d>%s://127.0.0.1:%d</>", pidFile.Scheme, pidFile.Port, pidFile.Scheme, pidFile.Port)
		homeDir := util.GetHomeDir()
		phpStore := phpstore.New(homeDir, true, nil)
		version, _, _, err := phpStore.BestVersionForDir(projectDir)
		if err == nil {
			terminal.Printfln("    The Web server is using <info>%s %s</>", version.ServerTypeName(), version.Version)
		}
		terminal.Println()
		terminal.Println("<info>Local Domains</>")
		if proxyConf, err := proxy.Load(util.GetHomeDir()); err == nil {
			for _, domain := range proxyConf.GetDomains(projectDir) {
				terminal.Printfln("    <href=%s://%s>%s://%s</>", pidFile.Scheme, domain, pidFile.Scheme, domain)
			}
		}
	}

	// workers
	terminal.Println()
	terminal.Println("<info>Workers</info>")
	if len(workers) == 0 {
		terminal.Println("    <warning>No Workers</>")
	} else {
		for _, p := range workers {
			msg := fmt.Sprintf(`    PID <info>%d</>: %s`, p.Pid, p.Command())
			if len(p.Watched) > 0 {
				msg += fmt.Sprintf(" (watching <comment>%s/</comment>)", strings.Join(p.Watched, "/, "))
			}
			terminal.Println(msg)
		}
	}

	// env vars
	terminal.Println()
	terminal.Println("<info>Environment Variables</>")
	data, err := envs.GetEnv(projectDir, terminal.IsDebug())
	if err != nil {
		return err
	}
	env := envs.AsMap(data)
	envVars := `<comment>None</>`
	if env["SYMFONY_TUNNEL"] != "" && env["SYMFONY_TUNNEL_ENV"] != "" {
		envVars = `Exposed from <info>SymfonyCloud</>`
	}
	if env["SYMFONY_DOCKER_ENV"] == "1" && env["SYMFONY_TUNNEL_ENV"] == "" {
		envVars = `Exposed from <info>Docker</>`
	}
	terminal.Printfln("    %s", envVars)

	return nil
}
