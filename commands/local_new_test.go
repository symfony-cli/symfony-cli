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
	"errors"
	"os"
	"testing"
)

type fakeUpsunRegistry map[string]string

func (r fakeUpsunRegistry) ServiceLastVersion(serviceType string) (string, error) {
	return r[serviceType], nil
}

func (r fakeUpsunRegistry) IsPHPExtensionAvailable(ext, phpVersion string) (bool, error) {
	return true, nil
}

type failingUpsunRegistry struct{}

var errRegistryUnavailable = errors.New("registry unavailable")

func (failingUpsunRegistry) ServiceLastVersion(serviceType string) (string, error) {
	return "", errRegistryUnavailable
}

func TestParseDockerComposeServices(t *testing.T) {
	registry := fakeUpsunRegistry{"postgresql": "18"}
	os.Setenv("POSTGRES_NEXT_VERSION", "19")
	defer os.Unsetenv("POSTGRES_NEXT_VERSION")

	for dir, expected := range map[string]CloudService{
		"testdata/docker/postgresql/noversion/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "18",
		},
		"testdata/docker/postgresql/10/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "10",
		},
		"testdata/docker/postgresql/9/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "9.6",
		},
		"testdata/docker/postgresql/next/": {
			Name:    "database",
			Type:    "postgresql",
			Version: "18",
		},
	} {
		result, err := parseDockerComposeServices(registry, dir)
		if err != nil {
			t.Fatal(err)
		}
		if result[0].Version != expected.Version {
			t.Errorf("parseDockerComposeServices(none/%q): got %v, expected %v", dir, result[0].Version, expected.Version)
		}
	}
}

func TestParseDockerComposeServicesRegistryError(t *testing.T) {
	if _, err := parseDockerComposeServices(failingUpsunRegistry{}, "testdata/docker/postgresql/10/"); !errors.Is(err, errRegistryUnavailable) {
		t.Errorf("expected the registry error, got %v", err)
	}
}

func TestParseCLIServices(t *testing.T) {
	registry := fakeUpsunRegistry{"postgresql": "18", "redis": "8.0"}
	for config, expected := range map[string]CloudService{
		"postgresql":                 {Name: "postgresql", Type: "postgresql", Version: "18"},
		"database:postgresql":        {Name: "database", Type: "postgresql", Version: "18"},
		"database:postgresql:14":     {Name: "database", Type: "postgresql", Version: "14"},
		"cache:redis-persistent":     {Name: "cache", Type: "redis-persistent", Version: "8.0"},
		"cache:redis-persistent:7.2": {Name: "cache", Type: "redis-persistent", Version: "7.2"},
		"unknown":                    {Name: "unknown", Type: "unknown", Version: ""},
	} {
		t.Run(config, func(t *testing.T) {
			result, err := parseCLIServices(registry, []string{config})
			if err != nil {
				t.Fatal(err)
			}
			if result[0].Name != expected.Name || result[0].Type != expected.Type || result[0].Version != expected.Version {
				t.Errorf("got %s:%s:%s, expected %s:%s:%s", result[0].Name, result[0].Type, result[0].Version, expected.Name, expected.Type, expected.Version)
			}
		})
	}
}

func TestParseCLIServicesWithVersionDoesNotQueryRegistry(t *testing.T) {
	if _, err := parseCLIServices(failingUpsunRegistry{}, []string{"database:postgresql:14"}); err != nil {
		t.Fatal(err)
	}
}

func TestParseCLIServicesRegistryError(t *testing.T) {
	if _, err := parseCLIServices(failingUpsunRegistry{}, []string{"database:postgresql"}); !errors.Is(err, errRegistryUnavailable) {
		t.Errorf("expected the registry error, got %v", err)
	}
}
