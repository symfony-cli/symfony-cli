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
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func TestPreloadLinksOverHTTP1AreNotErrors(t *testing.T) {
	for name, useGzip := range map[string]bool{"plain": false, "gzip": true} {
		t.Run(name, func(t *testing.T) {
			logs := &syncBuffer{}
			s := &Server{
				DocumentRoot: t.TempDir(),
				ListenIp:     "127.0.0.1",
				Logger:       zerolog.New(logs),
				UseGzip:      useGzip,
				Callback: func(w http.ResponseWriter, r *http.Request, env map[string]string) error {
					w.Header().Set("Link", "</app.css>; rel=preload; as=style")
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

			res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()

			if strings.Contains(logs.String(), "unable to preload links") {
				t.Errorf("preload links over HTTP/1.1 must not be logged as errors, got %s", logs.String())
			}
		})
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.buf.Write(p)
	return n, errors.WithStack(err)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
