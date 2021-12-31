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
		watcherChan := make(chan inotify.EventInfo, 1)

		for _, applog := range applogs {
			dir := filepath.Dir(applog)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := inotify.Watch(dir, watcherChan, inotify.Create); err != nil {
				return errors.Wrap(err, "unable to watch the applog directory")
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
	terminal.Printfln("Tailing <info>%s</info> log file (%s)", p.String(), p.LogFile())
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
	return tail.TailFile(filename, tail.Config{
		Location: &tail.SeekInfo{
			Offset: pos,
			Whence: os.SEEK_SET,
		},
		ReOpen: follow,
		Follow: follow,
		Poll:   true,
		Logger: tail.DiscardingLogger,
	})
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
