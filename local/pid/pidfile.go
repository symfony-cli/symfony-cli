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
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/inotify"
	"github.com/symfony-cli/symfony-cli/local/projects"
	"github.com/symfony-cli/symfony-cli/util"
)

const WebServerName = "Web Server"

type PidFile struct {
	Dir        string   `json:"dir"`
	Watched    []string `json:"watch"`
	Pid        int      `json:"pid"`
	Port       int      `json:"port"`
	Scheme     string   `json:"scheme"`
	Args       []string `json:"args"`
	CustomName string   `json:"name"`

	path string

	lwInit sync.Once      // used to ensure that the log writer is only created once
	lw     io.WriteCloser // log writer, used to write logs to the log file
}

func New(dir string, args []string) *PidFile {
	var path string
	command := strings.Join(args, " ")
	if args == nil {
		// server or proxy
		path = filepath.Join(util.GetHomeDir(), "var", name(dir)+".pid")
	} else {
		// workers are stored in a subdirectory
		path = filepath.Join(util.GetHomeDir(), "var", name(dir), name(command)+".pid")
	}
	// we need to load the existing file if there is one
	p, err := Load(path)
	if err != nil {
		p = &PidFile{
			Dir:  dir,
			Args: args,
			path: path,
		}
	}
	return p
}

func Load(path string) (*PidFile, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p *PidFile
	if err := json.Unmarshal(contents, &p); err != nil {
		return nil, err
	}
	p.path = path
	return p, nil
}

func (p *PidFile) Command() string {
	return strings.Join(p.Args, " ")
}

func (p *PidFile) String() string {
	if p.CustomName != "" {
		return p.CustomName
	}
	if p.Args == nil {
		return WebServerName
	}
	return p.Command()
}

func (p *PidFile) ShortName() string {
	if p.CustomName != "" {
		return p.CustomName
	}
	if len(p.Args) == 0 {
		return WebServerName
	}
	return "Worker " + p.Args[0]
}

func (p *PidFile) WaitForExit() error {
	if p.Pid == 0 {
		return nil
	}

	process, err := os.FindProcess(p.Pid)
	if err != nil {
		return err
	}

	defer p.Remove()
	ch := make(chan error)
	go func() {
		if process.Signal(syscall.Signal(0)) != nil {
			ch <- nil
			return
		}

		_, err := process.Wait()
		if err == nil {
			ch <- nil
			return
		}
		if serr, isSysCallError := err.(*os.SyscallError); isSysCallError {
			if errn, isErrno := serr.Err.(syscall.Errno); isErrno && errn == syscall.ECHILD {
				ch <- nil
				return
			}
		}
		ch <- errors.WithMessagef(err, "while waiting for process %v (%s)", p.Pid, p.ShortName())
		close(ch)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(30 * time.Second):
		return errors.Errorf("Time out detected during \"%s\" process exit", p.ShortName())
	}
}

func (p *PidFile) WaitForPid() <-chan error {
	ch := make(chan error, 1)

	watcherChan := make(chan inotify.EventInfo, 1)

	// First ensure the directory exists to be able to watch creation inside
	if err := os.MkdirAll(filepath.Dir(p.path), 0755); err != nil && !os.IsExist(err) {
		ch <- err
		return ch
	}

	if err := inotify.Watch(filepath.Dir(p.path), watcherChan, inotify.Create); err != nil {
		ch <- err
		return ch
	}

	_, err := os.Stat(p.PidFile())
	if err == nil {
		ch <- nil
		inotify.Stop(watcherChan)
		return ch
	}

	go func() {
		defer inotify.Stop(watcherChan)

		for {
			e := <-watcherChan
			if e.Path() == p.PidFile() {
				ch <- nil
				return
			}
		}
	}()

	return ch
}

func (p *PidFile) WaitForLogs() error {
	watcherChan := make(chan inotify.EventInfo, 1)
	defer inotify.Stop(watcherChan)
	logFile := p.LogFile()
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return err
	}
	if err := inotify.Watch(filepath.Dir(logFile), watcherChan, inotify.Create); err != nil {
		return errors.Wrap(err, "unable to watch log file")
	}
	if _, err := os.Stat(logFile); err == nil {
		return nil
	}
	for {
		e := <-watcherChan
		if e.Path() == logFile {
			return nil
		}
	}
}

func (p *PidFile) LogFile() string {
	if p.Args == nil {
		return filepath.Join(util.GetHomeDir(), "log", name(p.Dir)+".log")
	}
	if p.CustomName != "" {
		return filepath.Join(p.WorkerLogDir(), name(p.CustomName)+".log")
	}
	return filepath.Join(p.WorkerLogDir(), name(p.Command())+".log")
}

func (p *PidFile) PidFile() string {
	return p.path
}

func (p *PidFile) WorkerLogDir() string {
	return filepath.Join(util.GetHomeDir(), "log", name(p.Dir))
}

func (p *PidFile) WorkerPidDir() string {
	return filepath.Join(util.GetHomeDir(), "var", name(p.Dir))
}

func (p *PidFile) TempDirectory() string {
	return filepath.Join(util.GetHomeDir(), "php", name(p.Dir))
}

