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

package upsun

import (
	"testing"

	"github.com/symfony-cli/console"
)

func TestUpstreamHiddenAliasesResolve(t *testing.T) {
	app := &console.Application{Commands: Commands}
	for alias, want := range map[string]string{
		"cloud:logs":                 "cloud:environment:logs",
		"upsun:logs":                 "cloud:environment:logs",
		"cloud:snapshot:create":      "cloud:backup:create",
		"cloud:environment:sql":      "cloud:db:sql",
		"cloud:environment:metadata": "cloud:environment:info",
	} {
		cmd := app.Command(alias)
		if cmd == nil {
			t.Errorf("%s: command not found", alias)
			continue
		}
		if got := cmd.FullName(); got != want {
			t.Errorf("%s resolves to %s, want %s", alias, got, want)
		}
	}
	for _, alias := range []string{"logs", "snapshot:create", "environment:sql"} {
		if cmd := app.Command(alias); cmd != nil {
			t.Errorf("%s must not resolve without a prefix, got %s", alias, cmd.FullName())
		}
	}
}
