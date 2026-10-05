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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const metaImagesFixture = `{
	"postgresql": {
		"service": true,
		"versions": {
			"9.6": {"upsun": {"status": "retired", "internal_status": "sunset"}},
			"14": {"upsun": {"status": "supported", "internal_status": "active"}},
			"18": {"upsun": {"status": "supported", "internal_status": "active"}},
			"19": {"upsun": {"status": "incoming", "internal_status": "active"}}
		}
	},
	"opensearch": {
		"service": true,
		"versions": {
			"1": {"upsun": {"status": "retired", "internal_status": "sunset"}},
			"2": {"upsun": {"status": "deprecated", "internal_status": "active"}},
			"3": {"upsun": {"status": "supported", "internal_status": "active"}}
		}
	},
	"legacy": {
		"service": true,
		"versions": {
			"1.9": {"upsun": {"status": "deprecated", "internal_status": "active"}},
			"1.10": {"upsun": {"status": "deprecated", "internal_status": "active"}},
			"2.0": {"upsun": {"status": "retired", "internal_status": "sunset"}}
		}
	},
	"gone": {
		"service": true,
		"versions": {
			"1.0": {"upsun": {"status": "decommissioned", "internal_status": "decommissioned"}}
		}
	},
	"php": {
		"service": false,
		"versions": {
			"8.5": {"upsun": {"status": "supported", "internal_status": "active"}}
		}
	}
}`

const metaPHPExtensionsFixture = `{
	"Redis": {
		"description": "Redis client.",
		"versions": {
			"8.4": {"status": "available", "options": []},
			"8.3": {"status": "default", "options": []},
			"8.2": {"status": "deprecated", "options": []},
			"5.6": {"status": "unavailable", "options": []}
		}
	},
	"opcache": {
		"description": "Opcode cache.",
		"versions": {
			"8.4": {"status": "built-in", "options": []}
		}
	}
}`

func newTestMetaRegistry(t *testing.T) (*MetaRegistry, map[string]int) {
	t.Helper()
	hits := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("unexpected Accept header %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "symfony-cli/1.2.3" {
			t.Errorf("unexpected User-Agent header %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/images":
			_, _ = io.WriteString(w, metaImagesFixture)
		case "/extensions/php/cloud":
			_, _ = io.WriteString(w, metaPHPExtensionsFixture)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	registry := NewMetaRegistry("1.2.3")
	registry.BaseURL = server.URL
	return registry, hits
}

func TestMetaRegistryServiceLastVersion(t *testing.T) {
	registry, hits := newTestMetaRegistry(t)
	for serviceType, expected := range map[string]string{
		"postgresql": "18",
		"opensearch": "3",
		"legacy":     "1.10",
		"gone":       "",
		"php":        "",
		"unknown":    "",
	} {
		t.Run(serviceType, func(t *testing.T) {
			got, err := registry.ServiceLastVersion(serviceType)
			if err != nil {
				t.Fatal(err)
			}
			if got != expected {
				t.Errorf("got %q, expected %q", got, expected)
			}
		})
	}
	if hits["/images"] != 1 {
		t.Errorf("expected images to be fetched once, got %d", hits["/images"])
	}
}

func TestMetaRegistryIsPHPExtensionAvailable(t *testing.T) {
	registry, hits := newTestMetaRegistry(t)
	for _, tc := range []struct {
		ext, phpVersion string
		expected        bool
	}{
		{"redis", "8.4", true},
		{"REDIS", "8.3", true},
		{"redis", "8.2", true},
		{"redis", "5.6", false},
		{"redis", "7.0", false},
		{"opcache", "8.4", false},
		{"unknown", "8.4", false},
	} {
		t.Run(tc.ext+"@"+tc.phpVersion, func(t *testing.T) {
			got, err := registry.IsPHPExtensionAvailable(tc.ext, tc.phpVersion)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.expected {
				t.Errorf("got %v, expected %v", got, tc.expected)
			}
		})
	}
	if hits["/extensions/php/cloud"] != 1 {
		t.Errorf("expected extensions to be fetched once, got %d", hits["/extensions/php/cloud"])
	}
}

func TestMetaRegistryFailsOnErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	registry := NewMetaRegistry("1.2.3")
	registry.BaseURL = server.URL

	if _, err := registry.ServiceLastVersion("postgresql"); err == nil {
		t.Error("expected an error when fetching services fails")
	}
	if _, err := registry.IsPHPExtensionAvailable("redis", "8.4"); err == nil {
		t.Error("expected an error when fetching PHP extensions fails")
	}
}