func (p *PidFile) CleanupDirectories() {
	os.RemoveAll(p.TempDirectory())
	// We don't want to force removal of log and pid files, we only want to
	// clean up empty directories. To do so we use `os.Remove` instead of
	// `os.RemoveAll`
	os.Remove(p.WorkerLogDir())
	os.Remove(p.WorkerPidDir())
}

func (p *PidFile) LogReader() (io.ReadCloser, error) {
	logFile := p.LogFile()
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, err
	}
	r, err := os.OpenFile(logFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// LogWriter returns a writer to write logs to the log file. It creates the log
// file if it does not exist, and truncates it if it does. It is safe to call
// this method multiple times, it will only create the log file once per process
// lifetime: it is useful to have a single truncation (and thus a clean log
// file) at the beginning of the process management but not to truncate the log
// file when the process is restarted.
// Please note this method might not return a writer even if the error is nil
// (the error is returned only for the first call).
func (p *PidFile) LogWriter() (io.WriteCloser, error) {
	var err error

	p.lwInit.Do(func() {
		logFile := p.LogFile()
		if err = errors.WithStack(os.MkdirAll(filepath.Dir(logFile), 0755)); err != nil {
			return
		}
		p.lw, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
	})

	if err != nil {
		return nil, err
	}

	return p.lw, err
}

func (p *PidFile) Binary() string {
	if len(p.Args) == 0 {
		return ""
	}
	return p.Args[0]
}

func AllWorkers(dir string) []*PidFile {
	return doAll(filepath.Join(util.GetHomeDir(), "var", name(dir)))
}

// Remove a pidfile
func (p *PidFile) Remove() error {
	for _, file := range []string{p.LogFile(), p.PidFile()} {
		// Ensure log writer is closed first to avoid "file is being used" on Windows
		if p.lw != nil {
			_ = p.lw.Close()
			p.lw = nil
		}
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return errors.WithStack(err)
		}
		// DO NOT remove empty dirs (as it makes inotify fail)
	}
	return nil
}

// Write writes a pidfile
func (p *PidFile) Write(pid, port int, scheme string) error {
	oldPid, err := Load(p.PidFile())
	if err == nil && oldPid.IsRunning() {
		return errors.Errorf("Process is already running under PID %d", oldPid.Pid)
	}

	p.Pid = pid
	p.Port = port
	p.Scheme = scheme

	if err := os.MkdirAll(filepath.Dir(p.path), 0755); err != nil && !os.IsExist(err) {
		return err
	}

	b, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(p.path, b, 0644)
}

// Stop kills the current process
func (p *PidFile) Stop() error {
	if p.Pid == 0 {
		return nil
	}
	defer p.Remove()
	return kill(p.Pid)
}

// Signal sends a signal to the current process for this PidFile
func (p *PidFile) Signal(sig os.Signal) error {
	if p.Pid == 0 {
		return nil
	}

	process, err := os.FindProcess(p.Pid)
	if err != nil {
		return err
	}

	return process.Signal(sig)
}

func ToConfiguredProjects(shortenHomeDir bool) (map[string]*projects.ConfiguredProject, error) {
	ps := make(map[string]*projects.ConfiguredProject)
	userHomeDir, err := homedir.Dir()
	if err != nil {
		userHomeDir = ""
	}
	for _, pid := range doAll(filepath.Join(util.GetHomeDir(), "var")) {
		if !pid.IsRunning() {
			continue
		}
		port := pid.Port
		shortDir := pid.Dir
		if strings.HasPrefix(shortDir, userHomeDir) && shortenHomeDir {
			shortDir = "~" + shortDir[len(userHomeDir):]
		}
		ps[shortDir] = &projects.ConfiguredProject{
			Port:   port,
			Scheme: pid.Scheme,
		}
	}
	return ps, nil
}

// IsRunning returns true if the process is currently running
func (p *PidFile) IsRunning() bool {
	if p.Pid == 0 {
		return false
	}
	process, err := os.FindProcess(p.Pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	if err != nil && err.Error() == "no such process" {
		return false
	}
	if err != nil && err.Error() == "os: process already finished" {
		return false
	}
	return true
}

func (p *PidFile) Name() string {
	return name(p.Dir)
}

func name(dir string) string {
	h := sha1.New()
	io.WriteString(h, dir)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func doAll(dir string) []*PidFile {
	pidFiles := []*PidFile{}
	filepath.Walk(dir, func(p string, f os.FileInfo, err error) error {
		if err != nil {
			// prevent panic by handling failure accessing a path
			return nil
		}
		// one level of depth only
		if dir != p && f.IsDir() {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(p, ".pid") {
			return nil
		}
		contents, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		var pidFile *PidFile
		if err := json.Unmarshal(contents, &pidFile); err != nil {
			return nil
		}
		if strings.Contains(pidFile.Dir, "__proxy__") {
			return nil
		}
		pidFile.path = p
		if !pidFile.IsRunning() {
			pidFile.Remove()
			return nil
		}
		pidFiles = append(pidFiles, pidFile)
		return nil
	})
	return pidFiles
}
