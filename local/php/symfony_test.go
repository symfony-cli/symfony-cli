/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
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

package php

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/symfony-cli/symfony-cli/util"
)

func TestSymfonyConsoleInvocationEnvironment(t *testing.T) {
	for _, test := range []struct {
		name           string
		consolePath    string
		workDir        string
		customPath     bool
		customLauncher bool
		launcher       string
	}{
		{name: "bin console", consolePath: "bin/console"},
		{name: "legacy console from subdirectory", consolePath: "app/console", workDir: "src", customLauncher: true},
		{name: "custom console", consolePath: "tools/custom-console", customPath: true},
		{name: "Windows launcher", consolePath: "bin/console", launcher: `C:\Program Files\Symfony\symfony.exe`},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, filepath.FromSlash(test.consolePath))
			if err := os.MkdirAll(filepath.Dir(script), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(script, nil, 0644); err != nil {
				t.Fatal(err)
			}
			workDir := filepath.Join(dir, test.workDir)
			if err := os.MkdirAll(workDir, 0755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(workDir)
			t.Setenv("SYMFONY_CONSOLE_PATH", test.consolePath)
			if !test.customPath {
				if err := os.Unsetenv("SYMFONY_CONSOLE_PATH"); err != nil {
					t.Fatal(err)
				}
			}

			launcher := test.launcher
			if launcher == "" {
				launcher = "symfony"
			}
			if test.customLauncher {
				launcher = filepath.Join(dir, "my symfony link")
			}
			prepareConsoleEnvironmentTest(t, launcher)
			executor, err := SymfonyConsoleExecutor(zerolog.Nop(), []string{"cache:clear", "--env=test", "--help"})
			if err != nil {
				t.Fatal(err)
			}
			environment := executeConsoleEnvironmentTest(t, executor)
			var command []string
			if err := json.Unmarshal([]byte(environment["SYMFONY_CLI_CONSOLE_COMMAND"]), &command); err != nil {
				t.Fatalf("invalid console command: %v", err)
			}
			if expected := []string{launcher, "console"}; !reflect.DeepEqual(expected, command) {
				t.Fatalf("console command = %q, want %q", command, expected)
			}
			if actual := environment["SYMFONY_CLI_CONSOLE_SCRIPT"]; actual != script {
				t.Fatalf("console script = %q, want %q", actual, script)
			}
		})
	}
}

func TestExecutorClearsInheritedConsoleInvocation(t *testing.T) {
	t.Chdir(t.TempDir())
	prepareConsoleEnvironmentTest(t, "symfony")
	environment := executeConsoleEnvironmentTest(t, &Executor{
		BinName: "php",
		Args:    []string{"php", "other.php"},
		Logger:  zerolog.Nop(),
	})
	for _, name := range []string{"SYMFONY_CLI_CONSOLE_COMMAND", "SYMFONY_CLI_CONSOLE_SCRIPT"} {
		if value, exists := environment[name]; !exists || value != "" {
			t.Errorf("%s = %q (exists: %t), want an empty value", name, value, exists)
		}
	}
}

func prepareConsoleEnvironmentTest(t *testing.T, launcher string) {
	t.Helper()
	t.Cleanup(util.OverrideHomeDir(t.TempDir()))
	t.Setenv("PLATFORM_PROJECT_ENTROPY", "console-invocation-test")
	t.Setenv("SYMFONY_CLI_CONSOLE_COMMAND", `["stale","console"]`)
	t.Setenv("SYMFONY_CLI_CONSOLE_SCRIPT", "/stale/bin/console")
	testBinary := os.Args[0]
	originalArgs := os.Args
	os.Args = []string{launcher, "console", "cache:clear", "--help"}
	t.Cleanup(func() { os.Args = originalArgs })
	originalExecCommand := execCommand
	execCommand = func(string, ...string) *exec.Cmd {
		cmd := exec.Command(testBinary, "-test.run=^TestHelperProcess$", "--", "dump-env")
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}
	t.Cleanup(func() { execCommand = originalExecCommand })
}

func executeConsoleEnvironmentTest(t *testing.T, executor *Executor) map[string]string {
	t.Helper()
	executor.ExtraEnv = []string{
		"SYMFONY_CLI_CONSOLE_COMMAND=extra-stale-command",
		"SYMFONY_CLI_CONSOLE_SCRIPT=extra-stale-script",
	}
	var output bytes.Buffer
	executor.Stdout = &output
	executor.Stderr = &output
	if code := executor.Execute(false); code != 0 {
		t.Fatalf("executor exited with code %d: %s", code, output.String())
	}
	environment := make(map[string]string)
	for line := range strings.Lines(output.String()) {
		line = strings.TrimRight(line, "\r\n")
		if name, value, ok := strings.Cut(line, "="); ok {
			environment[name] = value
		}
	}
	return environment
}
