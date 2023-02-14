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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/stoicperlman/fls"
	"github.com/symfony-cli/symfony-cli/humanlog"
	"github.com/symfony-cli/symfony-cli/inotify"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
	realinotify "github.com/syncthing/notify"
)

type namedLine struct {
	name string
	line *tail.Line
}

type logFileEvent string

func (e logFileEvent) Event() realinotify.Event {
	return inotify.Create
}

func (e logFileEvent) Path() string {
	return string(e)
}

func (e logFileEvent) Sys() interface{} {
	return nil
}

type Tailer struct {
	Follow       bool
	LinesNb      int64
	NoHumanize   bool
	AppLogs      []string
	NoAppLogs    bool
	NoWorkerLogs bool
	NoServerLogs bool

	pidFileChan chan *pid.PidFile
	lines       chan *namedLine
}

func (tailer *Tailer) Watch(pidFile *pid.PidFile) error {
	// This has to be the very first things to be sure to have the chans
	// initialized soon enough
	tailer.pidFileChan = make(chan *pid.PidFile)
	tailer.lines = make(chan *namedLine, 100)

	seenDirs := sync.Map{}
	go func() {
		for {
			pidFile := <-tailer.pidFileChan
			if _, ok := seenDirs.Load(pidFile.PidFile()); ok {
				continue
			}

			seenDirs.Store(pidFile.PidFile(), true)
			go tailLogFile(pidFile, tailer.lines, tailer.Follow, tailer.LinesNb)
		}
	}()

	// Web server/PHP log file
	if !tailer.NoServerLogs {
		tailer.pidFileChan <- pidFile
	}

	// Worker log files
	if !tailer.NoWorkerLogs {
		workerDir := pidFile.WorkerPidDir()
		if err := os.MkdirAll(workerDir, 0755); err != nil {
			return errors.WithStack(err)
		}
		watcherChan := make(chan inotify.EventInfo, 1)
		if err := inotify.Watch(workerDir, watcherChan, inotify.Create); err != nil {
			return errors.Wrap(err, "unable to watch the worker pid directory")
		}
		go func() {
			for {
				e := <-watcherChan
				if _, ok := seenDirs.Load(e.Path()); ok {
					continue
				}
				p, err := pid.Load(e.Path())
				if err != nil {
					terminal.Printfln("<warning>WARNING</> %s", err)
					continue
				}
				tailer.pidFileChan <- p
			}
		}()
		for _, p := range pid.AllWorkers(pidFile.Dir) {
			tailer.pidFileChan <- p
		}
	}

	// Application log file (Symfony for now)
	if !tailer.NoAppLogs {
		applogs := tailer.AppLogs
		if len(applogs) == 0 {
			applogs = findApplicationLogFiles(pidFile.Dir)
		}

		for _, applog := range applogs {
			watcherChan := make(chan inotify.EventInfo, 1)

			// Convert relative paths to absolute paths
			absAppLog, err := filepath.Abs(applog)
			if err != nil {
				return errors.Wrapf(err, "unable to get absolute path for %s", applog)
			}
			applog = absAppLog

			dir := filepath.Dir(applog)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return errors.WithStack(err)
			}
			if err := inotify.Watch(dir, watcherChan, inotify.Create); err != nil {
				return errors.Wrap(err, "unable to watch the applog directory")
			}

			// Evaluate possible symlinks in the applog path, this is needed because
			// inotify will notify us on source path, and not the symlink path.
			if _, err := os.Stat(applog); err == nil {
				realAppLog, err := filepath.EvalSymlinks(applog)
				if err != nil {
					return errors.Wrapf(err, "unable to evaluate symlinks for %s", applog)
				}
				applog = realAppLog
			} else if errors.Is(err, os.ErrNotExist) {
				realDir, err := filepath.EvalSymlinks(dir)
				if err != nil {
					return errors.Wrapf(err, "unable to evaluate symlinks for %s", dir)
				}
				applog = filepath.Join(realDir, filepath.Base(applog))
			} else {
				return errors.Wrapf(err, "unable to evaluate symlinks for %s", applog)
			}

			go func(applog string) {
				for {
					e := <-watcherChan
					if e.Path() != applog {
						continue
					}
					if _, ok := seenDirs.Load(applog); ok {
						continue
					}
					seenDirs.Store(applog, true)
					go func() {
						tsf, err := tailFile(applog, tailer.Follow, tailer.LinesNb)
						if err != nil {
							terminal.Printfln("<warning>WARNING</> %s log file cannot be tailed: %s", applog, err)
							return
						}
						for line := range tsf.Lines {
							tailer.lines <- &namedLine{name: "Application", line: line}
						}
					}()
				}
			}(applog)
			watcherChan <- logFileEvent(applog)
		}
	}

	return nil
}

func (tailer *Tailer) Tail(w io.Writer) error {
	var humanizer *humanlog.Handler
	if !tailer.NoHumanize {
		humanizer = humanlog.NewHandler(&humanlog.Options{
			SkipUnchanged: true,
			WithSource:    true,
		})
	}

	var buf bytes.Buffer
	for {
		line := <-tailer.lines
		if line == nil {
			continue
		}
		buf.Reset()
		fmt.Fprintf(&buf, "[<info>%-11s</>] ", line.name)
		content := strings.TrimRight(line.line.Text, "\n")
		if humanizer == nil {
			fmt.Fprintln(&buf, content)
		} else {
			buf.Write(humanizer.Prettify([]byte(content)))
			buf.Write([]byte("\n"))
		}
		_, _ = w.Write(buf.Bytes())
	}
}

func (tailer Tailer) WatchAdditionalPidFile(file *pid.PidFile) {
	tailer.pidFileChan <- file
}

func tailLogFile(p *pid.PidFile, lines chan *namedLine, follow bool, nblines int64) {
	if err := p.WaitForLogs(); err != nil {
		terminal.Printfln("<warning>WARNING</> %s log file cannot be tailed: %s", p.String(), err)
		return
	}
	t, err := tailFile(p.LogFile(), follow, nblines)
	if err != nil {
		terminal.Printfln("<warning>WARNING</> %s log file cannot be tailed: %s", p.String(), err)
		return
	}
	terminal.Printfln("Following <info>%s</info> log file (%s)", p.String(), p.LogFile())
	for line := range t.Lines {
		lines <- &namedLine{name: p.ShortName(), line: line}
	}
}

func tailFile(filename string, follow bool, nblines int64) (*tail.Tail, error) {
	var pos int64
	f, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err == nil {
		pos, _ = fls.LineFile(f).SeekLine(-nblines, io.SeekEnd)
	}
	f.Close()
	t, e := tail.TailFile(filename, tail.Config{
		Location: &tail.SeekInfo{
			Offset: pos,
			Whence: io.SeekStart,
		},
		ReOpen: follow,
		Follow: follow,
		Poll:   true,
		Logger: tail.DiscardingLogger,
	})

	return t, errors.WithStack(e)
}

// find the application log file(s) (only Symfony is supported for now)
func findApplicationLogFiles(projectDir string) []string {
	subdirs := []string{
		filepath.Join("var", "log"),
		filepath.Join("var", "logs"),
		filepath.Join("app", "logs"),
	}
	// FIXME
	env := "dev"
	files := []string{}
	for _, subdir := range subdirs {
		applog := filepath.Join(projectDir, subdir, env+".log")
		if _, err := os.Stat(applog); err != nil {
			continue
		}
		files = append(files, applog)
	}
	return files
}
