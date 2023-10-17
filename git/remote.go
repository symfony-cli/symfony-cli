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

package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func Fetch(cwd, remote, branch string) error {
	args := []string{"fetch", remote}

	if branch != "" {
		args = append(args, branch)
	}

	_, err := execGitQuiet(cwd, args...)

	return err
}

func Clone(url, dir string) error {
	args := []string{"clone", url, dir}

	return execGit(filepath.Dir(dir), args...)
}

func Push(cwd, remote, ref, remoteRef string) error {
	if ref == "" {
		return errors.New("ref is required when pushing")
	}

	if remoteRef != "" {
		ref = fmt.Sprintf("%s:%s", ref, remoteRef)
	}

	args := []string{"push", "--progress", remote, ref}

	return execGit(cwd, args...)
}

func GetUpstreamBranch(cwd string, remoteNames ...string) string {
	args := []string{"rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"}

	out, err := execGitQuiet(cwd, args...)
	if err != nil {
		return ""
	}

	upstream := strings.Trim(out.String(), " \n")
	if !strings.Contains(upstream, "/") {
		return ""
	}

	remote := strings.SplitN(upstream, "/", 2)
	for _, remoteName := range remoteNames {
		if remote[0] == remoteName {
			return remote[1]
		}
	}

	return ""
}

func GetRemoteURL(cwd, remote string) string {
	args := []string{"config", "--get", fmt.Sprintf("remote.%s.url", remote)}

	out, err := execGitQuiet(cwd, args...)
	if err != nil {
		return ""
	}

	return strings.Trim(out.String(), " \n")
}
