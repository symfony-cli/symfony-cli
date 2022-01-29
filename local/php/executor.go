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
	"io/ioutil"
	"math/rand"
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

// Config determines the right version of PHP depending on the configuration (+ its configuration)
func (e *Executor) Config(loadDotEnv bool) error {
	// reset environment
	e.environ = make([]string, 0)

	if len(e.Args) < 1 {
		return errors.New("args cannot be empty")
	}
	if e.scriptDir == "" {
		if e.SkipNbArgs == 0 {
			e.SkipNbArgs = 1
		}
		if e.SkipNbArgs < 0 {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			e.scriptDir = wd
		} else {
			e.scriptDir = detectScriptDir(e.Args[e.SkipNbArgs:])
		}
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
	phpDir := filepath.Join(cliDir, "tmp", xid.New().String(), "bin")
	e.tempDir = phpDir
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
	e.Paths = append([]string{path}, e.Paths...)
	if phpiniArgs {
		wd, _ := os.Getwd()
		e.iniDir, err = e.generateLocalPhpIniFile(wd, cliDir)
		if err != nil {
			return err
		}
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

// Find composer depending on the configuration
func (e *Executor) findComposer(extraBin string) (string, error) {
	if e.Config(false) == nil {
		for _, file := range []string{extraBin, "composer.phar", "composer"} {
			path := filepath.Join(e.scriptDir, file)
			d, err := os.Stat(path)
			if err != nil {
				continue
			}
			if m := d.Mode(); !m.IsDir() {
				// Yep!
				return path, nil
			}
		}
	}

	// fallback to default composer detection
	return findComposer(extraBin)
}

// Execute executes the right version of PHP depending on the configuration
func (e *Executor) Execute(loadDotEnv bool) int {
	if err := e.Config(loadDotEnv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() {
		if e.iniDir != "" {
			os.RemoveAll(e.iniDir)
		}
		if e.tempDir != "" {
			os.RemoveAll(e.tempDir)
		}
	}()
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

	sigChan := make(chan os.Signal)
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

func (e *Executor) generateLocalPhpIniFile(wd, cliDir string) (string, error) {
	rand.Seed(time.Now().UnixNano())
	tmpDir := filepath.Join(cliDir, "var", fmt.Sprintf("%s_%d", name(wd), rand.Intn(99999999)))
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", errors.Errorf("unable to create temp dir %s", tmpDir)
	}
	ini := GetPHPINISettings(e.scriptDir).Bytes()
	// don't write an empty ini file as it might be read as a non-valid ini file by PHP (and Composer)
	if len(ini) > 0 {
		extraIni := filepath.Join(tmpDir, "1-extra.ini")
		if err := ioutil.WriteFile(extraIni, ini, 0666); err != nil {
			os.RemoveAll(tmpDir)
			return "", errors.Wrapf(err, "unable to create temp file \"%s\"", extraIni)
		}
		return tmpDir, nil
	}
	os.RemoveAll(tmpDir)
	return "", nil
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
