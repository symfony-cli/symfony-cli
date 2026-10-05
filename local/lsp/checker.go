/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
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

package lsp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/local/externaltool"
	"github.com/symfony-cli/terminal"
)

const (
	SymfonyCLIEnvironment = "SYMFONY_LSP_SYMFONY_CLI"
	ExitWrapperFailure    = 1
)

type Resolver interface {
	Resolve(ctx context.Context, definition externaltool.Definition) (externaltool.Installation, error)
}

type ProcessRunner interface {
	Run(executable string, arguments []string, directory string, environment []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)
}

type Checker struct {
	Resolver           Resolver
	Runner             ProcessRunner
	Definition         externaltool.Definition
	SymfonyCLIPath     func() (string, error)
	ProjectEnvironment func(string) ([]string, error)
	Stdin              io.Reader
	Stdout             io.Writer
	Stderr             io.Writer
}

func NewChecker(home string) *Checker {
	manager := externaltool.NewManager()
	manager.Stderr = terminal.Stderr

	return &Checker{
		Resolver:           manager,
		Runner:             NativeProcessRunner{},
		Definition:         ToolDefinition(home),
		SymfonyCLIPath:     os.Executable,
		ProjectEnvironment: loadProjectEnvironment,
		Stdin:              os.Stdin,
		Stdout:             os.Stdout,
		Stderr:             os.Stderr,
	}
}

func ToolCacheDir(home string) string {
	return filepath.Join(home, "tools", "symfony-language-tools")
}

func ToolDefinition(home string) externaltool.Definition {
	return externaltool.Definition{
		Name:           "Symfony Language Tools",
		Repository:     "symfony/language-tools",
		ChecksumsAsset: "SHA256SUMS",
		MinimumVersion: "0.17.0",
		VersionPrefix:  "Symfony Language Tools ",
		InstallRoot:    ToolCacheDir(home),
		Packages: []externaltool.Package{
			{OS: "linux", Arch: "amd64", Asset: "symfony-lsp-v{version}-linux-x64.tar.gz", ArchiveRoot: "symfony-lsp-v{version}-linux-x64", Executable: "symfony-lsp"},
			{OS: "linux", Arch: "arm64", Asset: "symfony-lsp-v{version}-linux-arm64.tar.gz", ArchiveRoot: "symfony-lsp-v{version}-linux-arm64", Executable: "symfony-lsp"},
			{OS: "darwin", Arch: "amd64", Asset: "symfony-lsp-v{version}-macos-x64.tar.gz", ArchiveRoot: "symfony-lsp-v{version}-macos-x64", Executable: "symfony-lsp"},
			{OS: "darwin", Arch: "arm64", Asset: "symfony-lsp-v{version}-macos-arm64.tar.gz", ArchiveRoot: "symfony-lsp-v{version}-macos-arm64", Executable: "symfony-lsp"},
			{OS: "windows", Arch: "amd64", Asset: "symfony-lsp-v{version}-windows-x64.zip", ArchiveRoot: "symfony-lsp-v{version}-windows-x64", Executable: "symfony-lsp.exe"},
		},
	}
}

func (c *Checker) Run(arguments []string) (int, error) {
	directory, err := os.Getwd()
	if err != nil {
		return ExitWrapperFailure, fmt.Errorf("unable to determine the project directory: %w", err)
	}
	installation, err := c.Resolver.Resolve(context.Background(), c.Definition)
	if err != nil {
		return ExitWrapperFailure, err
	}
	projectEnvironment, err := c.ProjectEnvironment(directory)
	if err != nil {
		return ExitWrapperFailure, fmt.Errorf("unable to prepare the project environment: %w", err)
	}
	symfonyCLI, err := c.SymfonyCLIPath()
	if err != nil {
		return ExitWrapperFailure, fmt.Errorf("unable to locate the Symfony CLI executable: %w", err)
	}
	symfonyCLI, err = filepath.Abs(symfonyCLI)
	if err != nil {
		return ExitWrapperFailure, fmt.Errorf("unable to resolve the Symfony CLI executable: %w", err)
	}
	environment := append(os.Environ(), projectEnvironment...)
	environment = append(environment, SymfonyCLIEnvironment+"="+symfonyCLI)

	return c.Runner.Run(
		installation.Executable,
		append([]string{"check"}, arguments...),
		directory,
		environment,
		c.Stdin,
		c.Stdout,
		c.Stderr,
	)
}

func loadProjectEnvironment(directory string) ([]string, error) {
	environment, err := envs.GetEnv(directory, terminal.IsDebug())
	if err != nil {
		return nil, err
	}
	values := envs.LoadDotEnv(envs.AsMap(environment), directory)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}

	return result, nil
}
