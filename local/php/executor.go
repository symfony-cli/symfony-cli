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

package php

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

type Executor struct {
	Dir        string
	BinName    string
	Args       []string
	SkipNbArgs int
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	Paths      []string
	ExtraEnv   []string
	Logger     zerolog.Logger

	environ   []string
	iniDir    string
	scriptDir string
	tempDir   string
}

var execCommand = exec.Command

// IsBinaryName returns true if the command is a PHP binary name
func IsBinaryName(name string) bool {
	for _, bin := range GetBinaryNames() {
		if name == bin {
			return true
		}
	}
	return false
}

func GetBinaryNames() []string {
	return []string{"php", "pecl", "pear", "php-fpm", "php-cgi", "php-config", "phpdbg", "phpize"}
}

func (e *Executor) lookupPHP(cliDir string, forceReload bool) (*phpstore.Version, string, bool, error) {
	phpStore := phpstore.New(cliDir, forceReload, nil)
	v, source, warning, err := phpStore.BestVersionForDir(e.scriptDir)
	if warning != "" {
		terminal.Eprintfln("<warning>WARNING</> %s", warning)
	}
	if err != nil {
		return nil, "", true, err
	}
	e.Logger.Debug().Str("source", "PHP").Msgf("Using PHP version %s (from %s)", v.Version, source)
	path := v.PHPPath
	phpiniArgs := true
	if e.BinName == "php-fpm" {
		if v.FPMPath == "" {
			return nil, "", true, errors.Errorf("%s does not seem to be available under %s\n", e.BinName, filepath.Dir(path))
		}
		path = v.FPMPath
	}
	if e.BinName == "php-cgi" {
		if v.CGIPath == "" {
			return nil, "", true, errors.Errorf("%s does not seem to be available under %s\n", e.BinName, filepath.Dir(path))
		}
		path = v.CGIPath
	}
	if e.BinName == "php-config" {
		if v.PHPConfigPath == "" {
			return nil, "", true, errors.Errorf("%s does not seem to be available under %s\n", e.BinName, filepath.Dir(path))
		}
		phpiniArgs = false
		path = v.PHPConfigPath
	}
	if e.BinName == "phpize" {
		if v.PHPizePath == "" {
			return nil, "", true, errors.Errorf("%s does not seem to be available under %s\n", e.BinName, filepath.Dir(path))
		}
		phpiniArgs = false
		path = v.PHPizePath
	}
	if e.BinName == "phpdbg" {
		if v.PHPdbgPath == "" {
			return nil, "", true, errors.Errorf("%s does not seem to be available under %s\n", e.BinName, filepath.Dir(path))
		}
		phpiniArgs = false
		path = v.PHPdbgPath
	}
	if e.BinName == "pecl" || e.BinName == "pear" {
		phpiniArgs = false
		path = filepath.Join(filepath.Dir(path), e.BinName)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// if a version does not exist anymore, it probably means that PHP has been updated
		// try again after forcing the reload of the versions
		if !forceReload {
			return e.lookupPHP(cliDir, true)
		}

		// we should never get here
		return nil, "", true, errors.Errorf("%s does not seem to be available anymore under %s\n", e.BinName, filepath.Dir(path))
	}

	return v, path, phpiniArgs, nil
}

// DetectScriptDir detects the script dir based on the current configuration
func (e *Executor) DetectScriptDir() (string, error) {
	if e.scriptDir != "" {
		return e.scriptDir, nil
	}

	if e.SkipNbArgs == 0 {
		e.SkipNbArgs = 1
	}

	if e.SkipNbArgs < 0 {
		wd, err := os.Getwd()
		if err != nil {
			return "", errors.WithStack(err)
		}
		e.scriptDir = wd
	} else {
		if len(e.Args) < 1 {
			return "", errors.New("args cannot be empty")
		}

		e.scriptDir = detectScriptDir(e.Args[e.SkipNbArgs:])
	}

	return e.scriptDir, nil
}

