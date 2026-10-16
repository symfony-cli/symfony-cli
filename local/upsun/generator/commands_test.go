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

package main

import (
	"slices"
	"testing"
)

func TestCommandAliases(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		namespace string
		visible   []string
		hidden    []string
		want      []string
	}{
		{
			name:      "namespaced command with visible and hidden aliases",
			command:   "environment:logs",
			namespace: "cloud:environment",
			visible:   []string{"log"},
			hidden:    []string{"logs"},
			want: []string{
				`{Name: "environment:logs", Hidden: true}`,
				`{Name: "upsun:environment:logs", Hidden: true}`,
				`{Name: "cloud:log"}`,
				`{Name: "upsun:log", Hidden: true}`,
				`{Name: "log", Hidden: true}`,
				`{Name: "cloud:logs", Hidden: true}`,
				`{Name: "upsun:logs", Hidden: true}`,
			},
		},
		{
			name:      "global command does not get unprefixed aliases",
			command:   "clear-cache",
			namespace: "cloud",
			visible:   []string{"cc"},
			hidden:    []string{"clearcache"},
			want: []string{
				`{Name: "upsun:clear-cache", Hidden: true}`,
				`{Name: "cloud:cc"}`,
				`{Name: "upsun:cc", Hidden: true}`,
				`{Name: "cloud:clearcache", Hidden: true}`,
				`{Name: "upsun:clearcache", Hidden: true}`,
			},
		},
		{
			name:      "deploy alias is skipped",
			command:   "environment:push",
			namespace: "cloud:environment",
			visible:   []string{"push", "deploy"},
			want: []string{
				`{Name: "environment:push", Hidden: true}`,
				`{Name: "upsun:environment:push", Hidden: true}`,
				`{Name: "cloud:push"}`,
				`{Name: "upsun:push", Hidden: true}`,
				`{Name: "push", Hidden: true}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandAliases(tt.command, tt.namespace, tt.visible, tt.hidden)
			if !slices.Equal(got, tt.want) {
				t.Errorf("commandAliases() =\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}
