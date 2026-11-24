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
	"os"
	"path/filepath"
	"testing"
)

var validurlcases = []string{
	"http://symfony.com",
	"https://symfony.com",
	"https://symfony.com/blog",
	"https://symfony.com/blog?foo=bar",
	"https://foo:bar@symfony.com/blog?foo=bar",
}

var invalidurlcases = []string{
	"symfony.com",
	"/Users/tucksaun/Work/src/github.com/symfony-cli/symfony-cli/cloud/init-templates/00-flex.yaml",
	"c:\\windows\test.yaml",
}

func TestIsValidUrl(t *testing.T) {
	for _, test := range validurlcases {
		if !isValidURL(test) {
			t.Errorf("isValidUrl(%q): got false, expected true", test)
		}
	}
	for _, test := range invalidurlcases {
		if isValidURL(test) {
			t.Errorf("isValidUrl(%q): got true, expected false", test)
		}
	}
}

var validfilecases = []string{
	"init_templating.go",
}

var invalidfilecases = []string{
	"../cmd",
	"foo.go",
	"http://symfony.com",
	"https://symfony.com",
	"https://symfony.com/blog",
	"https://symfony.com/blog?foo=bar",
	"https://foo:bar@symfony.com/blog?foo=bar",
}

func TestIsValidFilePath(t *testing.T) {
	for _, test := range validfilecases {
		if !isValidFilePath(test) {
			t.Errorf("isValidFilePath(%q): got false, expected true", test)
		}
	}
	for _, test := range invalidfilecases {
		if isValidFilePath(test) {
			t.Errorf("isValidFilePath(%q): got true, expected false", test)
		}
	}
}

func TestHasComposerPackage(t *testing.T) {
	for pkg, expected := range map[string]bool{
		"foo/bar":         false,
		"symfony/symfony": false,
	} {
		result := hasComposerPackage(filepath.Join("tests", "composer_packages", "none"), pkg)

		if result != expected {
			t.Errorf("hasComposerPackage(none/%q): got %v, expected %v", pkg, result, expected)
		}
	}

	packages := map[string]bool{
		"foo/bar":        false,
		"symfony/flex":   true,
		"symfony/dotenv": true,
	}

	for _, testCase := range []string{"lock", "json", "both"} {
		for pkg, expected := range packages {
			result := hasComposerPackage(filepath.Join("tests", "composer_packages", testCase), pkg)

			if result != expected {
				t.Errorf("hasComposerPackage(%q/%q): got %v, expected %v", testCase, pkg, result, expected)
			}
		}
	}
}

func TestHasComposerPackageMinimumVersion(t *testing.T) {
	for _, test := range []struct {
		version string
		matches bool
	}{
		{"v6.4.0", false},
		{"v7.3.9", false},
		{"v7.4.0-RC1", false},
		{"v7.4.0", true},
		{"7.4.1", true},
		{"v8.0.0", true},
		{"7.4.x-dev", false},
		{"dev-main", false},
		{"", false},
	} {
		t.Run(test.version, func(t *testing.T) {
			dir := t.TempDir()
			lock := `{"packages":[{"name":"symfony/framework-bundle","version":"` + test.version + `"}]}`
			if err := os.WriteFile(filepath.Join(dir, "composer.lock"), []byte(lock), 0644); err != nil {
				t.Fatal(err)
			}
			if got := hasComposerPackage(dir, "symfony/framework-bundle:>=7.4"); got != test.matches {
				t.Errorf("version %q: got %v, want %v", test.version, got, test.matches)
			}
		})
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require":{"symfony/framework-bundle":"^7.4"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if hasComposerPackage(dir, "symfony/framework-bundle:>=7.4") {
		t.Fatal("a version check requires a locked version")
	}
	if err := os.WriteFile(filepath.Join(dir, "composer.lock"), []byte(`{"packages-dev":[{"name":"symfony/framework-bundle","version":"v7.4.0"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if !hasComposerPackage(dir, "symfony/framework-bundle:>=7.4") {
		t.Fatal("development packages also have locked versions")
	}
	for _, pkg := range []string{"symfony/missing:>=7.4", "symfony/framework-bundle:>=unknown", "symfony/framework-bundle:>="} {
		if hasComposerPackage(dir, pkg) {
			t.Errorf("unexpected match for %q", pkg)
		}
	}
}

func TestHasPHPExtension(t *testing.T) {
	for pkg, expected := range map[string]bool{
		"iconv": false,
		"pdo":   false,
	} {
		result := hasPHPExtension(filepath.Join("tests", "composer_packages", "none"), pkg)

		if result != expected {
			t.Errorf("hasPHPExtension(none/%q): got %v, expected %v", pkg, result, expected)
		}
	}

	exts := map[string]bool{
		"pdo":       false,
		"iconv":     true,
		"ext-iconv": true,
	}

	for _, testCase := range []string{"lock", "json", "both"} {
		for ext, expected := range exts {
			result := hasPHPExtension(filepath.Join("tests", "composer_packages", testCase), ext)

			if result != expected {
				t.Errorf("hasPHPExtension(%q/%q): got %v, expected %v", testCase, ext, result, expected)
			}
		}
	}
}