// Config determines the right version of PHP depending on the configuration
// (+ its configuration). It also creates some symlinks to ease the integration
// with underlying tools that could try to run PHP. This is the responsibility
// of the caller to clean those temporary files. One can call
// CleanupTemporaryDirectories to do so.
func (e *Executor) Config(loadDotEnv bool) error {
	// reset environment
	e.environ = make([]string, 0)

	if len(e.Args) < 1 {
		return errors.New("args cannot be empty")
	}

	if _, err := e.DetectScriptDir(); err != nil {
		return err
	}

	vars := make(map[string]string)
	// env defined by Platform.sh services/tunnels or docker-compose services
	if env, err := envs.GetEnv(e.scriptDir, terminal.IsDebug()); err == nil {
		for k, v := range envs.AsMap(env) {
			vars[k] = v
		}
	}
	if loadDotEnv {
		for k, v := range envs.LoadDotEnv(vars, e.scriptDir) {
			vars[k] = v
		}
	}
	for k, v := range vars {
		e.environ = append(e.environ, fmt.Sprintf("%s=%s", k, v))
	}

	// When running in Cloud we don't need to detect PHP or do anything fancy
	// with the configuration, the only thing we want is to potentially load the
	// .env file
	if util.InCloud() {
		// args[0] MUST be the same as path
		// but as we change the path, we should update args[0] accordingly
		e.Args[0] = e.BinName
		return nil
	}

	cliDir := util.GetHomeDir()
	var v *phpstore.Version
	var path string
	var phpiniArgs bool
	var err error
	if v, path, phpiniArgs, err = e.lookupPHP(cliDir, false); err != nil {
		// try again after reloading PHP versions
		if v, path, phpiniArgs, err = e.lookupPHP(cliDir, true); err != nil {
			return err
		}
	}
	e.environ = append(e.environ, fmt.Sprintf("PHP_BINARY=%s", v.PHPPath))
	e.environ = append(e.environ, fmt.Sprintf("PHP_PATH=%s", v.PHPPath))
	// for pecl
	e.environ = append(e.environ, fmt.Sprintf("PHP_PEAR_PHP_BIN=%s", v.PHPPath))
	// prepending the PHP directory in the PATH does not work well if the PHP binary is not named "php" (like php7.3 for instance)
	// in that case, we create a temp directory with a symlink
	// we also link php-config for pecl to pick up the right one (it is always looks for something called php-config)
	if e.tempDir == "" {
		e.tempDir = filepath.Join(cliDir, "tmp", xid.New().String())
	}
	phpDir := filepath.Join(e.tempDir, "bin")
	if err := os.MkdirAll(phpDir, 0755); err != nil {
		return err
	}
	// always symlink (copy on Windows) these binaries as they can be called internally (like pecl for instance)
	if v.PHPConfigPath != "" {
		if err := symlink(v.PHPConfigPath, filepath.Join(phpDir, "php-config")); err != nil {
			return err
		}
		// we also alias a version with the prefix/suffix as required by pecl
		if filepath.Base(v.PHPConfigPath) != "php-config" {
			if err := symlink(v.PHPConfigPath, filepath.Join(phpDir, filepath.Base(v.PHPConfigPath))); err != nil {
				return err
			}
		}
	}
	if v.PHPizePath != "" {
		if err := symlink(v.PHPizePath, filepath.Join(phpDir, "phpize")); err != nil {
			return err
		}
		// we also alias a version with the prefix/suffix as required by pecl
		if filepath.Base(v.PHPizePath) != "phpize" {
			if err := symlink(v.PHPizePath, filepath.Join(phpDir, filepath.Base(v.PHPizePath))); err != nil {
				return err
			}
		}
	}
	if v.PHPdbgPath != "" {
		if err := symlink(v.PHPdbgPath, filepath.Join(phpDir, "phpdbg")); err != nil {
			return err
		}
	}
	// if the bin is not one of the previous created symlink, create the symlink now
	if _, err := os.Stat(filepath.Join(phpDir, e.BinName)); os.IsNotExist(err) {
		if err := symlink(path, filepath.Join(phpDir, e.BinName)); err != nil {
			return err
		}
	}
	e.Paths = append([]string{filepath.Dir(path), phpDir}, e.Paths...)
	if phpiniArgs {
		// see https://php.net/manual/en/configuration.file.php
		// if PHP_INI_SCAN_DIR exists, just append our new directory
		// if not, add the default one (empty string) and then our new directory
		// Look for php.ini in the script dir and go up if needed (symfony php ./app/test.php should read php/ini in ./)
		dirs := ""
		if phpIniDir := e.phpiniDirForDir(); phpIniDir != "" {
			dirs += string(os.PathListSeparator) + phpIniDir
		}
		if e.iniDir != "" {
			dirs += string(os.PathListSeparator) + e.iniDir
		}
		if dirs != "" {
			e.environ = append(e.environ, fmt.Sprintf("PHP_INI_SCAN_DIR=%s%s", os.Getenv("PHP_INI_SCAN_DIR"), dirs))
		}
	}

	// args[0] MUST be the same as path
	// but as we change the path, we should update args[0] accordingly
	e.Args[0] = path

	return err
}

