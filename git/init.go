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

import "bytes"

func Init(dir string, forceMainGitBranchName, debug bool) (*bytes.Buffer, error) {
	if content, err := doExecGit(dir, []string{"init"}, !debug); err != nil {
		return content, err
	}
	if forceMainGitBranchName {
		return doExecGit(dir, []string{"checkout", "-b", "main"}, !debug)
	}
	return nil, nil
}

func AddAndCommit(dir string, files []string, msg string, debug bool) (*bytes.Buffer, error) {
	cmds := [][]string{
		append([]string{"add"}, files...),
		{"commit", "-m", msg},
	}
	for _, cmd := range cmds {
		if content, err := doExecGit(dir, cmd, !debug); err != nil {
			return content, err
		}
	}
	return nil, nil
}
