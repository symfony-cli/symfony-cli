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
	"embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/posener/complete"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/local/php"
	"github.com/symfony-cli/terminal"
)

// completionTemplates holds our custom shell completions templates.
//
//go:embed resources/completion.*
var completionTemplates embed.FS

func init() {
	// override console completion templates with our custom ones
	console.CompletionTemplates = completionTemplates
}

func autocompleteApplicationConsoleWrapper(context *console.Context, words complete.Args) []string {
	return autocompleteSymfonyConsoleWrapper(words, "console", func(args []string) (*php.Executor, error) {
		return php.SymfonyConsoleExecutor(terminal.Logger, args)
	})
}

func autocompletePieWrapper(context *console.Context, words complete.Args) []string {
	return autocompleteSymfonyConsoleWrapper(words, "pie", func(args []string) (*php.Executor, error) {
		return php.PieExecutor("", args, []string{}, context.App.Writer, context.App.ErrWriter, io.Discard, terminal.Logger)
	})
}

// autocompleteComposerWrapper is a bridge between Go autocompletion and
// Composer one. It does not use the generic Symfony wrapper because Composer
// can not support multiple shells yet
func autocompleteComposerWrapper(context *console.Context, words complete.Args) []string {
	args := buildSymfonyConsoleAutocompleteArgs("composer", words)
	// Composer does not support multiple shell yet, so we only use the default one
	args = append(args, "-sbash")

	res := php.Composer("", args, []string{}, context.App.Writer, context.App.ErrWriter, io.Discard, terminal.Logger)
	os.Exit(res.ExitCode())

	// unreachable
	return []string{}
}

// autocompleteSymfonyConsoleWrapper bridges the symfony-cli/console (Go)
// autocompletion with a symfony/console (PHP) one.
func autocompleteSymfonyConsoleWrapper(words complete.Args, commandName string, executor func(args []string) (*php.Executor, error)) []string {
	args := buildSymfonyConsoleAutocompleteArgs(commandName, words)
	// Composer does not support those options yet, so we only use them for Symfony Console
	args = append(args, "-a1", fmt.Sprintf("-s%s", console.GuessShell()))

	if executor, err := executor(args); err == nil {
		os.Exit(executor.Execute(false))
	}

	return []string{}
}

func buildSymfonyConsoleAutocompleteArgs(wrappedCommand string, words complete.Args) []string {
	current, err := strconv.Atoi(os.Getenv("CURRENT"))
	if err != nil {
		current = 1
	} else {
		// we decrease one position corresponding to `symfony` command
		current -= 1
	}

	args := make([]string, 0, len(words.All))
	// build the inputs command line that Symfony expects
	for _, input := range words.All {
		if input = strings.TrimSpace(input); input != "" {

			// remove quotes from typed values
			quote := input[0]
			if quote == '\'' || quote == '"' {
				input = strings.TrimPrefix(input, string(quote))
				input = strings.TrimSuffix(input, string(quote))
			}

			args = append(args, fmt.Sprintf("-i%s", input))
		}
	}

	return append([]string{
		"_complete", "--no-interaction",
		fmt.Sprintf("-c%d", current),
		fmt.Sprintf("-i%s", wrappedCommand),
	}, args...)
}
