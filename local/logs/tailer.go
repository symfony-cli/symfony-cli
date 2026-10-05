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

	"github.com/nxadm/tail"
	"github.com/pkg/errors"
	"github.com/stoicperlman/fls"
	"github.com/symfony-cli/symfony-cli/humanlog"
	"github.com/symfony-cli/symfony-cli/inotify"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/terminal"
)

type namedLine struct {
	name string
	line *tail.Line
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
			return err
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
				if fi, err := os.Stat(e.Path()); err == nil && fi.IsDir() {
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

	// Application log files (Symfony for now)
	if !tailer.NoAppLogs {
		for _, applog := range tailer.AppLogs {
			if err := tailer.watchAppLogFile(applog, &seenDirs); err != nil {
				return err
			}
		}
		if len(tailer.AppLogs) == 0 {
			for _, dir := range findApplicationLogDirs(pidFile.Dir) {
				if err := tailer.watchAppLogDir(dir, &seenDirs); err != nil {
					return err
				}
			}
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
		w.Write(buf.Bytes())
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
	return tailFileAt(filename, follow, pos)
}

func tailFileAt(filename string, follow bool, pos int64) (*tail.Tail, error) {
	return tail.TailFile(filename, tail.Config{
		Location: &tail.SeekInfo{
			Offset: pos,
			Whence: io.SeekStart,
		},
		ReOpen: follow,
		Follow: follow,
		Poll:   true,
		Logger: tail.DiscardingLogger,
	})
}

// find the application log directories (only Symfony is supported for now)
func findApplicationLogDirs(projectDir string) []string {
	dirs := []string{}
	for _, subdir := range []string{
		filepath.Join("var", "log"),
		filepath.Join("var", "logs"),
		filepath.Join("app", "logs"),
	} {
		dir := filepath.Join(projectDir, subdir)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// watchAppLogFile tails the given log file, waiting for it to be created if needed
func (tailer *Tailer) watchAppLogFile(applog string, seen *sync.Map) error {
	applog, err := filepath.Abs(applog)
	if err != nil {
		return errors.Wrapf(err, "unable to get absolute path for %s", applog)
	}
	dir := filepath.Dir(applog)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.WithStack(err)
	}
	// inotify reports paths with symlinks evaluated
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return errors.Wrapf(err, "unable to evaluate symlinks for %s", dir)
	}
	applog = filepath.Join(realDir, filepath.Base(applog))
	if realAppLog, err := filepath.EvalSymlinks(applog); err == nil {
		applog = realAppLog
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.Wrapf(err, "unable to evaluate symlinks for %s", applog)
	}

	if err := tailer.watchAppLogs(dir, func(path string) bool { return path == applog }, seen); err != nil {
		return err
	}
	tailer.tailAppLog(applog, false, seen)
	return nil
}

// watchAppLogDir tails all the log files of a directory, including the ones created later on
func (tailer *Tailer) watchAppLogDir(dir string, seen *sync.Map) error {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return errors.Wrapf(err, "unable to evaluate symlinks for %s", dir)
	}
	isLogFile := func(path string) bool {
		return filepath.Dir(path) == realDir && filepath.Ext(path) == ".log"
	}

	if err := tailer.watchAppLogs(dir, isLogFile, seen); err != nil {
		return err
	}
	entries, err := os.ReadDir(realDir)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range entries {
		if applog := filepath.Join(realDir, entry.Name()); !entry.IsDir() && isLogFile(applog) {
			tailer.tailAppLog(applog, false, seen)
		}
	}
	return nil
}

func (tailer *Tailer) watchAppLogs(dir string, match func(path string) bool, seen *sync.Map) error {
	watcherChan := make(chan inotify.EventInfo, 10)
	if err := inotify.Watch(dir, watcherChan, inotify.Create); err != nil {
		return errors.Wrap(err, "unable to watch the applog directory")
	}
	go func() {
		for e := range watcherChan {
			if match(e.Path()) {
				// everything in a file created after the tailer started is new
				tailer.tailAppLog(e.Path(), true, seen)
			}
		}
	}()
	return nil
}

func (tailer *Tailer) tailAppLog(applog string, fromStart bool, seen *sync.Map) {
	if _, loaded := seen.LoadOrStore(applog, true); loaded {
		return
	}
	go func() {
		var tsf *tail.Tail
		var err error
		if fromStart {
			tsf, err = tailFileAt(applog, tailer.Follow, 0)
		} else {
			tsf, err = tailFile(applog, tailer.Follow, tailer.LinesNb)
		}
		if err != nil {
			terminal.Printfln("<warning>WARNING</> %s log file cannot be tailed: %s", applog, err)
			return
		}
		for line := range tsf.Lines {
			tailer.lines <- &namedLine{name: "Application", line: line}
		}
	}()
}
