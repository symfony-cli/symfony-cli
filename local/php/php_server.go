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
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/local"
	"github.com/symfony-cli/symfony-cli/local/html"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/process"
)

// Server represents a PHP server process (can be php-fpm, php-cgi, or php-cli)
type Server struct {
	Version      *phpstore.Version
	logger       zerolog.Logger
	StoppedChan  chan bool
	appVersion   string
	tempDir      string
	projectDir   string
	documentRoot string
	passthru     string
	addr         string
	proxy        *httputil.ReverseProxy
}

var addslashes = strings.NewReplacer("\\", "\\\\", "'", "\\'")

// NewServer creates a new PHP server backend
func NewServer(homeDir, projectDir, documentRoot, passthru, appVersion string, logger zerolog.Logger) (*Server, error) {
	logger.Debug().Str("source", "PHP").Msg("Reloading PHP versions")
	phpStore := phpstore.New(homeDir, true, nil)
	version, source, warning, err := phpStore.BestVersionForDir(projectDir)
	if warning != "" {
		logger.Warn().Str("source", "PHP").Msg(warning)
	}
	if err != nil {
		return nil, err
	}
	logger.Debug().Str("source", "PHP").Msgf("Using PHP version %s (from %s)", version.Version, source)
	return &Server{
		Version:      version,
		logger:       logger.With().Str("source", "PHP").Str("php", version.Version).Str("path", version.ServerPath()).Logger(),
		appVersion:   appVersion,
		projectDir:   projectDir,
		documentRoot: documentRoot,
		passthru:     passthru,
		StoppedChan:  make(chan bool, 1),
	}, nil
}

// Start starts a PHP server
func (p *Server) Start(ctx context.Context, pidFile *pid.PidFile) (*pid.PidFile, func() error, error) {
	p.tempDir = pidFile.TempDirectory()
	if _, err := os.Stat(p.tempDir); os.IsNotExist(err) {
		if err = os.MkdirAll(p.tempDir, 0755); err != nil {
			return nil, nil, err
		}
	}

	port, err := process.FindAvailablePort()
	if err != nil {
		p.logger.Debug().Err(err).Msg("unable to find an available port")
		return nil, nil, err
	}
	p.addr = net.JoinHostPort("", strconv.Itoa(port))

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	p.proxy = httputil.NewSingleHostReverseProxy(target)
	p.proxy.ModifyResponse = func(resp *http.Response) error {
		if err, processed := p.processToolbarInResponse(resp); processed {
			return err
		}

		if err, processed := p.processXSendFile(resp); processed {
			return err
		}

		return nil
	}
	p.proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(html.WrapHTML(err.Error(), html.CreateErrorTerminal("# "+err.Error()), "")))
	}

	workingDir := p.documentRoot
	env := []string{}
	var binName, workerName string
	var args []string
	if p.Version.IsFPMServer() {
		fpmConfigFile := p.fpmConfigFile()
		if err := os.WriteFile(fpmConfigFile, []byte(p.defaultFPMConf()), 0644); err != nil {
			return nil, nil, errors.WithStack(err)
		}
		p.proxy.Transport = &cgiTransport{}
		binName = "php-fpm"
		workerName = "PHP-FPM"
		args = []string{p.Version.ServerPath(), "--nodaemonize", "--fpm-config", fpmConfigFile}
		if p.Version.Version[0] >= '7' {
			args = append(args, "--force-stderr")
		}
	} else if p.Version.IsCGIServer() {
		p.proxy.Transport = &cgiTransport{}
		// as php-cgi reads the main php.ini file from the current directory,
		// we want to execute from another directory to be sure that we
		// are always loading the default PHP configuration
		// the local php.ini file is loaded anyway by us via PHP_INI_SCAN_DIR
		workingDir = os.TempDir()
		binName = "php-cgi"
		workerName = "PHP-CGI"
		errorLog := "/dev/fd/2"
		if runtime.GOOS == "windows" {
			errorLog = pidFile.LogFile()
		}
		args = []string{p.Version.ServerPath(), "-b", strconv.Itoa(port), "-d", "error_log=" + errorLog}
	} else {
		routerPath := p.phpRouterFile()
		if err := os.WriteFile(routerPath, phprouter, 0644); err != nil {
			return nil, nil, errors.WithStack(err)
		}
		binName = "php"
		workerName = "PHP"
		args = []string{p.Version.ServerPath(), "-S", "127.0.0.1:" + strconv.Itoa(port), "-d", "variables_order=EGPCS", routerPath}
		env = append(env, "APP_FRONT_CONTROLLER="+strings.TrimLeft(p.passthru, "/"))
	}

	e := &Executor{
		Dir:       workingDir,
		BinName:   binName,
		Args:      args,
		scriptDir: p.projectDir,
		Logger:    p.logger,
	}
	p.logger.Info().Int("port", port).Msg("listening")

	phpPidFile := pid.New(pidFile.Dir, append([]string{p.Version.ServerPath()}, e.Args[1:]...))
	if phpPidFile.IsRunning() {
		if err := phpPidFile.Stop(); err != nil {
			return nil, nil, errors.Wrapf(err, "PHP was already running, but we were unable to kill it")
		}
	}
	phpPidFile.CustomName = workerName
	phpPidFile.Watched = e.PathsToWatch()
	runner, err := local.NewRunner(phpPidFile, local.RunnerModeLoopAttached)
	if err != nil {
		return phpPidFile, nil, err
	}
	runner.AlwaysRestartOnExit = true
	runner.BuildCmdHook = func(cmd *exec.Cmd) error {
		cmd.Dir = workingDir

		if err = e.Config(false); err != nil {
			return err
		}

		cmd.Env = append(cmd.Env, e.environ...)
		cmd.Env = append(cmd.Env, env...)

		return nil
	}

	return phpPidFile, func() error {
		defer func() {
			e.CleanupTemporaryDirectories()
			p.StoppedChan <- true
		}()

		return errors.Wrap(errors.WithStack(runner.Run()), "PHP server exited unexpectedly")
	}, nil
}

// Serve serves an HTTP request
func (p *Server) Serve(w http.ResponseWriter, r *http.Request, env map[string]string) error {
	if p.passthru == "" {
		return errors.Errorf(`Unable to guess the web front controller under "%s"`, p.projectDir)
	}
	for k, v := range p.generateEnv(r) {
		env[k] = v
	}

	// inject our ResponseWriter and our environment into the request's context
	// to allow for processing at a later stage
	r = r.WithContext(context.WithValue(r.Context(), responseWriterContextKey, w))
	r = r.WithContext(context.WithValue(r.Context(), environmentContextKey, env))

	if p.Version.IsCLIServer() {
		rid := xid.New().String()
		r.Header.Add("__SYMFONY_LOCAL_REQUEST_ID__", rid)
		envPath := p.phpRouterFile() + "-" + rid + "-env"
		envContent := "<?php "
		for k, v := range env {
			envContent += fmt.Sprintf("$_ENV['%s'] = '%s';\n", addslashes.Replace(k), addslashes.Replace(v))
		}
		err := errors.WithStack(os.WriteFile(envPath, []byte(envContent), 0644))
		if err != nil {
			return err
		}
		defer os.Remove(envPath)
	}

	p.proxy.ServeHTTP(w, r)
	return nil
}

func name(dir string) string {
	h := sha1.New()
	io.WriteString(h, dir)
	return fmt.Sprintf("%x", h.Sum(nil))
}
