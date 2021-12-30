package commands

import (
	"fmt"

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
		abstractOpenCmd(fmt.Sprintf("%s://127.0.0.1:%d", pidFile.Scheme, pidFile.Port))
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
