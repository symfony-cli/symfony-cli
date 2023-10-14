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

package platformsh

import (
	goerr "errors"
	"fmt"
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

type Project struct {
	ID  string
	App string
	Env string
}

func ProjectFromDir(dir string, debug bool) (*Project, error) {
	projectRoot := repositoryRootDir(dir)
	if projectRoot == "" {
		return nil, errors.New("unable to get project repository root")
	}
	projectID := getProjectID(projectRoot, debug)
	if projectID == "" {
		return nil, errors.New("unable to get project id")
	}
	envID := git.GetCurrentBranch(projectRoot)
	if envID == "" {
		return nil, errors.New("unable to get current env: unable to retrieve the current Git branch name")
	}
	app := GuessSelectedAppByDirectory(dir, FindLocalApplications(projectRoot))
	if app == nil {
		return nil, errors.New("unable to get current application")
	}
	return &Project{
		ID:  projectID,
		App: app.Name,
		Env: envID,
	}, nil
}

func GetProjectRoot(debug bool) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.WithStack(err)
	}

	if projectRoot := repositoryRootDir(currentDir); projectRoot != "" {
		return projectRoot, nil
	}

	return "", errors.WithStack(ErrProjectRootNotFoundNoGitRemote)
}

func repositoryRootDir(currentDir string) string {
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

func getProjectID(projectRoot string, debug bool) string {
	contents, err := os.ReadFile(filepath.Join(projectRoot, ".platform", "local", "project.yaml"))
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "WARNING: unable to find Platform.sh config file: %s\n", err)
		}
		return ""
	}
	var config struct {
		ID string `yaml:"id"`
	}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "ERROR: unable to decode Platform.sh config file: %s\n", err)
		}
		return ""
	}
	return config.ID
}
