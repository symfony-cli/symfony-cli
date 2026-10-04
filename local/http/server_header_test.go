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

package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/rs/zerolog"
)

func TestServerHeader(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/custom" {
			w.Header().Set("Server", "app")
		}
		fmt.Fprint(w, "dynamic")
	}))
	defer backend.Close()
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)

	documentRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(documentRoot, "static.txt"), []byte("static"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		DocumentRoot: documentRoot,
		ListenIp:     "127.0.0.1",
		Appversion:   "1.2.3",
		Logger:       zerolog.Nop(),
		Callback: func(w http.ResponseWriter, r *http.Request, env map[string]string) error {
			reverseProxy.ServeHTTP(w, r)
			return nil
		},
	}
	port, err := s.Start(make(chan error, 1))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path     string
		expected []string
	}{
		{"/static.txt", []string{"symfony-cli/1.2.3"}},
		{"/index.php", []string{"symfony-cli/1.2.3"}},
		{"/custom", []string{"app"}},
	} {
		t.Run(test.path, func(t *testing.T) {
			res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, test.path))
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if got := res.Header.Values("Server"); !slices.Equal(got, test.expected) {
				t.Errorf("Server header = %q, want %q", got, test.expected)
			}
		})
	}
}
