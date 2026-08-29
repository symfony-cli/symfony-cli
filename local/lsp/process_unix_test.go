//go:build !windows

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
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestNativeProcessRunnerForwardsTerminationSignals(t *testing.T) {
	if os.Getenv("SYMFONY_LSP_FORWARD_SIGNAL_HELPER") == "1" {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM)
		if err := os.WriteFile(os.Getenv("SYMFONY_LSP_SIGNAL_READY"), nil, 0600); err != nil {
			os.Exit(2)
		}
		<-signals
		if err := os.WriteFile(os.Getenv("SYMFONY_LSP_SIGNAL_RECEIVED"), nil, 0600); err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	directory := t.TempDir()
	ready := filepath.Join(directory, "ready")
	received := filepath.Join(directory, "received")
	result := make(chan int, 1)
	errors := make(chan error, 1)
	go func() {
		exitCode, err := (NativeProcessRunner{}).Run(
			os.Args[0],
			[]string{"-test.run=TestNativeProcessRunnerForwardsTerminationSignals"},
			directory,
			append(os.Environ(),
				"SYMFONY_LSP_FORWARD_SIGNAL_HELPER=1",
				"SYMFONY_LSP_SIGNAL_READY="+ready,
				"SYMFONY_LSP_SIGNAL_RECEIVED="+received,
			),
			bytes.NewReader(nil),
			io.Discard,
			io.Discard,
		)
		result <- exitCode
		errors <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("signal helper did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	if exitCode := <-result; exitCode != 0 {
		t.Fatalf("unexpected forwarded signal exit status %d", exitCode)
	}
	if _, err := os.Stat(received); err != nil {
		t.Fatalf("child did not receive the forwarded signal: %v", err)
	}
}

func TestNativeProcessRunnerEncodesSignalExitStatus(t *testing.T) {
	if os.Getenv("SYMFONY_LSP_SIGNAL_HELPER") == "1" {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		select {}
	}
	runner := NativeProcessRunner{}

	exitCode, err := runner.Run(
		os.Args[0],
		[]string{"-test.run=TestNativeProcessRunnerEncodesSignalExitStatus"},
		t.TempDir(),
		append(os.Environ(), "SYMFONY_LSP_SIGNAL_HELPER=1"),
		bytes.NewReader(nil),
		io.Discard,
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 128+int(syscall.SIGTERM) {
		t.Fatalf("unexpected signal exit status %d", exitCode)
	}
}
