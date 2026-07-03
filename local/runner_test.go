//go:build !windows

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

package local

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/symfony-cli/symfony-cli/local/pid"
)

func TestRunnerWaitsBeforeRestartingFailingCommand(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	projectDir := filepath.Join(homeDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// the first run exits successfully, the second one fails, then the
	// command parks so that the runner stops spawning new processes
	runsLog := filepath.Join(projectDir, "runs.log")
	script := filepath.Join(projectDir, "cmd.sh")
	scriptContent := `#!/bin/sh
echo run >> ` + runsLog + `
n=$(wc -l < ` + runsLog + `)
if [ "$n" -le 1 ]; then exit 0; fi
if [ "$n" -le 2 ]; then exit 1; fi
sleep 30
`
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	pidFile := pid.New(projectDir, []string{"/bin/sh", script})
	runner, err := NewRunner(pidFile, RunnerModeLoopAttached)
	if err != nil {
		t.Fatal(err)
	}
	runner.AlwaysRestartOnExit = true

	go func() { _ = runner.Run() }()

	starts := make([]time.Time, 0, 3)
	deadline := time.Now().Add(30 * time.Second)
	for len(starts) < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("expected the command to run 3 times, got %d runs", len(starts))
		}
		if b, err := os.ReadFile(runsLog); err == nil {
			for n := strings.Count(string(b), "\n"); len(starts) < n; {
				starts = append(starts, time.Now())
			}
		}
		time.Sleep(5 * time.Millisecond)
	}

	if gap := starts[2].Sub(starts[1]); gap < 4500*time.Millisecond {
		t.Errorf("restart after a failure happened after only %s, expected a 5 seconds delay", gap)
	}
}
