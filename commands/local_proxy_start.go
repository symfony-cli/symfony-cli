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
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/cert"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/humanlog"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/reexec"
	"github.com/symfony-cli/symfony-cli/util"
	"github.com/symfony-cli/terminal"
)

var localProxyStartCmd = &console.Command{
	Category: "local",
	Name:     "proxy:start",
	Aliases:  []*console.Alias{{Name: "proxy:start"}},
	Usage:    "Start the local proxy server (local domains support)",
	Flags: []console.Flag{
		&console.BoolFlag{Name: "foreground", Aliases: []string{"f"}, Usage: "Run the proxy server in the foreground"},
		&console.BoolFlag{Name: "no-humanize", Usage: "Do not format JSON logs"},
		&console.StringFlag{Name: "host", Aliases: []string{"ip"}, Usage: "Host or IP to expose in PAC file"},
	},
	Action: func(c *console.Context) error {
		homeDir := util.GetHomeDir()

		pidFile := pid.New("__proxy__", nil)
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		if pidFile.IsRunning() {
			ui.Success(fmt.Sprintf("The proxy server is already running at port %d", pidFile.Port))
			return nil
		}

		if !c.Bool("foreground") && !reexec.IsChild() {
			varDir := filepath.Join(homeDir, "var")
			if err := os.MkdirAll(varDir, 0755); err != nil {
				return errors.Wrap(err, "Could not create status file")
			}
			if err := reexec.Background(varDir); err != nil {
				if _, isExitCoder := err.(console.ExitCoder); isExitCoder {
					return errors.WithStack(err)
				}
				terminal.Printfln("Impossible to go to the background: %s", err)
				terminal.Println("Continue in foreground")
			} else {
				return nil
			}
		}

		if err := reexec.NotifyForeground("boot"); err != nil {
			return console.Exit(fmt.Sprintf("Unable to go to the background: %s, aborting", err), 1)
		}

		ca, err := cert.NewCA(filepath.Join(homeDir, "certs"))
		if err != nil {
			terminal.Logger.Warn().Msg("Disabling TLS support: unable to load the local Certificate Authority")
		} else if !ca.HasCA() {
			ca = nil
			terminal.Logger.Warn().Msg("Disabling TLS support: no local Certificate Authority, generate one via server:ca:install")
		}
		if ca != nil {
			if err := ca.LoadCA(); err != nil {
				return errors.WithStack(err)
			}
			if ca.IsExpired() {
				ui.Warning(fmt.Sprintf(`Your local CA is expired, run "%s %s --renew" first to renew it`, c.App.HelpName, localServerCAInstallCmd.FullName()))
			} else if ca.MustBeRegenerated() {
				ui.Warning(fmt.Sprintf(`Your local CA must be regenerated, run "%s %s --renew" first to renew it`, c.App.HelpName, localServerCAInstallCmd.FullName()))
			}
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if terminal.IsVerbose() {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		logFile := filepath.Join(homeDir, "log", "proxy.log")
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
			return errors.WithStack(err)
		}
		os.Remove(logFile)
		f, err := os.Create(logFile)
		if err != nil {
			return errors.WithStack(err)
		}
		var lw io.Writer = f
		logger := zerolog.New(decorateLogger(lw, c.Bool("no-humanize"))).With().Timestamp().Logger()

		config, err := proxy.Load(homeDir)
		if err != nil {
			return errors.WithStack(err)
		}

		if c.IsSet("host") {
			config.Host = c.String("host")
			if err = config.Save(); err != nil {
				return errors.WithStack(err)
			}
		}

		spinner := terminal.NewSpinner(terminal.Stderr)
		spinner.Start()
		defer spinner.Stop()

		if !c.Bool("foreground") && reexec.IsChild() {
			logger = zerolog.New(lw).With().Timestamp().Logger()
		}

		proxy := proxy.New(config, ca, log.New(logger, "", 0), terminal.GetLogLevel() >= 5)
		errChan := make(chan error)
		go func() {
			errChan <- proxy.Start()
		}()

		// wait for a few seconds for the server to start or fail immediately
		timer := time.NewTimer(2 * time.Second)
		select {
		case err := <-errChan:
			if err != nil {
				timer.Stop()
				return errors.WithStack(err)
			}
		case <-timer.C:
			spinner.Stop()
			ui.Success(fmt.Sprintf("Proxy server listening on http://%s:%d", config.Host, config.Port))
		}

		if err := pidFile.Write(os.Getpid(), config.Port, "http"); err != nil {
			return errors.WithStack(err)
		}

		if !c.Bool("foreground") && reexec.IsChild() {
			terminal.RemapOutput(lw, lw).SetDecorated(true)
			if err = reexec.NotifyForeground(reexec.UP); err != nil {
				return errors.WithStack(err)
			}
		} else {
			defer func() {
				_ = pidFile.Remove()
			}()
		}

		shutdownCh := make(chan bool, 1)
		go func() {
			sigsCh := make(chan os.Signal, 1)
			signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
			<-sigsCh
			signal.Stop(sigsCh)
			shutdownCh <- true
		}()

		<-shutdownCh
		return nil
	},
}

func decorateLogger(lw io.Writer, noHumanize bool) io.Writer {
	var stderr io.Writer
	stderr = terminal.Stderr
	if !noHumanize {
		stderr = humanlog.New(terminal.Stderr, &humanlog.Options{
			SkipUnchanged: true,
			WithSource:    true,
		})
	}
	return io.MultiWriter(stderr, lw)
}
