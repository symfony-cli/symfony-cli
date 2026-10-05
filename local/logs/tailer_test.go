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

package logs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/symfony-cli/symfony-cli/local/pid"
)

func TestWatchApplicationLogDirectories(t *testing.T) {
	// glob special characters in the project path must not be interpreted
	projectDir := filepath.Join(t.TempDir(), "client[1")
	logDir := filepath.Join(projectDir, "var", "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"prod.log", "test.log", "notes.txt"} {
		writeLog(t, filepath.Join(logDir, name), "")
	}

	tailer := startTailer(t, projectDir, nil)

	writeLog(t, filepath.Join(logDir, "prod.log"), "prod line\n")
	writeLog(t, filepath.Join(logDir, "test.log"), "test line\n")
	writeLog(t, filepath.Join(logDir, "notes.txt"), "notes line\n")
	// log files created after the tailer started must be followed as well
	writeLog(t, filepath.Join(logDir, "dev.log"), "dev line\n")

	assertLines(t, tailer, "prod line", "test line", "dev line")
}

func TestWatchApplicationLogDirectoryOnlyReplaysLatestLogFile(t *testing.T) {
	projectDir := t.TempDir()
	logDir := filepath.Join(projectDir, "var", "log")
	rotated := filepath.Join(logDir, "dev-2026-10-01.log")
	writeLog(t, rotated, "rotated line\n")
	writeLog(t, filepath.Join(logDir, "dev-2026-10-02.log"), "latest line\n")
	old := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(rotated, old, old); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", t.TempDir())
	tailer := &Tailer{Follow: true, LinesNb: 10, NoServerLogs: true, NoWorkerLogs: true}
	if err := tailer.Watch(pid.New(projectDir, nil)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	// older log files are followed from their end once written to
	writeLog(t, rotated, "new rotated line\n")

	assertLines(t, tailer, "latest line", "new rotated line")
}

func TestWatchApplicationLogFile(t *testing.T) {
	projectDir := t.TempDir()
	logDir := filepath.Join(projectDir, "var", "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(projectDir, "logs", "custom.log")

	tailer := startTailer(t, projectDir, []string{custom})

	writeLog(t, filepath.Join(logDir, "dev.log"), "dev line\n")
	writeLog(t, custom, "custom line\n")

	assertLines(t, tailer, "custom line")
}

func startTailer(t *testing.T, projectDir string, appLogs []string) *Tailer {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	tailer := &Tailer{Follow: true, AppLogs: appLogs, NoServerLogs: true, NoWorkerLogs: true}
	if err := tailer.Watch(pid.New(projectDir, nil)); err != nil {
		t.Fatal(err)
	}
	// let the tailers and watchers start before writing
	time.Sleep(500 * time.Millisecond)
	return tailer
}

func writeLog(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func assertLines(t *testing.T, tailer *Tailer, expected ...string) {
	t.Helper()
	want := map[string]bool{}
	for _, line := range expected {
		want[line] = true
	}
	timeout := time.After(5 * time.Second)
	for len(want) > 0 {
		select {
		case line := <-tailer.lines:
			if !want[line.line.Text] {
				t.Fatalf("unexpected line %q", line.line.Text)
			}
			delete(want, line.line.Text)
		case <-timeout:
			t.Fatalf("lines not tailed: %v", want)
		}
	}
	select {
	case line := <-tailer.lines:
		t.Fatalf("unexpected line %q", line.line.Text)
	case <-time.After(time.Second):
	}
}
