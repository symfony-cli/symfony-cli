package php

import (
	"os"

	"github.com/pkg/errors"
	"path/filepath"
)

// ComposerExecutor returns an Executor prepared to run Symfony Console.
// It returns an error if no console binary is found.
func SymonyConsoleExecutor(args []string) (*Executor, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for {
		for _, consolePath := range []string{"bin/console", "app/console"} {
			consolePath = filepath.Join(dir, consolePath)
			if _, err := os.Stat(consolePath); err == nil {
				return &Executor{
					BinName: "php",
					Args:    append([]string{"php", consolePath}, args...),
				}, nil
			}
		}

		upDir := filepath.Dir(dir)
		if upDir == dir || upDir == "." {
			break
		}
		dir = upDir
	}

	return nil, errors.New("No console binary found")
}
