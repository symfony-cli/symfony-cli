package commands

import (
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/terminal"
)

var variableExportCmd = &console.Command{
	Category: "var",
	Name:     "export",
	Usage:    "Export environment variables depending on the current context",
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "multiline", Usage: "Display each export on its own line"},
		&console.BoolFlag{Name: "debug", Usage: "Debug Docker support"},
	},
	Args: []*console.Arg{
		{Name: "name", Optional: true, Description: "Print the value of this environment variable"},
	},
	Action: func(c *console.Context) error {
		dir := c.String("dir")
		if dir == "" {
			var err error
			if dir, err = os.Getwd(); err != nil {
				return err
			}
		}
		env, err := envs.GetEnv(dir, c.Bool("debug"))
		if err != nil {
			return err
		}

		if name := c.Args().Get("name"); name != "" {
			if v, ok := envs.AsMap(env)[name]; ok {
				terminal.Print(v)
				return nil
			}
			return errors.Errorf("no environment variable with name %s", name)
		}

		if c.Bool("multiline") {
			m := envs.AsMap(env)
			keys := make([]string, 0)
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				terminal.Printfln("export %s=%s", k, m[k])
			}
		} else {
			// output the string (useful when doing export $(envs))
			terminal.Print(envs.AsString(env))
		}

		return nil
	},
}
