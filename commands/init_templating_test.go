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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/symfony-cli/symfony-cli/local/upsun"
)

func TestCreateRequiredFilesProject(t *testing.T) {
	projectDir := "./testdata/project"
	slug := "slug"
	services := []*CloudService{
		{
			Name:    "foo",
			Type:    "bar",
			Version: "baz",
		},
		{
			Name:    "foo1",
			Type:    "bar1",
			Version: "baz1",
		},
		{
			Name:    "foo2",
			Type:    "postgresql",
			Version: "baz2",
		},
	}
	for _, service := range services {
		service.SetEndpoint()
	}

	if _, err := createRequiredFilesProject(upsun.Fixed, projectDir, slug, "", "8.0", services, false, true); err != nil {
		panic(err)
	}

	path := filepath.Join(projectDir, ".platform", "services.yaml")
	result, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	expected := `
foo:
    type: bar:baz

foo1:
    type: bar1:baz1

foo2:
    type: postgresql:baz2
    disk: 1024
`
	result = bytes.TrimSpace(result)
	expected = strings.TrimSpace(expected)
	if string(result) != expected {
		t.Errorf("platform/services.yaml: got %v, expected %v", string(result), expected)
	}
}

func TestCreateRequiredFilesProjectForUpsun(t *testing.T) {
	projectDir := "./testdata/project"
	slug := "slug"
	services := []*CloudService{
		{
			Name:    "foo",
			Type:    "bar",
			Version: "baz",
		},
		{
			Name:    "foo1",
			Type:    "bar1",
			Version: "baz1",
		},
		{
			Name:    "foo2",
			Type:    "postgresql",
			Version: "baz2",
		},
	}
	for _, service := range services {
		service.SetEndpoint()
	}

	if _, err := createRequiredFilesProject(upsun.Flex, projectDir, slug, "", "8.0", services, false, true); err != nil {
		panic(err)
	}

	path := filepath.Join(projectDir, ".upsun", "config.yaml")
	result, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	expected := `
services:
	foo:
		type: bar:baz

	foo1:
		type: bar1:baz1

	foo2:
		type: postgresql:baz2
		disk: 1024
`
	result = bytes.TrimSpace(result)
	expected = strings.TrimSpace(expected)
	if strings.Contains(string(result), expected) {
		t.Errorf("upsun/config.yaml: got %v, expected %v", string(result), expected)
	}
}