func (e *Executor) CleanupTemporaryDirectories() {
	backgroundCleanup := make(chan bool, 1)
	go cleanupStaleTemporaryDirectories(e.Logger, backgroundCleanup)

	if e.iniDir != "" {
		os.RemoveAll(e.iniDir)
	}
	if e.tempDir != "" {
		os.RemoveAll(e.tempDir)
	}

	// give some room to the background clean up job to do its work
	select {
	case <-backgroundCleanup:
	case <-time.After(100 * time.Millisecond):
		e.Logger.Debug().Msg("Allocated time for temporary directories to be cleaned up is over, it will resume later on")
	}
}

// The Symfony CLI used to leak temporary directories until v5.10.8. The bug is
// fixed but because directories names are random they are not going to be
// reused and thus are not going to be cleaned up. And because they might be
// in-use by running servers we can't simply delete the parent directory. This
// is why we make our best to find the oldest directories and remove then,
// cleaning the directory little by little.
func cleanupStaleTemporaryDirectories(mainLogger zerolog.Logger, doneCh chan<- bool) {
	defer func() {
		doneCh <- true
	}()
	parentDirectory := filepath.Join(util.GetHomeDir(), "tmp")
	mainLogger = mainLogger.With().Str("dir", parentDirectory).Logger()

	if len(parentDirectory) < 6 {
		mainLogger.Warn().Msg("temporary dir path looks too short")
		return
	}

	mainLogger.Debug().Msg("Starting temporary directory cleanup...")
	dir, err := os.Open(parentDirectory)
	if err != nil {
		mainLogger.Warn().Err(err).Msg("Failed to open temporary directory")
		return
	}
	defer dir.Close()

	// the duration after which we consider temporary directories as
	// stale and can be removed
	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	for {
		// we might have a lof of entries so we need to work in batches
		entries, err := dir.Readdirnames(30)
		if err == io.EOF {
			mainLogger.Debug().Msg("Cleaning is done...")
			return
		}
		if err != nil {
			mainLogger.Warn().Err(err).Msg("Failed to read entries")
			return
		}

		for _, entry := range entries {
			logger := mainLogger.With().Str("entry", entry).Logger()

			// we generate temporary directory names with
			// `xid.New().String()` which is always 20 char long
			if len(entry) != 20 {
				logger.Debug().Msg("found an entry that is not from us")
				continue
			} else if _, err := xid.FromString(entry); err != nil {
				logger.Debug().Err(err).Msg("found an entry that is not from us")
				continue
			}

			entryPath := filepath.Join(parentDirectory, entry)
			file, err := os.Open(entryPath)
			if err != nil {
				logger.Warn().Err(err).Msg("failed to read entry")
				continue
			} else if fi, err := file.Stat(); err != nil {
				logger.Warn().Err(err).Msg("failed to read entry")
				continue
			} else if !fi.IsDir() {
				logger.Warn().Err(err).Msg("entry is not a directory")
				continue
			} else if fi.ModTime().After(cutoff) {
				logger.Debug().Any("cutoff", cutoff).Msg("entry is more recent than cutoff, keeping it for now")
				continue
			}

			logger.Debug().Str("entry", entry).Msg("entry matches the criterias, removing it")
			if err := os.RemoveAll(entryPath); err != nil {
				logger.Warn().Err(err).Msg("failed to remove entry")
			}
		}
	}
}

// Find composer depending on the configuration
func (e *Executor) findComposer(extraBin string) (string, error) {
	if scriptDir, err := e.DetectScriptDir(); err == nil {
		for _, file := range []string{extraBin, "composer.phar", "composer"} {
			path := filepath.Join(scriptDir, file)
			e.Logger.Debug().Str("source", "Composer").Msgf(`Looking for Composer under "%s"`, path)
			d, err := os.Stat(path)
			if err != nil {
				continue
			}
			if m := d.Mode(); !m.IsDir() {
				// Yep!
				e.Logger.Debug().Str("source", "Composer").Msgf(`Found Composer as "%s"`, path)
				return path, nil
			}
		}
	}

	// fallback to default composer detection
	return findComposer(extraBin, e.Logger)
}

