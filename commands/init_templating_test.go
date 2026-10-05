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
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"text/template"

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

	if _, err := createRequiredFilesProject(fakeUpsunRegistry{}, upsun.Fixed, projectDir, slug, "", "8.0", services, false, true); err != nil {
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

	if _, err := createRequiredFilesProject(fakeUpsunRegistry{}, upsun.Flex, projectDir, slug, "", "8.0", services, false, true); err != nil {
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

func TestCloudPHPExtensionsAreUnique(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require": {"ext-redis": "*", "ext-sodium": "*", "ext-zip": "*"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	services := []*CloudService{
		{Name: "cache", Type: "redis"},
		{Name: "sessions", Type: "redis-persistent"},
	}
	for _, service := range services {
		service.SetEndpoint()
	}

	got := cloudPHPExtensions(dir, services)
	expected := []string{"apcu", "blackfire", "mbstring", "redis", "sodium", "xsl", "zip"}
	if !slices.Equal(got, expected) {
		t.Errorf("got %v, expected %v", got, expected)
	}
}

func TestWriteTemplatesDoesNotLeaveTruncatedFilesOnError(t *testing.T) {
	dir := t.TempDir()
	failing := template.Must(template.New("config").Funcs(template.FuncMap{
		"php_extension_available": func(string, string) (bool, error) {
			return false, errors.New("unable to fetch the registry")
		},
	}).Parse("runtime:\n    extensions:\n{{ if php_extension_available \"apcu\" \"8.4\" }}        - apcu\n{{ end }}"))
	templates := map[string]*template.Template{".upsun/config.yaml": failing}

	if _, err := writeTemplates(templates, dir, nil, false, false); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(dir, ".upsun", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected no file to be written, got %v", err)
	}

	templates[".upsun/config.yaml"] = template.Must(template.New("config").Parse("runtime: {}\n"))
	createdFiles, err := writeTemplates(templates, dir, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if expected := []string{filepath.Join(dir, ".upsun", "config.yaml")}; !slices.Equal(createdFiles, expected) {
		t.Errorf("got %v, expected %v", createdFiles, expected)
	}
}
