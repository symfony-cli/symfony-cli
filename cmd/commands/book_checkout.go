package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/book"
	"github.com/symfony-cli/terminal"
)

var BookCheckoutCmd = &console.Command{
	Category: "book",
	Name:     "checkout",
	Usage:    `Check out a step of the "Symfony 5: The Fast Track" book repository`,
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "debug", Usage: "Display commands output"},
		&console.BoolFlag{Name: "force", Usage: "Force the use of the command without checking pre-requisites"},
	},
	Args: []*console.Arg{
		{Name: "step", Description: "The step of the book to checkout (code at the end of the step)"},
	},
	Action: func(c *console.Context) error {
		dir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}

		book := &book.Book{
			Dir:   dir,
			Debug: c.Bool("debug"),
			Force: c.Bool("force"),
		}
		if !c.Bool("force") {
			if err := book.CheckRepository(); err != nil {
				return err
			}
		}
		if err := book.Checkout(c.Args().Get("step")); err != nil {
			terminal.Println("")
			if !c.Bool("debug") {
				terminal.Println("Re-run the command with <comment>--debug</> to get more information about the error")
				terminal.Println("")
			}
			return err
		}
		return nil
	},
}
