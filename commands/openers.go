/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package commands

import (
	"fmt"
	"strings"

	"github.com/skratchdot/open-golang/open"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var projectLocalOpenCmd = &console.Command{
	Category: "open",
	Name:     "local",
	Usage:    "Open the local project in a browser",
	Flags: []console.Flag{
		dirFlag,
		&console.StringFlag{
			Name:  "path",
			Usage: "Default path which the project should open on",
		},
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		pidFile := pid.New(projectDir, nil)
		if !pidFile.IsRunning() {
			return console.Exit("Local web server is down.", 1)
		}
		host := fmt.Sprintf("127.0.0.1:%d", pidFile.Port)
		if proxyConf, err := proxy.Load(util.GetHomeDir()); err == nil {
			domains := proxyConf.GetReachableDomains(projectDir)
			if len(domains) > 0 {
				host = domains[0]
			}
		}
		abstractOpenCmd(fmt.Sprintf("%s://%s/%s",
			pidFile.Scheme,
			host,
			strings.TrimLeft(c.String("path"), "/"),
		))
		return nil
	},
}

var projectLocalMailCatcherOpenCmd = &console.Command{
	Category: "open",
	Name:     "local:webmail",
	Usage:    "Open the local project mail catcher web interface in a browser",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		env, err := envs.NewLocal(projectDir, terminal.IsDebug())
		if err != nil {
			return err
		}
		url, exists := env.FindServiceUrl("mailer")
		if !exists {
			return console.Exit("Mailcatcher Web interface not found", 1)
		}
		abstractOpenCmd(url)
		return nil
	},
}

var projectLocalRabbitMQManagementOpenCmd = &console.Command{
	Category: "open",
	Name:     "local:rabbitmq",
	Usage:    "Open the local project RabbitMQ web management interface in a browser",
	Flags: []console.Flag{
		dirFlag,
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		env, err := envs.NewLocal(projectDir, terminal.IsDebug())
		if err != nil {
			return err
		}
		url, exists := env.FindServiceUrl("amqp")
		if !exists {
			return console.Exit("RabbitMQ management not found", 1)
		}
		abstractOpenCmd(url)
		return nil
	},
}

var projectLocalServiceOpenCmd = &console.Command{
	Category: "open",
	Name:     "local:service",
	Usage:    "Open a local service web interface in a browser",
	Flags: []console.Flag{
		dirFlag,
	},
	Args: []*console.Arg{
		{Name: "service", Description: "The service name (or type) to open"},
	},
	Action: func(c *console.Context) error {
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		env, err := envs.NewLocal(projectDir, terminal.IsDebug())
		if err != nil {
			return err
		}
		service := c.Args().Get("service")
		url, exists := env.FindServiceUrl(service)
		if !exists {
			return console.Exit(fmt.Sprintf("Service \"%s\" not found", service), 1)
		}

		abstractOpenCmd(url)
		return nil
	},
}

func abstractOpenCmd(url string) {
	if err := open.Run(url); err != nil {
		terminal.Eprintln("<error>Error while opening:", err, "</>")
		terminal.Eprintfln("Please visit <href=%s>%s</> manually.", url, url)
	} else {
		terminal.Eprintfln("Opened: <href=%s>%s</>", url, url)
	}
}
