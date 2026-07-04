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

package pid

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("PIDFILE_TEST_HELPER_PROCESS") == "1" {
		io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// startHelperProcess starts a child process that runs until its stdin is closed.
func startHelperProcess(t *testing.T) (*exec.Cmd, io.WriteCloser) {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "PIDFILE_TEST_HELPER_PROCESS=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd, stdin
}

func TestIsRunning(t *testing.T) {
	if (&PidFile{}).IsRunning() {
		t.Error("a pid file without a pid should not be reported as running")
	}
	if !(&PidFile{Pid: os.Getpid()}).IsRunning() {
		t.Error("the current process should be reported as running")
	}

	cmd, stdin := startHelperProcess(t)
	p := &PidFile{Pid: cmd.Process.Pid}
	if !p.IsRunning() {
		t.Error("a live process should be reported as running")
	}

	stdin.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if p.IsRunning() {
		t.Error("an exited process should not be reported as running")
	}
}

func TestRemovePidFileKeepsLogFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	p := &PidFile{Dir: dir, path: filepath.Join(dir, "test.pid")}

	if err := os.WriteFile(p.PidFile(), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p.LogFile()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.LogFile(), []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := p.RemovePidFile(); err != nil {
		t.Fatalf("RemovePidFile should succeed, got: %s", err)
	}
	if _, err := os.Stat(p.PidFile()); !os.IsNotExist(err) {
		t.Error("the pid file should have been removed")
	}
	if _, err := os.Stat(p.LogFile()); err != nil {
		t.Errorf("the log file should have been kept, got: %s", err)
	}
	// removing a missing pid file is not an error
	if err := p.RemovePidFile(); err != nil {
		t.Errorf("RemovePidFile on a missing file should succeed, got: %s", err)
	}
}

func TestRemoveDeletesPidAndLogFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	p := &PidFile{Dir: dir, path: filepath.Join(dir, "test.pid")}

	if err := os.WriteFile(p.PidFile(), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p.LogFile()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.LogFile(), []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := p.Remove(); err != nil {
		t.Fatalf("Remove should succeed, got: %s", err)
	}
	if _, err := os.Stat(p.PidFile()); !os.IsNotExist(err) {
		t.Error("the pid file should have been removed")
	}
	if _, err := os.Stat(p.LogFile()); !os.IsNotExist(err) {
		t.Error("the log file should have been removed")
	}
}

func TestRemoveDeletesLeftoverWorkerLogs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	p := &PidFile{Dir: dir, Args: []string{"worker"}, path: filepath.Join(dir, "server.pid")}

	if err := os.MkdirAll(p.WorkerLogDir(), 0755); err != nil {
		t.Fatal(err)
	}
	workerLog := filepath.Join(p.WorkerLogDir(), "worker.log")
	if err := os.WriteFile(workerLog, []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := p.Remove(); err != nil {
		t.Fatalf("Remove should succeed, got: %s", err)
	}
	if _, err := os.Stat(workerLog); !os.IsNotExist(err) {
		t.Error("leftover worker log files should have been removed")
	}
}

func TestWriteRefusesOnlyLiveProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pid")
	cmd, stdin := startHelperProcess(t)

	if err := (&PidFile{path: path}).Write(cmd.Process.Pid, 0, "http"); err != nil {
		t.Fatalf("unable to write pid file: %s", err)
	}
	if err := (&PidFile{path: path}).Write(os.Getpid(), 0, "http"); err == nil {
		t.Error("writing a pid file over a live process should be refused")
	}

	stdin.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := (&PidFile{path: path}).Write(os.Getpid(), 0, "http"); err != nil {
		t.Errorf("writing a pid file over an exited process should succeed, got: %s", err)
	}
}