// Execute executes the right version of PHP depending on the configuration
func (e *Executor) Execute(loadDotEnv bool) int {
	if err := e.Config(loadDotEnv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer e.CleanupTemporaryDirectories()
	cmd := execCommand(e.Args[0], e.Args[1:]...)
	environ := append(os.Environ(), e.environ...)
	gpathname := "PATH"
	if runtime.GOOS == "windows" {
		gpathname = "Path"
	}
	fullPath := os.Getenv(gpathname)
	for _, path := range e.Paths {
		fullPath = fmt.Sprintf("%s%c%s", path, filepath.ListSeparator, fullPath)
	}
	environ = append(environ, fmt.Sprintf("%s=%s", gpathname, fullPath))
	cmd.Env = append(cmd.Env, environ...)
	cmd.Env = append(cmd.Env, e.ExtraEnv...)
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}
	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr
	cmd.Stdin = e.Stdin
	if e.Dir != "" {
		cmd.Dir = e.Dir
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	waitCh := make(chan error)
	go func() {
		waitCh <- cmd.Wait()
		close(waitCh)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)
	defer signal.Stop(sigChan)

	for {
		select {
		case sig := <-sigChan:
			if shouldSignalBeIgnored(sig) {
				continue
			}
			if err := cmd.Process.Signal(sig); err != nil {
				if err.Error() != "os: process already finished" {
					fmt.Fprintln(os.Stderr, "error sending signal", sig, err)
				}
			}
		case err := <-waitCh:
			exitCode := 0
			if err == nil {
				return exitCode
			}
			if !strings.Contains(err.Error(), "exit status") {
				fmt.Fprintln(os.Stderr, err)
			}
			if exiterr, ok := err.(*exec.ExitError); ok {
				if s, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCode = s.ExitStatus()
				}
			}
			return exitCode
		}
	}
}

// we look in the directory of the current PHP version first, then fall back to PATH
func LookPath(file string) (string, error) {
	if util.InCloud() {
		// does not make sense to look for the php store, fall back
		return exec.LookPath(file)
	}
	phpStore := phpstore.New(util.GetHomeDir(), false, nil)
	wd, _ := os.Getwd()
	v, _, warning, _ := phpStore.BestVersionForDir(wd)
	if warning != "" {
		terminal.Eprintfln("<warning>WARNING</> %s", warning)
	}
	if v == nil {
		// unable to find the current PHP version, fall back
		return exec.LookPath(file)
	}

	path := filepath.Join(filepath.Dir(v.PHPPath), file)
	d, err := os.Stat(path)
	if err != nil {
		// file does not exist, fall back
		return exec.LookPath(file)
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		// Yep!
		return path, nil
	}
	// found, but not executable, fall back
	return exec.LookPath(file)
}

// detectScriptDir tries to get the script directory from args
func detectScriptDir(args []string) string {
	script := ""
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "-f") {
			if len(arg) > 2 {
				script = arg[2:]
				break
			} else if len(args) >= i+1 {
				script = args[i+1]
				break
			}
			continue
		}
		// skip options that take an option
		for _, flag := range []string{"-c", "-d", "-r", "-B", "-R", "-F", "-E", "-S", "-t", "-z"} {
			if strings.HasPrefix(arg, flag) {
				if len(arg) == 2 {
					skipNext = true
				}
				continue
			}
		}
		// done
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		script = arg
		break
	}
	if script != "" {
		if script, err := filepath.Abs(script); err == nil {
			return filepath.Dir(script)
		}
		return filepath.Dir(script)
	}

	// fallback to the current directory
	wd, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return wd
}

func (e *Executor) PathsToWatch() []string {
	var paths []string

	if dir := e.phpiniDirForDir(); dir != "" {
		paths = append(paths, filepath.Join(dir, "php.ini"))
	}

	return paths
}

func (e *Executor) phpiniDirForDir() string {
	dir := e.scriptDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "php.ini")); err == nil {
			return dir
		}
		upDir := filepath.Dir(dir)
		if upDir == dir || upDir == "." {
			break
		}
		dir = upDir
	}
	return ""
}
