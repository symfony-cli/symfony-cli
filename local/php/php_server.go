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
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/phpstore"
	"github.com/symfony-cli/symfony-cli/local"
	fcgiclient "github.com/symfony-cli/symfony-cli/local/fcgi_client"
	"github.com/symfony-cli/symfony-cli/local/html"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/process"
)

// Server represents a PHP server process (can be php-fpm, php-cgi, or php-cli)
type Server struct {
	Version      *phpstore.Version
	logger       zerolog.Logger
	appVersion   string
	homeDir      string
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
		homeDir:      homeDir,
		projectDir:   projectDir,
		documentRoot: documentRoot,
		passthru:     passthru,
	}, nil
}

// Start starts a PHP server
func (p *Server) Start(ctx context.Context, pidFile *pid.PidFile) (*pid.PidFile, func() error, error) {
	var pathsToRemove []string
	port, err := process.FindAvailablePort()
	if err != nil {
		p.logger.Debug().Err(err).Msg("unable to find an available port")
		return nil, nil, err
	}
	p.addr = net.JoinHostPort("", strconv.Itoa(port))
	workingDir := p.documentRoot
	env := []string{}
	var binName, workerName string
	var args []string
	if p.Version.IsFPMServer() {
		fpmConfigFile := p.fpmConfigFile()
		if err := os.WriteFile(fpmConfigFile, []byte(p.defaultFPMConf()), 0644); err != nil {
			return nil, nil, errors.WithStack(err)
		}
		pathsToRemove = append(pathsToRemove, fpmConfigFile)
		binName = "php-fpm"
		workerName = "PHP-FPM"
		args = []string{p.Version.ServerPath(), "--nodaemonize", "--fpm-config", fpmConfigFile}
		if p.Version.Version[0] >= '7' {
			args = append(args, "--force-stderr")
		}
	} else if p.Version.IsCGIServer() {
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
		pathsToRemove = append(pathsToRemove, routerPath)
		addr := "127.0.0.1:" + strconv.Itoa(port)
		binName = "php"
		workerName = "PHP"
		args = []string{p.Version.ServerPath(), "-S", addr, "-d", "variables_order=EGPCS", routerPath}
		target, err := url.Parse(fmt.Sprintf("http://%s", addr))
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}
		env = append(env, "APP_FRONT_CONTROLLER="+strings.TrimLeft(p.passthru, "/"))
		p.proxy = httputil.NewSingleHostReverseProxy(target)
		p.proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(html.WrapHTML(err.Error(), html.CreateErrorTerminal("# "+err.Error()), "")))
		}
	}

	e := &Executor{
		Dir:       workingDir,
		BinName:   binName,
		Args:      args,
		scriptDir: p.projectDir,
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
			for _, path := range pathsToRemove {
				os.RemoveAll(path)
			}
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
		pw := httptest.NewRecorder()
		p.proxy.ServeHTTP(pw, r)
		return p.writeResponse(w, r, env, pw.Result())
	}
	return p.serveFastCGI(env, w, r)
}

func (p *Server) serveFastCGI(env map[string]string, w http.ResponseWriter, r *http.Request) error {
	// as the process might have been just created, it might not be ready yet
	var fcgi *fcgiclient.FCGIClient
	var err error
	max := 10
	i := 0
	for {
		if fcgi, err = fcgiclient.Dial("tcp", p.addr); err == nil {
			break
		}
		i++
		if i > max {
			return errors.Wrapf(err, "unable to connect to the PHP FastCGI process")
		}
		time.Sleep(time.Millisecond * 50)
	}
	defer fcgi.Close()
	defer r.Body.Close()

	// fetching the response from the fastcgi backend, and check for errors
	resp, err := fcgi.Request(env, r.Body)
	if err != nil {
		return errors.Wrapf(err, "unable to fetch the response from the backend")
	}

	// X-SendFile
	sendFilename := resp.Header.Get("X-SendFile")
	_, err = os.Stat(sendFilename)
	if sendFilename != "" && err == nil {
		http.ServeFile(w, r, sendFilename)
		return nil
	}
	return p.writeResponse(w, r, env, resp)
}

func (p *Server) writeResponse(w http.ResponseWriter, r *http.Request, env map[string]string, resp *http.Response) error {
	defer resp.Body.Close()
	if env["SYMFONY_TUNNEL"] != "" && env["SYMFONY_TUNNEL_ENV"] == "" {
		p.logger.Warn().Msgf("Tunnel to %s open but environment variables not exposed", env["SYMFONY_TUNNEL_BRAND"])
	}
	bodyModified := false
	if r.Method == http.MethodGet && r.Header.Get("x-requested-with") == "XMLHttpRequest" {
		var err error
		if resp.Body, err = p.tweakToolbar(resp.Body, env); err != nil {
			return err
		}
		bodyModified = true
	}
	for k, v := range resp.Header {
		if bodyModified && strings.ToLower(k) == "content-length" {
			// we drop the incoming Content-Length, it will be recomputed by Go automatically anyway
			continue
		}
		for i := 0; i < len(v); i++ {
			if w.Header().Get(k) == "" {
				w.Header().Set(k, v[i])
			} else {
				w.Header().Add(k, v[i])
			}
		}
	}
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		io.Copy(w, resp.Body)
	}
	return nil
}

func name(dir string) string {
	h := sha1.New()
	io.WriteString(h, dir)
	return fmt.Sprintf("%x", h.Sum(nil))
}
