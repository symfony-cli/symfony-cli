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
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestMercureProxy(t *testing.T) {
	type hubRequest struct{ host, uri, forwardedHost, forwardedProto string }
	hubRequests := make(chan hubRequest, 10)
	done := make(chan struct{})
	defer close(done)
	hubHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hubRequests <- hubRequest{r.Host, r.URL.RequestURI(), r.Header.Get("X-Forwarded-Host"), r.Header.Get("X-Forwarded-Proto")}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: hello\n\n")
		w.(http.Flusher).Flush()
		// keep the stream open to make sure events are not buffered
		select {
		case <-done:
		case <-r.Context().Done():
		}
	})

	for name, newHub := range map[string]func(http.Handler) *httptest.Server{
		"http":  httptest.NewServer,
		"https": httptest.NewTLSServer,
	} {
		t.Run(name, func(t *testing.T) {
			hub := newHub(hubHandler)
			defer hub.Close()
			hubURL, _ := url.Parse(hub.URL)

			port := startServerWithMercureProxy(t, func() (string, error) {
				return hub.URL + MercureHubPath, nil
			})

			for _, uri := range []string{
				"/.well-known/mercure?match=foo",
				"/.well-known/mercure/subscriptions",
				"/.well-known/oauth-protected-resource/.well-known/mercure",
			} {
				res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, uri))
				if err != nil {
					t.Fatal(err)
				}
				line := make(chan string, 1)
				go func() {
					l, _ := bufio.NewReader(res.Body).ReadString('\n')
					line <- l
				}()
				select {
				case l := <-line:
					if l != "data: hello\n" {
						t.Errorf("%s: got %q, want the hub event", uri, l)
					}
				case <-time.After(5 * time.Second):
					t.Fatalf("%s: the hub event was not streamed", uri)
				}
				res.Body.Close()

				want := hubRequest{hubURL.Host, uri, fmt.Sprintf("127.0.0.1:%d", port), "http"}
				if got := <-hubRequests; got != want {
					t.Errorf("hub got %+v, want %+v", got, want)
				}
			}

			res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/.well-known/mercure-not", port))
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if string(body) != "app" {
				t.Errorf("got %q, want the request to be served by the app", body)
			}
		})
	}
}

func TestMercureProxyBehindTLSTerminator(t *testing.T) {
	forwardedProto := make(chan string, 1)
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedProto <- r.Header.Get("X-Forwarded-Proto")
	}))
	defer hub.Close()

	port := startServerWithMercureProxy(t, func() (string, error) {
		return hub.URL + MercureHubPath, nil
	})

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port, MercureHubPath), nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	if got := <-forwardedProto; got != "https" {
		t.Errorf("hub got X-Forwarded-Proto %q, want the scheme of MERCURE_PUBLIC_URL (https)", got)
	}
}

func TestMercureProxyWithoutHub(t *testing.T) {
	port := startServerWithMercureProxy(t, func() (string, error) {
		return "", nil
	})

	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/.well-known/mercure", port))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusBadGateway || !strings.Contains(string(body), "No Mercure hub detected") {
		t.Errorf("got %d %q, want a 502 explaining that no hub was detected", res.StatusCode, body)
	}
}

func startServerWithMercureProxy(t *testing.T, hubURL MercureHubURLFunc) int {
	t.Helper()
	s := &Server{
		DocumentRoot: t.TempDir(),
		ListenIp:     "127.0.0.1",
		Logger:       zerolog.Nop(),
		UseGzip:      true,
		MercureProxy: NewMercureProxy(hubURL, zerolog.Nop()),
		Callback: func(w http.ResponseWriter, r *http.Request, env map[string]string) error {
			fmt.Fprint(w, "app")
			return nil
		},
	}
	port, err := s.Start(make(chan error, 1))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.httpserver.Close()
	})

	return port
}
