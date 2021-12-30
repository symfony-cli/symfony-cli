package book

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

func (b *Book) Clone(version string) error {
	ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
	ui.Section("Checking Book Requirements")
	ready, err := CheckRequirements()
	if err != nil {
		return err
	}
	terminal.Println("")
	if !ready {
		return errors.New("You should fix the reported issues before starting reading the book.")
	}

	ui.Section("Cloning the Repository")
	cmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/the-fast-track/book-%s", version), b.Dir)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error cloning the Git repository for the book")
	}
	terminal.Println("")

	os.Chdir(b.Dir)
	// checkout the first step by default
	ui.Section("Getting Ready for the First Step of the Book")
	if err := b.Checkout("3"); err != nil {
		terminal.Println("")
		if !b.Debug {
			terminal.Println("Re-run the command with <comment>--debug</> to get more information about the error")
			terminal.Println("")
		}
		return err
	}
	return nil
}
