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

package reexec

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/inotify"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
	"github.com/syncthing/notify"
)

const UP = "up"

func ExecBinaryWithEnv(binary string, envs []string) bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}

	files := []*os.File{
		os.Stdin,
		os.Stdout,
		os.Stderr,
	}

	p, err := os.StartProcess(binary, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   append(os.Environ(), envs...),
		Files: files,
		Sys:   &syscall.SysProcAttr{},
	})
	if err != nil {
		return false
	}

	done := make(chan bool)
	go func() {
		state, err := p.Wait()
		if err != nil {
			fmt.Println(err)
			done <- false
			return
		}
		if !state.Success() {
			done <- false
			return
		}
		done <- true
	}()

	select {
	case state := <-done:
		return state
	case <-time.After(10 * time.Second):
		p.Kill()
		return false
	}
}

func Background(homeDir string) error {
	if util.IsGoRun() {
		return errors.New("Not applicable in a Go run context")
	}

	statusFile, err := ioutil.TempFile(homeDir, "status-")
	if err != nil {
		return errors.Wrap(err, "Could not create status file")
	}
	statusFile.Close()
	defer os.Remove(statusFile.Name())

	watcherChan := make(chan inotify.EventInfo, 10)
	if err := inotify.Watch(statusFile.Name(), watcherChan, inotify.Write, inotify.Remove); err != nil {
		return errors.Wrap(err, "Could not watch status file")
	}
	defer inotify.Stop(watcherChan)

	statusCh := make(chan int)

	os.Setenv("REEXEC_STATUS_FILE", statusFile.Name())
	os.Setenv("REEXEC_WATCH_PID", fmt.Sprint(Getppid()))
	// We are in a reexec.Restart call (probably after an upgrade), watch
	// REEXEC_SHELL_PID instead of direct parent.
	// For interactive sessions, this is not an issue, but for long-running
	// processes like tunnel:open, if we don't do that, they will exit right
	// after returning back to the shell because the direct parent (the initial
	// process that got upgraded) is the one watched.
	if shellPID := os.Getenv("REEXEC_SHELL_PID"); shellPID != "" {
		os.Setenv("REEXEC_WATCH_PID", shellPID)
	}

	terminal.Logger.Debug().Msg("Let's go to the background!")
	p, err := Respawn()
	if err != nil {
		return errors.Wrap(err, "Could not respawn")
	}

	ticker := time.NewTicker(5 * time.Second)

	go func() {
		status := 0
		state, err := p.Wait()
		if err != nil {
			status = 1
		} else if !state.Success() {
			exiterr := exec.ExitError{ProcessState: state}
			// This will work on Windows and Unix
			if s, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				status = s.ExitStatus()
			}
		}

		statusCh <- status
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)

	for {
		select {
		case sig := <-sigChan:
			if shouldSignalBeIgnored(sig) {
				continue
			}
			if err := p.Signal(sig); err != nil {
				p.Kill()
				return errors.Wrapf(err, "error sending signal %s", sig)
			}
			// if the signal terminates the process we will loop over and
			// end-up receiving a status in statusCh so no particular
			// processing to do here.
		case event := <-watcherChan:
			terminal.Logger.Info().Msg("FS event received: " + event.Event().String())
			if event.Event() == notify.Remove {
				return nil
			}

			if event.Event() == notify.Write {
				ticker.Stop()
				break
			}
		case status := <-statusCh:
			return console.Exit("", status)
		case <-ticker.C:
			p.Kill()
			return errors.New("reexec timed out")
		}
	}
}

func NotifyForeground(status string) error {
	if !IsChild() {
		return nil
	}
	statusFile := os.Getenv("REEXEC_STATUS_FILE")
	if statusFile == "" {
		return nil
	}
	if UP == status {
		os.Unsetenv("REEXEC_STATUS_FILE")
		os.Unsetenv("REEXEC_WATCH_PID")
		os.Stdin.Close()
		os.Stdout.Close()
		os.Stderr.Close()
		return os.Remove(statusFile)
	}
	return ioutil.WriteFile(statusFile, []byte(status), 0600)
}

func WatchParent(stopCh chan bool) error {
	spid := os.Getenv("REEXEC_WATCH_PID")
	if spid == "" {
		return nil
	}

	pid, err := strconv.Atoi(spid)
	if err != nil {
		return errors.WithStack(err)
	}

	parent, err := os.FindProcess(pid)
	if err != nil {
		return errors.WithStack(err)
	}
	if parent == nil {
		return errors.Errorf("Can't find process %d", pid)
	}

	go func() {
		waitForProcess(parent)
		terminal.Logger.Info().Msgf("Parent %d is dead, leaving", pid)
		stopCh <- true
	}()

	return nil
}

func Restart(postRespawn func()) error {
	if err := os.Setenv("REEXEC_PPID", fmt.Sprint(os.Getpid())); nil != err {
		return errors.WithStack(err)
	}
	if err := os.Setenv("REEXEC_SHELL_PID", fmt.Sprint(Getppid())); nil != err {
		return errors.WithStack(err)
	}
	p, err := Respawn()
	if err != nil {
		return err
	}

	if postRespawn != nil {
		postRespawn()
	}

	waitCh := make(chan *os.ProcessState, 1)
	errCh := make(chan error, 1)
	go func() {
		state, err := p.Wait()
		if err != nil {
			errCh <- err
		} else {
			waitCh <- state
		}
		close(errCh)
		close(waitCh)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)

	for {
		select {
		case sig := <-sigChan:
			if shouldSignalBeIgnored(sig) {
				continue
			}
			if err := p.Signal(sig); err != nil {
				fmt.Fprintln(os.Stderr, "error sending signal", sig, err)
			}
		case <-errCh:
			os.Exit(1)
		case state := <-waitCh:
			status := 0
			if !state.Success() {
				exiterr := exec.ExitError{ProcessState: state}
				// This will work on Windows and Unix
				if s, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					status = s.ExitStatus()
				}
			}
			os.Exit(status)
		}
	}
}

func Respawn() (*os.Process, error) {
	argv0, err := console.CurrentBinaryPath()
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	files := []*os.File{
		os.Stdin,
		os.Stdout,
		os.Stderr,
	}

	sys := &syscall.SysProcAttr{}
	setsid(sys)
	p, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: files,
		Sys:   sys,
	})
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "error starting the process"))
	}
	return p, nil
}

func IsChild() bool {
	return os.Getenv("REEXEC_WATCH_PID") != ""
}
