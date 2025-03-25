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

package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soheilhy/cmux"
	"github.com/symfony-cli/cert"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/symfony-cli/humanlog"
	"github.com/symfony-cli/symfony-cli/local"
	"github.com/symfony-cli/symfony-cli/local/logs"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/project"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/reexec"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
	"golang.org/x/sync/errgroup"
)

var localWebServerProdWarningMsg = "The local web server is optimized for local development and MUST never be used in a production setup."
var localWebServerTlsKeyLogWarningMsg = "Logging TLS master key is enabled. It means TLS connections between the client and this server will be INSECURE. This is NOT recommended unless you are debugging the connections."
var localWebServerAllowsCORSLogWarningMsg = "Cross-origin resource sharing (CORS) is enabled for all requests.\nYou may want to use https://github.com/nelmio/NelmioCorsBundle to have better control over HTTP headers."

var localServerStartCmd = &console.Command{
	Category:    "local",
	Name:        "server:start",
	Aliases:     []*console.Alias{{Name: "server:start"}, {Name: "serve"}},
	Usage:       "Run a local web server",
	Description: localWebServerProdWarningMsg,
	Flags: []console.Flag{
		dirFlag,
		&console.BoolFlag{Name: "allow-http", Usage: "Prevent auto-redirection from HTTP to HTTPS"},
		&console.StringFlag{Name: "document-root", Usage: "Project document root (auto-configured by default)"},
		&console.StringFlag{Name: "passthru", Usage: "Project passthru index (auto-configured by default)"},
		&console.IntFlag{Name: "port", DefaultValue: 8000, Usage: "Preferred HTTP port"},
		&console.StringFlag{Name: "listen-ip", DefaultValue: "127.0.0.1", Usage: "The IP on which the CLI should listen"},
		&console.BoolFlag{Name: "allow-all-ip", Usage: "Listen on all the available interfaces"},
		&console.BoolFlag{Name: "daemon", Aliases: []string{"d"}, Usage: "Run the server in the background"},
		&console.BoolFlag{Name: "no-humanize", Usage: "Do not format JSON logs"},
		&console.StringFlag{Name: "p12", Usage: "Name of the file containing the TLS certificate to use in p12 format"},
		&console.BoolFlag{Name: "no-tls", Usage: "Use HTTP instead of HTTPS"},
		&console.BoolFlag{Name: "use-gzip", Usage: "Use GZIP"},
		&console.StringFlag{
			Name:  "tls-key-log-file",
			Usage: "Destination for TLS master secrets in NSS key log format",
			// If 'SSLKEYLOGFILE' environment variable is set, uses this as a
			// destination of TLS key log. In this context, the name
			// 'SSLKEYLOGFILE' is common, so using 'SSL' instead of 'TLS' name.
			// This environment variable is preferred than the key log file
			// from the console argument.
			EnvVars: []string{"SSLKEYLOGFILE"},
		},
		&console.BoolFlag{Name: "no-workers", Usage: "Do not start workers"},
		&console.BoolFlag{Name: "allow-cors", Usage: "Allow Cross-origin resource sharing (CORS) requests"},
	},
	Action: func(c *console.Context) error {
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		projectDir, err := getProjectDir(c.String("dir"))
		if err != nil {
			return err
		}
		pidFile := pid.New(projectDir, nil)
		pidFile.CustomName = pid.WebServerName
		if pidFile.IsRunning() {
			ui.Warning("The local web server is already running")
			return errors.WithStack(printWebServerStatus(projectDir))
		}
		if err := cleanupWebServerFiles(projectDir, pidFile); err != nil {
			return err
		}

		homeDir := util.GetHomeDir()

		if err := reexec.NotifyForeground("boot"); err != nil {
			terminal.Logger.Error().Msg("Unable to go to the background: %s.\nAborting\n" + err.Error())
			return console.Exit("", 1)
		}

		reexec.NotifyForeground("config")
		config, fileConfig, err := project.NewConfigFromContext(c, projectDir)
		if err != nil {
			return errors.WithStack(err)
		}
		config.HomeDir = homeDir

		if config.Daemon && !reexec.IsChild() {
			varDir := filepath.Join(homeDir, "var")
			if err := os.MkdirAll(varDir, 0755); err != nil {
				return errors.Wrap(err, "Could not create status file")
			}
			if err := reexec.Background(varDir); err != nil {
				if _, isExitCoder := err.(console.ExitCoder); isExitCoder {
					return err
				}
				terminal.Eprintln("Impossible to go to the background")
				terminal.Eprintln("Continue in foreground")
				config.Daemon = false
			} else {
				terminal.Eprintfln("Stream the logs via <info>%s server:log</>", c.App.HelpName)
				return nil
			}
		}

		shutdownCh := make(chan bool)
		go func() {
			sigsCh := make(chan os.Signal, 10)
			signal.Notify(sigsCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
			<-sigsCh
			signal.Stop(sigsCh)
			shutdownCh <- true
		}()

		reexec.NotifyForeground("proxy")
		proxyConfig, err := proxy.Load(homeDir)
		if err != nil {
			return errors.WithStack(err)
		}
		if fileConfig != nil && fileConfig.Proxy != nil {
			if err := proxyConfig.ReplaceDirDomains(projectDir, fileConfig.Proxy.Domains); err != nil {
				return errors.WithStack(err)
			}
		}

		reexec.NotifyForeground("tls")
		if !config.NoTLS && config.PKCS12 == "" {
			ca, err := cert.NewCA(filepath.Join(homeDir, "certs"))
			if err != nil {
				return errors.WithStack(err)
			} else if !ca.HasCA() {
				ui.Warning(fmt.Sprintf(`run "%s server:ca:install" first if you want to run the web server with TLS support, or use "--p12" or "--no-tls" to avoid this warning`, c.App.HelpName))
				config.NoTLS = true
			} else {
				p12 := filepath.Join(homeDir, "certs", "default.p12")
				if _, err := os.Stat(p12); os.IsNotExist(err) {
					if err := ca.LoadCA(); err != nil {
						return errors.Wrap(err, "Failed to generate a default certificate for localhost.")
					}
					err := ca.MakeCert(p12, []string{"localhost", "127.0.0.1", "::1"})
					if err != nil {
						return errors.Wrap(err, "Failed to generate a default certificate for localhost.")
					}
				} else if err == nil {
					if err := ca.LoadCA(); err != nil {
						return errors.Wrap(err, "Failed to load the default certificate for localhost.")
					}
					if ca.IsExpired() {
						ui.Warning(fmt.Sprintf(`Your local CA is expired, run "%s %s --renew" first to renew it`, c.App.HelpName, localServerCAInstallCmd.FullName()))
					} else if ca.MustBeRegenerated() {
						ui.Warning(fmt.Sprintf(`Your local CA must be regenerated, run "%s %s --renew" first to renew it`, c.App.HelpName, localServerCAInstallCmd.FullName()))
					}
				}
				config.PKCS12 = p12
			}
		}

		if config.TlsKeyLogFile != "" {
			ui.Warning(localWebServerTlsKeyLogWarningMsg)
		}

		if config.AllowCORS {
			ui.Warning(localWebServerAllowsCORSLogWarningMsg)
		}

		lw, err := pidFile.LogWriter()
		if err != nil {
			return err
		}
		config.Logger = zerolog.New(lw).With().Str("source", "server").Timestamp().Logger()
		p, err := project.New(config)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tailer := logs.Tailer{
			Follow:     true,
			NoHumanize: c.Bool("no-humanize"),
			LinesNb:    10, // needed to catch early logs
		}

		errChan := make(chan error, 1)

		if !reexec.IsChild() {
			tailer.Watch(pidFile)
		}

		if p.PHPServer != nil {
			reexec.NotifyForeground("php")
			phpPidFile, phpStartCallback, err := p.PHPServer.Start(ctx, pidFile)
			if err != nil {
				return err
			}

			// We retrieve a reader on logs as soon as possible to be able to
			// display error logs in case of startup errors. We can't do it
			// later as the log file will already be deleted.
			logs, err := phpPidFile.LogReader()
			if err != nil {
				return err
			}

			if !reexec.IsChild() {
				tailer.WatchAdditionalPidFile(phpPidFile)
			}

			// we run FPM in its own goroutine to allow it to run even when
			// foreground is forced
			go func() { errChan <- phpStartCallback() }()

			// Give time to PHP to fail or to be ready
			select {
			case err := <-errChan:
				terminal.Logger.Error().Msgf("Unable to start %s", phpPidFile.CustomName)

				humanizer := humanlog.NewHandler(&humanlog.Options{
					SkipUnchanged: true,
					WithSource:    true,
				})

				buf := bytes.Buffer{}
				fmt.Fprintf(&buf, "%s failed to start:\n", phpPidFile.CustomName)

				scanner := bufio.NewScanner(logs)
				for scanner.Scan() {
					buf.Write(humanizer.Simplify(scanner.Bytes()))
					buf.WriteRune('\n')
				}

				ui.Error(buf.String())

				if err != nil {
					return err
				}
				return nil
			case err := <-phpPidFile.WaitForPid():
				// PHP started, we can close logs and go ahead
				logs.Close()
				if err != nil {
					return err
				}
			}
		}

		reexec.NotifyForeground("http")
		port, err := p.HTTP.Start(errChan)
		if err != nil {
			return err
		}

		scheme := "https"
		if config.NoTLS {
			scheme = "http"
		}

		address := config.ListenIp
		if c.Bool("allow-all-ip") {
			address = "0.0.0.0"
		}

		msg := "Web server listening\n"
		if p.PHPServer != nil {
			msg += fmt.Sprintf("     The Web server is using %s %s\n", p.PHPServer.Version.ServerTypeName(), p.PHPServer.Version.Version)
		}
		msg += fmt.Sprintf("\n     <href=%s://%s:%d>%s://%s:%d</>", scheme, address, port, scheme, address, port)
		if proxyConf, err := proxy.Load(homeDir); err == nil {
			for _, domain := range proxyConf.GetDomains(projectDir) {
				msg += fmt.Sprintf("\n     <href=%s://%s>%s://%s</>", scheme, domain, scheme, domain)
			}
		}

		select {
		case err := <-errChan:
			if err != cmux.ErrListenerClosed && err != http.ErrServerClosed {
				return err
			}
		default:
			if err := pidFile.Write(os.Getpid(), port, scheme); err != nil {
				return err
			}

			reexec.NotifyForeground("listening")
			ui.Warning(localWebServerProdWarningMsg)
			if config.ListenIp == "127.0.0.1" {
				ui.Warning(`Please note that the Symfony CLI only listens on 127.0.0.1 by default since version 5.10.3.
          You can use the --allow-all-ip or --listen-ip flags to change this behavior.`)
			}
			ui.Success(msg)
		}

		if !reexec.IsChild() {
			go tailer.Tail(terminal.Stderr)
		}

		if fileConfig != nil && !config.NoWorkers {
			reexec.NotifyForeground("workers")

			_, isDockerComposeWorkerConfigured := fileConfig.Workers[project.DockerComposeWorkerKey]
			var dockerWg sync.WaitGroup
			if isDockerComposeWorkerConfigured {
				dockerWg.Add(1)
			}

			for name, worker := range fileConfig.Workers {
				pidFile := pid.New(projectDir, worker.Cmd)
				if pidFile.IsRunning() {
					terminal.Eprintfln("<warning>WARNING</> Unable to start worker \"%s\": it is already running for this project as PID %d", name, pidFile.Pid)
					continue
				}
				pidFile.Watched = worker.Watch
				pidFile.CustomName = name

				// we run each worker in its own goroutine for several reasons:
				// * to get things up and running faster
				// * to allow all commands to run when foreground is forced
				go func(name string, pidFile *pid.PidFile) {
					runner, err := local.NewRunner(pidFile, local.RunnerModeLoopAttached)
					if err != nil {
						terminal.Eprintfln("<warning>WARNING</> Unable to start worker \"%s\": %s", name, err)
						return
					}

					env, err := envs.GetEnv(pidFile.Dir, terminal.IsDebug())
					if err != nil {
						errChan <- errors.WithStack(err)
						return
					}

					runner.BuildCmdHook = func(cmd *exec.Cmd) error {
						cmd.Env = append(cmd.Env, envs.AsSlice(env)...)
						return nil
					}

					if name == project.DockerComposeWorkerKey {
						originalBuildCmdHook := runner.BuildCmdHook

						runner.BuildCmdHook = func(cmd *exec.Cmd) error {
							cmd.Args = append(cmd.Args, "--wait")

							return originalBuildCmdHook(cmd)
						}

						runner.SuccessHook = func(runner *local.Runner, cmd *exec.Cmd) {
							terminal.Eprintln("<info>INFO</> Docker Compose is now up, switching to non detached mode")

							// set up the worker for an immediate restart so
							// that it starts monitoring the containers as soon
							// as possible after the initial startup
							runner.AlwaysRestartOnExit = true
							// but next time this process is successful we don't
							// have to do anything specific
							runner.SuccessHook = nil
							// and we move back AlwaysRestartOnExit to false

							runner.BuildCmdHook = func(cmd *exec.Cmd) error {
								runner.AlwaysRestartOnExit = false

								return originalBuildCmdHook(cmd)
							}

							dockerWg.Done()
						}
					} else if isDockerComposeWorkerConfigured {
						terminal.Eprintfln("<info>INFO</> Worker \"%s\" waiting for Docker Compose to be up", name)
						dockerWg.Wait()
					}

					ui.Success(fmt.Sprintf("Started worker \"%s\"", name))
					if err := runner.Run(); err != nil {
						terminal.Eprintfln("<warning>WARNING</> Worker \"%s\" exited with an error: %s", name, err)
					}
				}(name, pidFile)
			}
		}

		reexec.NotifyForeground(reexec.UP)
		if reexec.IsChild() {
			terminal.RemapOutput(lw, lw).SetDecorated(true)
		}

		select {
		case err := <-errChan:
			return err
		case <-shutdownCh:
			terminal.Eprintln("")
			terminal.Eprintln("Shutting down! Waiting for all workers to be done.")
			err := waitForWorkers(projectDir, pidFile)
			// wait for the PHP Server to be done cleaning up
			if p.PHPServer != nil {
				<-p.PHPServer.StoppedChan
			}
			pidFile.CleanupDirectories()
			if err != nil {
				return err
			}
			terminal.Eprintln("")
			ui.Success("Stopped all processes successfully")
		}
		return nil
	},
}

func cleanupWebServerFiles(projectDir string, pidFile *pid.PidFile) error {
	pids := pid.AllWorkers(projectDir)
	var g errgroup.Group
	for _, p := range pids {
		if p.IsRunning() {
			g.Go(p.Stop)
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if err := pidFile.Remove(); err != nil {
		return err
	}
	return nil
}

func waitForWorkers(projectDir string, pidFile *pid.PidFile) error {
	pids := pid.AllWorkers(projectDir)
	if len(pids) < 1 {
		return nil
	}

	var g errgroup.Group
	for _, p := range pids {
		g.Go(p.WaitForExit)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if err := pidFile.Remove(); err != nil {
		return err
	}
	return nil
}
