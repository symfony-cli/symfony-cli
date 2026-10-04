//go:build darwin || linux || freebsd || openbsd

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
	"slices"
	"testing"

	"github.com/posener/complete"
)

func TestBuildSymfonyConsoleWrapperArgsShell(t *testing.T) {
	for name, tc := range map[string]struct {
		compShell string
		shell     string
		expected  []string
	}{
		"completion script shell":            {compShell: "fish", shell: "/bin/zsh", expected: []string{"-a1", "-sfish"}},
		"SHELL fallback":                     {shell: "/usr/bin/zsh", expected: []string{"-a1", "-szsh"}},
		"no shell detected omits the option": {expected: []string{"-a1"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("COMP_SHELL", tc.compShell)
			t.Setenv("SHELL", tc.shell)
			t.Setenv("CURRENT", "2")

			args := buildSymfonyConsoleWrapperArgs("console", complete.Args{All: []string{"console", ""}})
			if got := args[len(args)-len(tc.expected):]; !slices.Equal(got, tc.expected) {
				t.Errorf("buildSymfonyConsoleWrapperArgs() = %v, want suffix %v", args, tc.expected)
			}
		})
	}
}
