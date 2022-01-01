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
