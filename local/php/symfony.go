package php

import (
	"os"

	"github.com/pkg/errors"
)

// ComposerExecutor returns an Executor prepared to run Symfony Console.
// It returns an error if no console binary is found.
func SymonyConsoleExecutor(args []string) (*Executor, error) {
	consolePath := "bin/console"

	if _, err := os.Stat(consolePath); err != nil {
		// Fallback to app/console for projects created with older versions of Symfony
		consolePath = "app/console"

		if _, err2 := os.Stat(consolePath); err2 != nil {
			return nil, errors.WithStack(err)
		}
	}

	return &Executor{
		BinName: "php",
		Args:    append([]string{"php", consolePath}, args...),
	}, nil
}
