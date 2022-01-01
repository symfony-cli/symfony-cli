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

package util

import (
	goerr "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/git"
	"gopkg.in/yaml.v2"
)

var (
	ErrProjectRootNotFoundNoGitRemote = goerr.New("project root not found, current directory not linked to a Platform.sh project")
	ErrNoGitBranchMatching            = goerr.New("current git branch name doesn't match any Platform.sh environments")
)

func GetProjectRoot(debug bool) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.WithStack(err)
	}

	if projectRoot, _ := GuessProjectRoot(currentDir, debug); projectRoot != "" {
		return projectRoot, nil
	}

	return "", errors.WithStack(ErrProjectRootNotFoundNoGitRemote)
}

func PotentialCurrentEnvironmentID(cwd string) (string, error) {
	for _, potentialEnvironment := range guessCloudBranch(cwd) {
		return potentialEnvironment, nil
	}

	return "", errors.New("no known git upstream, branch or environment name")
}

func RepositoryRootDir(currentDir string) string {
	for {
		f, err := os.Stat(filepath.Join(currentDir, ".git"))
		if err == nil && f.IsDir() {
			return currentDir
		}

		upDir := filepath.Dir(currentDir)
		if upDir == currentDir || upDir == "." {
			break
		}
		currentDir = upDir
	}

	return ""
}

func GuessProjectRoot(currentDir string, debug bool) (string, *gitInfo) {
	rootDir := RepositoryRootDir(currentDir)
	if rootDir == "" {
		return "", nil
	}
	config := GetProjectConfig(rootDir, debug)
	if config == nil {
		return "", nil
	}
	return rootDir, config
}

type gitInfo struct {
	ID string
}

func GetProjectConfig(projectRoot string, debug bool) *gitInfo {
	contents, err := ioutil.ReadFile(filepath.Join(projectRoot, ".platform", "local", "project.yaml"))
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "WARNING: unable to find Platform.sh config file: %s\n", err)
		}
		return nil
	}
	var config struct {
		ID string `yaml:"id"`
	}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "ERROR: unable to decode Platform.sh config file: %s\n", err)
		}
		return nil
	}
	return &gitInfo{
		ID: config.ID,
	}
}

func guessCloudBranch(cwd string) []string {
	localBranch := git.GetCurrentBranch(cwd)
	if localBranch == "" {
		return []string{}
	}

	branches := []string{}
	branches = append(branches, localBranch)

	if remoteBranch := git.GetUpstreamBranch(cwd, "origin", "upstream"); remoteBranch != "" {
		branches = append(branches, remoteBranch)
	}

	return branches
}
