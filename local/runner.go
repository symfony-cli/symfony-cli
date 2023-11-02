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
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/inotify"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/reexec"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

type runnerMode int

const (
	RunnerModeOnce         runnerMode = iota // run once
	RunnerModeLoopAttached                   // run in the foreground and restart automatically in case of an error except if the first run failed
	RunnerModeLoopDetached                   // run as a daemon (run in the background, restarted automatically in case of an error except if the first run failed)
)

const RunnerReliefDuration = 2 * time.Second

type RunnerWentToBackground struct{}

func (RunnerWentToBackground) Error() string { return "" }

type Runner struct {
	binary  string
	mode    runnerMode
	pidFile *pid.PidFile

	BuildCmdHook        func(*exec.Cmd) error
	AlwaysRestartOnExit bool
}

func NewRunner(pidFile *pid.PidFile, mode runnerMode) (*Runner, error) {
	var err error
	r := &Runner{
		mode:    mode,
		pidFile: pidFile,
	}
	r.binary, err = exec.LookPath(pidFile.Binary())
	if err != nil {
		r.pidFile.Remove()
		return nil, errors.WithStack(err)
	}

	return r, nil
}

func (r *Runner) Run() error {
	if r.mode == RunnerModeLoopDetached {
		if !reexec.IsChild() {
			varDir := filepath.Join(util.GetHomeDir(), "var")
			if err := os.MkdirAll(varDir, 0755); err != nil {
				return errors.Wrap(err, "Could not create status file")
			}
			err := reexec.Background(varDir)
			if err == nil {
				return RunnerWentToBackground{}
			}

			if _, isExitCoder := err.(console.ExitCoder); isExitCoder {
				return err
			}
			terminal.Printfln("Impossible to go to the background: %s", err)
			terminal.Println("Continue in foreground")
			r.mode = RunnerModeOnce
		} else {
			if err := reexec.NotifyForeground("boot"); err != nil {
				return console.Exit(fmt.Sprintf("Unable to go to the background: %s, aborting", err), 1)
			}
		}
	}

	// We want those NOT to be buffered on purpose to be able to skip events when restarting
	cmdExitChan := make(chan error) // receives command exit status, allow to cmd.Wait() in non-blocking way
	restartChan := make(chan bool)  // receives requests to restart the command
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if len(r.pidFile.Watched) > 0 {
		// Make the channel buffered to ensure no event is dropped. Notify will drop
		// an event if the receiver is not able to keep up the sending pace.
		c := make(chan inotify.EventInfo, 10)
		defer inotify.Stop(c)

		go func() {
			for {
				event := <-c

				// ignore vim temporary files events
				if strings.HasSuffix(filepath.Ext(event.Path()), "~") {
					continue
				}

				terminal.Logger.Debug().Msg("Got event: " + event.Event().String())

				select {
				case restartChan <- true:
				default:
				}
			}
		}()

		for _, watched := range r.pidFile.Watched {
			if fi, err := os.Stat(watched); err != nil {
				continue
			} else if fi.IsDir() {
				terminal.Logger.Info().Msg("Watching directory " + watched)
				watched = filepath.Join(watched, "...")
			} else {
				terminal.Logger.Info().Msg("Watching file " + watched)
			}
			if err := inotify.Watch(watched, c, inotify.All); err != nil {
				return errors.Wrapf(err, `could not watch "%s"`, watched)
			}
		}
	}

	firstBoot := r.mode != RunnerModeOnce
	looping := r.mode != RunnerModeOnce || len(r.pidFile.Watched) > 0

	// duration is not really important here, we just need enough time to stop
	// the timer to be sure no event is fired and got stocked in the channel
	timer := time.NewTimer(time.Hour)
	timer.Stop()

	pid := os.Getpid()

	for {
		cmd, err := r.buildCmd()
		if err != nil {
			return errors.Wrap(err, "unable to build cmd")
		}

		if err := cmd.Start(); err != nil {
			return errors.Wrapf(err, `command "%s" failed to start`, r.pidFile)
		}

		go func() { cmdExitChan <- cmd.Wait() }()

		if firstBoot {
			timer.Reset(RunnerReliefDuration)

			if r.mode == RunnerModeLoopDetached {
				reexec.NotifyForeground("started")
			}

			select {
			case err := <-cmdExitChan:
				if err != nil {
					return errors.Wrapf(err, `command "%s" failed early`, r.pidFile)
				}

				timer.Stop()
				// when the command is really fast to run, it will be already
				// done here, so we need to forward exit status as if it has
				// finished later one
				go func() { cmdExitChan <- err }()
			case <-timer.C:
			}
		}

		if r.mode == RunnerModeLoopAttached {
			pid = cmd.Process.Pid
		}
		if firstBoot || r.mode == RunnerModeLoopAttached {
			if err := r.pidFile.Write(pid, 0, ""); err != nil {
				return errors.Wrap(err, "unable to write pid file")
			}
		}
		if firstBoot && r.mode == RunnerModeLoopDetached {
			terminal.RemapOutput(cmd.Stdout, cmd.Stderr).SetDecorated(true)
			reexec.NotifyForeground(reexec.UP)
		}

		firstBoot = false

		select {
		case sig := <-sigChan:
			terminal.Logger.Info().Msgf("Signal \"%s\" received, forwarding to command and exiting\n", sig)
			err := cmd.Process.Signal(sig)
			if err != nil && runtime.GOOS == "windows" && strings.Contains(err.Error(), "not supported by windows") {
				return exec.Command("CMD", "/C", "TASKKILL", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
			}
			return err
		case <-restartChan:
			// We use SIGTERM here because it's nicer and thus when we use our
			// wrappers, signal will be nicely forwarded
			cmd.Process.Signal(syscall.SIGTERM)
			// we need to drain cmdExit channel to unblock cmd channel receiver
			<-cmdExitChan
		// Command exited
		case err := <-cmdExitChan:
			err = errors.Wrapf(err, `command "%s" failed`, r.pidFile)

			// Command is NOT set up to loop, stop here and remove the pidFile
			// if the command is successful
			if !looping {
				if err != nil {
					return err
				}

				return r.pidFile.Remove()
			}

			// Command is set up to restart on exit (usually PHP builtin
			// server), so we restart immediately without waiting
			if r.AlwaysRestartOnExit {
				terminal.Logger.Error().Msgf(`command "%s" exited, restarting it immediately`, r.pidFile)
				continue
			}

			// In case of error we want to wait up-to 5 seconds before
			// restarting the command, this avoids overloading the system with a
			// failing command
			if err != nil {
				terminal.Logger.Error().Msgf("%s, waiting 5 seconds before restarting it", err)
				timer.Reset(5 * time.Second)
			}

			// Wait for a timer to expire or a file to be changed to restart
			// or a signal to be received to exit
			select {
			case sig := <-sigChan:
				terminal.Logger.Info().Msgf(`Signal "%s" received, exiting`, sig)
				return nil
			case <-restartChan:
				timer.Stop()
			case <-timer.C:
			}
		}

		terminal.Logger.Info().Msgf(`Restarting command "%s"`, r.pidFile)
	}
}

func (r *Runner) buildCmd() (*exec.Cmd, error) {
	cmd := exec.Command(r.binary, r.pidFile.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Dir = r.pidFile.Dir

	if r.mode == RunnerModeOnce {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	} else if logWriter, err := r.pidFile.LogWriter(); err != nil {
		return nil, errors.WithStack(err)
	} else {
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}

	if r.BuildCmdHook != nil {
		if err := r.BuildCmdHook(cmd); err != nil {
			return cmd, errors.WithStack(err)
		}
	}

	return cmd, nil
}
