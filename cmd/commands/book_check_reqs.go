package commands

import (
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/book"
	"github.com/symfony-cli/terminal"
)

var BookCheckReqsCmd = &console.Command{
	Category: "book",
	Name:     "check-requirements",
	Usage:    `Check that you have all the pre-requisites locally to code while reading the "Symfony 5: The Fast Track" book`,
	Aliases:  []*console.Alias{{Name: "book:check"}},
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)

		ready, err := book.CheckRequirements()
		if err != nil {
			return err
		}
		terminal.Println("")
		if ready {
			ui.Success("Congrats! You are ready to start reading the book.")
			return nil
		}
		return console.Exit("You should fix the reported issues before starting reading the book.", 1)
	},
}
