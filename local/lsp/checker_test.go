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
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/symfony-cli/symfony-cli/local/externaltool"
)

type fakeResolver struct {
	installation externaltool.Installation
	err          error
}

func (r *fakeResolver) Resolve(context.Context, externaltool.Definition) (externaltool.Installation, error) {
	return r.installation, r.err
}

type capturingProcessRunner struct {
	executable  string
	arguments   []string
	directory   string
	environment []string
	exitCode    int
	err         error
}

func (r *capturingProcessRunner) Run(executable string, arguments []string, directory string, environment []string, _ io.Reader, stdout, stderr io.Writer) (int, error) {
	r.executable = executable
	r.arguments = slices.Clone(arguments)
	r.directory = directory
	r.environment = slices.Clone(environment)
	_, _ = io.WriteString(stdout, "report")
	_, _ = io.WriteString(stderr, "details")

	return r.exitCode, r.err
}

func TestCheckerDelegatesArgumentsStreamsEnvironmentAndExitStatus(t *testing.T) {
	directory := t.TempDir()
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDirectory)
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	resolver := &fakeResolver{installation: externaltool.Installation{Executable: "/managed/symfony-lsp", Version: "0.17.0"}}
	runner := &capturingProcessRunner{exitCode: 10}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	checker := &Checker{
		Resolver:   resolver,
		Runner:     runner,
		Definition: ToolDefinition("/cache"),
		SymfonyCLIPath: func() (string, error) {
			return "./symfony", nil
		},
		ProjectEnvironment: func(projectDirectory string) ([]string, error) {
			if projectDirectory != workingDirectory {
				t.Fatalf("unexpected project directory %q", projectDirectory)
			}
			return []string{"DATABASE_URL=mysql://database", "APP_ENV=test"}, nil
		},
		Stdin:  bytes.NewBufferString("input"),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	exitCode, err := checker.Run([]string{"--format=json", "src", "--", "-literal"})
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 10 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}
	if runner.executable != "/managed/symfony-lsp" {
		t.Fatalf("unexpected executable %q", runner.executable)
	}
	expectedArguments := []string{"check", "--format=json", "src", "--", "-literal"}
	if !slices.Equal(runner.arguments, expectedArguments) {
		t.Fatalf("unexpected arguments %#v", runner.arguments)
	}
	if runner.directory != workingDirectory {
		t.Fatalf("unexpected working directory %q", runner.directory)
	}
	if !slices.Contains(runner.environment, "DATABASE_URL=mysql://database") || !slices.Contains(runner.environment, "APP_ENV=test") {
		t.Fatalf("project environment was not forwarded: %#v", runner.environment)
	}
	expectedCLI := SymfonyCLIEnvironment + "=" + filepath.Join(workingDirectory, "symfony")
	if !slices.Contains(runner.environment, expectedCLI) {
		t.Fatalf("Symfony CLI handoff was not forwarded: %#v", runner.environment)
	}
	if stdout.String() != "report" || stderr.String() != "details" {
		t.Fatalf("streams were not preserved: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCheckerDoesNotStartAfterWrapperFailure(t *testing.T) {
	runner := &capturingProcessRunner{}
	checker := &Checker{
		Resolver:   &fakeResolver{err: errors.New("release unavailable")},
		Runner:     runner,
		Definition: ToolDefinition(t.TempDir()),
		SymfonyCLIPath: func() (string, error) {
			return "/symfony", nil
		},
		ProjectEnvironment: func(string) ([]string, error) { return nil, nil },
		Stdin:              bytes.NewReader(nil),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	exitCode, err := checker.Run(nil)
	if exitCode != ExitWrapperFailure || err == nil || err.Error() != "release unavailable" {
		t.Fatalf("unexpected wrapper result: exit=%d err=%v", exitCode, err)
	}
	if runner.executable != "" {
		t.Fatalf("checker started after wrapper failure: %q", runner.executable)
	}
}

func TestProjectEnvironmentIncludesProjectDotenvValues(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte("APP_ENV=test\nAPP_SECRET=project-secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV", "")

	environment, err := loadProjectEnvironment(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(environment, "APP_ENV=test") || !slices.Contains(environment, "APP_SECRET=project-secret") {
		t.Fatalf("dotenv values were not prepared: %#v", environment)
	}
}

func TestToolDefinitionPublishesSupportedPlatformsAndMinimumVersion(t *testing.T) {
	definition := ToolDefinition("/configuration")
	if definition.MinimumVersion != "0.17.0" || definition.ChecksumsAsset != "SHA256SUMS" {
		t.Fatalf("unexpected release contract %#v", definition)
	}
	if definition.InstallRoot != filepath.Join("/configuration", "tools", "symfony-language-tools") {
		t.Fatalf("unexpected cache directory %q", definition.InstallRoot)
	}
	platforms := make([]string, 0, len(definition.Packages))
	for _, pkg := range definition.Packages {
		platforms = append(platforms, pkg.OS+"/"+pkg.Arch+":"+pkg.Asset)
	}
	expected := []string{
		"linux/amd64:symfony-lsp-v{version}-linux-x64.tar.gz",
		"linux/arm64:symfony-lsp-v{version}-linux-arm64.tar.gz",
		"darwin/amd64:symfony-lsp-v{version}-macos-x64.tar.gz",
		"darwin/arm64:symfony-lsp-v{version}-macos-arm64.tar.gz",
		"windows/amd64:symfony-lsp-v{version}-windows-x64.zip",
	}
	if !slices.Equal(platforms, expected) {
		t.Fatalf("unexpected platforms %#v", platforms)
	}
}

func TestNativeProcessRunnerPreservesStreamsAndExitStatus(t *testing.T) {
	if os.Getenv("SYMFONY_LSP_RUNNER_HELPER") == "1" {
		_, _ = io.WriteString(os.Stdout, "machine-report")
		_, _ = io.WriteString(os.Stderr, "operational-details")
		os.Exit(12)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NativeProcessRunner{}

	exitCode, err := runner.Run(
		os.Args[0],
		[]string{"-test.run=TestNativeProcessRunnerPreservesStreamsAndExitStatus"},
		t.TempDir(),
		append(os.Environ(), "SYMFONY_LSP_RUNNER_HELPER=1"),
		bytes.NewReader(nil),
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 12 || stdout.String() != "machine-report" || stderr.String() != "operational-details" {
		t.Fatalf("unexpected process result: exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}
