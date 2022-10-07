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
	"github.com/symfony-cli/terminal"
)

var openDocCmd = &console.Command{
	Category: "open",
	Name:     "docs",
	Usage:    "Open the online Web documentation",
	Action: func(c *console.Context) error {
		abstractOpenCmd("https://symfony.com/doc/cloud")
		return nil
	},
}

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
		&console.StringFlag{
			Name:  "domain",
			Usage: "Which domain the project should open on (Default: 127.0.0.1)",
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
		domain := strings.TrimSpace(c.String("domain"))
		if domain == "" {
			domain = "127.0.0.1"
		}
		abstractOpenCmd(fmt.Sprintf("%s://%s:%d/%s",
			pidFile.Scheme,
			domain,
			pidFile.Port,
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
		prefix := env.FindRelationshipPrefix("mailer", "http")
		values := envs.AsMap(env)
		url, exists := values[prefix+"URL"]
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
		prefix := env.FindRelationshipPrefix("amqp", "http")
		values := envs.AsMap(env)
		url, exists := values[prefix+"URL"]
		if !exists {
			return console.Exit("RabbitMQ management not found", 1)
		}
		abstractOpenCmd(url)
		return nil
	},
}

func abstractOpenCmd(url string) {
	if err := open.Run(url); err != nil {
		terminal.Eprintln("<error>Error while opening:", err, "</>")
		terminal.Eprintfln("Please visit <href=%>%s</> manually.", url)
	} else {
		terminal.Eprintfln("Opened: <href=%s>%s</>", url, url)
	}
}
