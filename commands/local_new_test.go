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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/symfony-cli/symfony-cli/local/upsun"
)

type fakeUpsunRegistry map[string]string

func (r fakeUpsunRegistry) ServiceVersion(serviceType, wanted string) (string, bool, error) {
	return r[serviceType], wanted != "" && r[serviceType] == wanted, nil
}

func (r fakeUpsunRegistry) IsPHPExtensionAvailable(ext, phpVersion string) (bool, error) {
	return true, nil
}

type failingUpsunRegistry struct{}

var errRegistryUnavailable = errors.New("registry unavailable")

func (failingUpsunRegistry) ServiceVersion(serviceType, wanted string) (string, bool, error) {
	return "", false, errRegistryUnavailable
}

func TestParseDockerComposeServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"postgresql": {"service": true, "versions": {
			"9.6": {"upsun": {"status": "retired"}},
			"10": {"upsun": {"status": "retired"}},
			"14": {"upsun": {"status": "supported"}},
			"18": {"upsun": {"status": "supported"}}
		}}}`)
	}))
	defer server.Close()
	registry := upsun.NewMetaRegistry("dev")
	registry.BaseURL = server.URL
	t.Setenv("POSTGRES_NEXT_VERSION", "19")

	for dir, expected := range map[string]string{
		"testdata/docker/postgresql/noversion/": "18",
		"testdata/docker/postgresql/10/":        "14",
		"testdata/docker/postgresql/9/":         "14",
		"testdata/docker/postgresql/next/":      "18",
	} {
		result, err := parseDockerComposeServices(registry, dir)
		if err != nil {
			t.Fatal(err)
		}
		if result[0].Name != "database" || result[0].Type != "postgresql" || result[0].Version != expected {
			t.Errorf("parseDockerComposeServices(%q): got %s:%s:%s, expected database:postgresql:%s", dir, result[0].Name, result[0].Type, result[0].Version, expected)
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
