package php

import (
	"os"

	"path/filepath"

	"github.com/pkg/errors"
)

// SymfonyConsoleExecutor returns an Executor prepared to run Symfony Console.
// It returns an error if no console binary is found.
func SymfonyConsoleExecutor(projectDir string, args []string) (*Executor, error) {
	for {
		for _, consolePath := range []string{"bin/console", "app/console"} {
			consolePath = filepath.Join(projectDir, consolePath)
			if _, err := os.Stat(consolePath); err == nil {
				return &Executor{
					BinName: "php",
					Args:    append([]string{"php", consolePath}, args...),
				}, nil
			}
		}

		upDir := filepath.Dir(projectDir)
		if upDir == projectDir || upDir == "." {
			break
		}
		projectDir = upDir
	}

	return nil, errors.New("No console binary found")
}
