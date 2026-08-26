/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
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
	"io"
	"os"
	"slices"
	"testing"

	"github.com/symfony-cli/console"
)

func TestLspCheckPassesEveryArgumentThrough(t *testing.T) {
	tests := map[string][]string{
		"options paths and delimiter": {"--format=json", "src", "--", "-literal"},
		"checker help":                {"--help"},
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			originalArguments := os.Args
			os.Args = append([]string{"symfony", "lsp:check"}, expected...)
			defer func() { os.Args = originalArguments }()

			var arguments []string
			command := newLspCheckCommand(func(received []string) (int, error) {
				arguments = slices.Clone(received)

				return 0, nil
			})
			app := &console.Application{
				Name:      "symfony",
				Commands:  []*console.Command{command},
				Writer:    io.Discard,
				ErrWriter: io.Discard,
			}

			if err := app.Run(os.Args); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(arguments, expected) {
				t.Fatalf("unexpected delegated arguments %#v", arguments)
			}
		})
	}
}
