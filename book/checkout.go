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

package book

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/local/upsun"
	"github.com/symfony-cli/terminal"
)

func (b *Book) Checkout(step string) error {
	// FIXME: keep vendor/ node_modules/ around before git clean, but them back as they will be updated the right way, less Internet traffic
	// FIXME: if the checkout is to a later step, no need to remove the DB, we can just migrate it
	os.Chdir(b.Dir)
	step = strings.ReplaceAll(step, ".", "-")
	tag := fmt.Sprintf("step-%s", step)
	branch := "work-" + tag
	printBanner("<comment>[GIT]</> Check for not yet committed changes", b.Debug)
	if err := executeCommand([]string{"git", "diff-index", "--quiet", "HEAD", "--"}, b.Debug, false, nil); err != nil {
		if !b.Force && !terminal.AskConfirmation("<warning>WARNING</> There are not yet committed changes in the repository, do you want to discard them?", true) {
			return nil
		}
	}

	printBanner("<comment>[GIT]</> Check Git un-tracked files", b.Debug)
	cmd := exec.Command("git", "ls-files", "--exclude-standard", "--others")
	cmd.Env = os.Environ()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil || buf.String() != "" {
		if !b.Debug {
			terminal.Println("<error>[ KO ]</>")
		}
		terminal.Println(buf.String())
		if !b.Force && !terminal.AskConfirmation("<warning>WARNING</> There are un-tracked files in the repository, do you want to discard them?", true) {
			return nil
		}
	} else if !b.Debug {
		terminal.Println("<info>[ OK ]</>")
	}

	if !b.Force && !b.AutoConfirm && !terminal.AskConfirmation("<warning>WARNING</> All current code, data, and containers are going to be REMOVED, do you confirm?", true) {
		return nil
	}

	if !b.Debug {
		s := terminal.NewSpinner(terminal.Stderr)
		s.Start()
		defer s.Stop()
	}

	terminal.Println("")

	printBanner("<comment>[GIT]</> Removing Git ignored files (vendor, cache, ...)", b.Debug)
	if err := executeCommand([]string{"git", "clean", "-d", "-f", "-x"}, b.Debug, false, nil); err != nil {
		return err
	}
	printBanner("<comment>[GIT]</> Resetting Git staged files", b.Debug)
	if err := executeCommand([]string{"git", "reset", "HEAD", "."}, b.Debug, false, nil); err != nil {
		return err
	}
	printBanner("<comment>[GIT]</> Removing un-tracked Git files", b.Debug)
	if err := executeCommand([]string{"git", "checkout", "."}, b.Debug, false, nil); err != nil {
		return err
	}

	printBanner("<comment>[WEB]</> Adding .env.local", b.Debug)
	emptyFile, err := os.Create(filepath.Join(b.Dir, ".env.local"))
	if err != nil {
		return err
	}
	emptyFile.Close()
	if !b.Debug {
		terminal.Println("<info>[ OK ]</>")
	}

	printBanner("<comment>[WEB]</> Stopping Docker Containers", b.Debug)
	hasDocker := false
	for _, filename := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if _, err = os.Stat(filepath.Join(b.Dir, filename)); err == nil {
			hasDocker = true
			break
		}
	}
	if hasDocker {
		if err := executeCommand(append(dockerComposeBin(), "down", "--remove-orphans", "--volume"), b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Stopping the Local Web Server", b.Debug)
	executeCommand([]string{"symfony", "server:stop"}, b.Debug, true, nil)

	brand := upsun.GuessCloudFromDirectory(b.Dir)
	printBanner(fmt.Sprintf("<comment>[WEB]</> Stopping the %s tunnel", brand.Name), b.Debug)
	if err := executeCommand([]string{"symfony", "tunnel:close", "-y"}, b.Debug, true, nil); err != nil {
		return err
	}

	printBanner("<comment>[GIT]</> Checking out the step", b.Debug)
	if err := executeCommand([]string{"git", "checkout", "-B", branch, tag}, b.Debug, false, nil); err != nil {
		return err
	}

	printBanner("<comment>[SPA]</> Stopping the Local Web Server", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "spa")); err == nil {
		executeCommand([]string{"symfony", "server:stop", "--dir", filepath.Join(b.Dir, "spa")}, b.Debug, true, nil)
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Installing Composer dependencies (might take some time)", b.Debug)
	if err := executeCommand([]string{"symfony", "composer", "install"}, b.Debug, false, nil); err != nil {
		return err
	}

	printBanner("<comment>[WEB]</> Adding .env.local", b.Debug)
	if emptyFile, err = os.Create(filepath.Join(b.Dir, ".env.local")); err != nil {
		return err
	}
	emptyFile.Close()
	if !b.Debug {
		terminal.Println("<info>[ OK ]</>")
	}

	printBanner("<comment>[WEB]</> Starting Docker Compose", b.Debug)
	if hasDocker {
		if err := executeCommand(append(dockerComposeBin(), "up", "-d"), b.Debug, false, nil); err != nil {
			return err
		}
		printBanner("<comment>[WEB]</> Waiting for the Containers to be ready", b.Debug)
		if _, err := os.Stat(filepath.Join(b.Dir, "src", "MessageHandler", "CommentMessageHandler.php")); err == nil {
			// FIXME: ping rabbitmq instead
			time.Sleep(10 * time.Second)
		} else {
			// FIXME: ping PostgreSQL instead
			time.Sleep(5 * time.Second)
		}
		if !b.Debug {
			terminal.Println("<info>[ OK ]</>")
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Migrating the database", b.Debug)
	files, err := filepath.Glob(filepath.Join(b.Dir, "src", "Migrations", "*.php"))
	hasMigrations := err == nil && len(files) > 0
	if !hasMigrations {
		files, err = filepath.Glob(filepath.Join(b.Dir, "migrations", "*.php"))
		hasMigrations = err == nil && len(files) > 0
	}
	if hasMigrations {
		if err := executeCommand([]string{"symfony", "console", "doctrine:migrations:migrate", "-n"}, b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Inserting Fixtures", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "src", "DataFixtures")); err == nil {
		if err := executeCommand([]string{"symfony", "console", "doctrine:fixtures:load", "-n"}, b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Installing Node dependencies (might take some time)", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "package.json")); err == nil {
		args := []string{"npm", "install"}
		if _, err := os.Stat(filepath.Join(b.Dir, "yarn.lock")); err == nil {
			// old version of the book using Yarn instead of npm
			args = []string{"yarn", "install"}
		}
		if err := executeCommand(args, b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Building CSS and JS assets", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "package.json")); err == nil {
		args := []string{"npx", "encore", "dev"}
		if _, err := os.Stat(filepath.Join(b.Dir, "yarn.lock")); err == nil {
			args = []string{"yarn", "encore", "dev"}
		}
		if err := executeCommand(args, b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[WEB]</> Starting the Local Web Server", b.Debug)
	if err := executeCommand([]string{"symfony", "server:start", "-d"}, b.Debug, false, nil); err != nil {
		return err
	}

	printBanner("<comment>[WEB]</> Starting Message Consumer", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "src", "MessageHandler", "CommentMessageHandler.php")); err == nil {
		if err := executeCommand([]string{"symfony", "run", "-d", "--watch", "config,src,templates,vendor", "symfony", "console", "messenger:consume", "async", "-vv"}, b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[SPA]</> Installing Node dependencies (might take some time)", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "spa")); err == nil {
		os.Chdir(filepath.Join(b.Dir, "spa"))
		args := []string{"npm", "install"}
		if _, err := os.Stat(filepath.Join(b.Dir, "yarn.lock")); err == nil {
			// old version of the book using Yarn instead of npm
			args = []string{"yarn", "install"}
		}
		if err := executeCommand(args, b.Debug, false, nil); err != nil {
			return err
		}
		os.Chdir(b.Dir)
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[SPA]</> Building CSS and JS assets", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "spa")); err == nil {
		cmd := exec.Command("symfony", "var:export", "SYMFONY_PROJECT_DEFAULT_ROUTE_URL")
		cmd.Env = os.Environ()
		var endpoint, stderr bytes.Buffer
		cmd.Stdout = &endpoint
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "unable to get the URL of the local web server")
		}
		if endpoint.String() == "" {
			return errors.Errorf("unable to get the URL of the local web server:\n%s\n%s", stderr.String(), endpoint.String())
		}
		os.Chdir(filepath.Join(b.Dir, "spa"))
		env := append(os.Environ(), "API_ENDPOINT="+endpoint.String())
		args := []string{"npx", "encore", "dev"}
		if _, err := os.Stat(filepath.Join(b.Dir, "yarn.lock")); err == nil {
			args = []string{"yarn", "encore", "dev"}
		}
		if err := executeCommand(args, b.Debug, false, env); err != nil {
			return err
		}
		os.Chdir(b.Dir)
	} else {
		terminal.Println("Skipped for this step")
	}

	printBanner("<comment>[SPA]</> Starting the Local Web Server", b.Debug)
	if _, err := os.Stat(filepath.Join(b.Dir, "spa")); err == nil {
		if err := executeCommand([]string{"symfony", "server:start", "-d", "--passthru", "index.html", "--dir", filepath.Join(b.Dir, "spa")}, b.Debug, false, nil); err != nil {
			return err
		}
	} else {
		terminal.Println("Skipped for this step")
	}

	terminal.Println("")
	ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
	ui.Success("All done!")
	return nil
}

func printBanner(msg string, debug bool) {
	if debug {
		terminal.Println("")
		ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
		ui.Section(msg)
	} else {
		terminal.Printf("%s: ", msg)
	}
}

func executeCommand(args []string, debug, skipErrors bool, env []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	if env == nil {
		cmd.Env = os.Environ()
	}
	var buf bytes.Buffer
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	if err := cmd.Run(); err != nil && !skipErrors {
		if !debug {
			terminal.Println("<error>[ KO ]</>")
		}
		terminal.Print(buf.String())
		return err
	}
	if !debug {
		terminal.Println("<info>[ OK ]</>")
	}
	return nil
}

func dockerComposeBin() []string {
	if path, err := exec.LookPath("docker-compose"); err == nil {
		return []string{path}
	}
	return []string{"docker", "compose"}
}
