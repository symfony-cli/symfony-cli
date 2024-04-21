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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

func (b *Book) Clone(version string) error {
	ui := terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin)
	ui.Section("Checking Book Requirements")
	ready, err := CheckRequirements()
	if err != nil {
		return err
	}
	terminal.Println("")
	if !ready {
		return errors.New("You should fix the reported issues before starting reading the book.")
	}

	ui.Section("Cloning the Repository")

	// check that version exists on Github via the API
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/the-fast-track/book-%s", version))
	if err != nil {
		return errors.Wrap(err, "unable to get version on Github")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		versions, err := Versions()
		if err != nil {
			return errors.Wrap(err, "unable to get book versions")
		}
		terminal.Println("The version you requested does not exist; available versions:")
		for _, v := range versions {
			terminal.Println(fmt.Sprintf(" - %s", v))
		}
		return errors.New("please choose a valid version")
	}

	cmd := exec.Command("git", "clone", fmt.Sprintf("https://github.com/the-fast-track/book-%s", version), b.Dir)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error cloning the Git repository for the book")
	}
	terminal.Println("")

	os.Chdir(b.Dir)
	// checkout the first step by default
	ui.Section("Getting Ready for the First Step of the Book")
	if err := b.Checkout("3"); err != nil {
		terminal.Println("")
		if !b.Debug {
			terminal.Println("Re-run the command with <comment>--debug</> to get more information about the error")
			terminal.Println("")
		}
		return err
	}
	return nil
}

func Versions() ([]string, error) {
	resp, err := http.Get("https://api.github.com/orgs/the-fast-track/repos")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get repositories from Github")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get repositories from Github")
	}
	var repos []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, errors.Wrap(err, "failed to decode response body")
	}
	versions := []string{}
	for _, repo := range repos {
		versions = append(versions, strings.Replace(repo.Name, "book-", "", 1))
	}
	return versions, nil
}
