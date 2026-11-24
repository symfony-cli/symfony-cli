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
	"time"

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

	lines chan *namedLine
	seen  sync.Map
	tails sync.WaitGroup
}

func (tailer *Tailer) Watch(pidFile *pid.PidFile) error {
	tailer.lines = make(chan *namedLine, 100)

	// Web server/PHP log file
	if !tailer.NoServerLogs {
		tailer.tailPidFile(pidFile)
	}

	// Worker log files
	if !tailer.NoWorkerLogs && tailer.Follow {
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
				if _, ok := tailer.seen.Load(e.Path()); ok {
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
				tailer.tailPidFile(p)
			}
		}()
	}
	if !tailer.NoWorkerLogs {
		for _, p := range pid.AllWorkers(pidFile.Dir) {
			tailer.tailPidFile(p)
		}
	}

	// Application log files (Symfony for now)
	if !tailer.NoAppLogs {
		for _, applog := range tailer.AppLogs {
			if err := tailer.watchAppLogFile(applog); err != nil {
				return err
			}
		}
		if len(tailer.AppLogs) == 0 {
			for _, dir := range findApplicationLogDirs(pidFile.Dir) {
				if err := tailer.watchAppLogDir(dir); err != nil {
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

	if !tailer.Follow {
		go func() {
			tailer.tails.Wait()
			close(tailer.lines)
		}()
	}

	var buf bytes.Buffer
	for line := range tailer.lines {
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
	return nil
}

func (tailer *Tailer) WatchAdditionalPidFile(file *pid.PidFile) {
	tailer.tailPidFile(file)
}

func (tailer *Tailer) tailPidFile(p *pid.PidFile) {
	if _, loaded := tailer.seen.LoadOrStore(p.PidFile(), true); loaded {
		return
	}
	tailer.tails.Add(1)
	go func() {
		defer tailer.tails.Done()
		tailLogFile(p, tailer.lines, tailer.Follow, tailer.LinesNb)
	}()
}

func tailLogFile(p *pid.PidFile, lines chan *namedLine, follow bool, nblines int64) {
	if follow {
		if err := p.WaitForLogs(); err != nil {
			terminal.Printfln("<warning>WARNING</> %s log file cannot be tailed: %s", p.String(), err)
			return
		}
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
		ReOpen:    follow,
		Follow:    follow,
		MustExist: !follow,
		Poll:      true,
		Logger:    tail.DiscardingLogger,
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
func (tailer *Tailer) watchAppLogFile(applog string) error {
	applog, err := filepath.Abs(applog)
	if err != nil {
		return errors.Wrapf(err, "unable to get absolute path for %s", applog)
	}
	if !tailer.Follow {
		tailer.tailAppLog(applog, replayLastLines)
		return nil
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

	if err := tailer.watchAppLogs(dir, func(path string) bool { return path == applog }); err != nil {
		return err
	}
	tailer.tailAppLog(applog, replayLastLines)
	return nil
}

// watchAppLogDir tails the log files of a directory: only the most recently
// modified one replays its last lines, the other ones (like rotated files) are
// followed from their current end once written to, and the ones created later
// on are read from their first line.
func (tailer *Tailer) watchAppLogDir(dir string) error {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return errors.Wrapf(err, "unable to evaluate symlinks for %s", dir)
	}
	isLogFile := func(path string) bool {
		return filepath.Dir(path) == realDir && filepath.Ext(path) == ".log"
	}

	var mu sync.Mutex
	idleSizes := map[string]int64{}
	mu.Lock()
	defer mu.Unlock()

	if tailer.Follow {
		watcherChan := make(chan inotify.EventInfo, 100)
		if err := inotify.Watch(dir, watcherChan, inotify.Create, inotify.Write); err != nil {
			return errors.Wrap(err, "unable to watch the applog directory")
		}
		go func() {
			for e := range watcherChan {
				if !isLogFile(e.Path()) {
					continue
				}
				mu.Lock()
				offset := idleSizes[e.Path()]
				delete(idleSizes, e.Path())
				mu.Unlock()
				if fi, err := os.Stat(e.Path()); err == nil && fi.Size() < offset {
					offset = 0
				}
				tailer.tailAppLog(e.Path(), offset)
			}
		}()
	}

	entries, err := os.ReadDir(realDir)
	if err != nil {
		return errors.WithStack(err)
	}
	latest := ""
	var latestModTime time.Time
	for _, entry := range entries {
		applog := filepath.Join(realDir, entry.Name())
		if entry.IsDir() || !isLogFile(applog) {
			continue
		}
		fi, err := os.Stat(applog)
		if err != nil {
			continue
		}
		idleSizes[applog] = fi.Size()
		if latest == "" || fi.ModTime().After(latestModTime) {
			latest, latestModTime = applog, fi.ModTime()
		}
	}
	if latest != "" {
		delete(idleSizes, latest)
		tailer.tailAppLog(latest, replayLastLines)
	}
	return nil
}

func (tailer *Tailer) watchAppLogs(dir string, match func(path string) bool) error {
	watcherChan := make(chan inotify.EventInfo, 10)
	if err := inotify.Watch(dir, watcherChan, inotify.Create); err != nil {
		return errors.Wrap(err, "unable to watch the applog directory")
	}
	go func() {
		for e := range watcherChan {
			if match(e.Path()) {
				// everything in a file created after the tailer started is new
				tailer.tailAppLog(e.Path(), 0)
			}
		}
	}()
	return nil
}

// replayLastLines makes tailAppLog start with the last tailer.LinesNb lines
const replayLastLines = -1

func (tailer *Tailer) tailAppLog(applog string, offset int64) {
	if _, loaded := tailer.seen.LoadOrStore(applog, true); loaded {
		return
	}
	tailer.tails.Add(1)
	go func() {
		defer tailer.tails.Done()
		var tsf *tail.Tail
		var err error
		if offset == replayLastLines {
			tsf, err = tailFile(applog, tailer.Follow, tailer.LinesNb)
		} else {
			tsf, err = tailFileAt(applog, tailer.Follow, offset)
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
