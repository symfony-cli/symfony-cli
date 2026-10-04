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
	"net"
	"net/http"

	"github.com/pkg/errors"
)

// withServerHeader sets the Server response header unless the wrapped handler already set one.
func withServerHeader(h http.Handler, value string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(&serverHeaderWriter{ResponseWriter: w, value: value}, r)
	})
}

type serverHeaderWriter struct {
	http.ResponseWriter
	value string
	done  bool
}

func (w *serverHeaderWriter) setServerHeader() {
	if w.done {
		return
	}
	w.done = true
	if w.Header().Get("Server") == "" {
		w.Header().Set("Server", w.value)
	}
}

func (w *serverHeaderWriter) WriteHeader(code int) {
	w.setServerHeader()
	w.ResponseWriter.WriteHeader(code)
}

func (w *serverHeaderWriter) Write(b []byte) (int, error) {
	w.setServerHeader()
	n, err := w.ResponseWriter.Write(b)
	return n, errors.WithStack(err)
}

func (w *serverHeaderWriter) Flush() {
	w.setServerHeader()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *serverHeaderWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	return conn, rw, errors.WithStack(err)
}

func (w *serverHeaderWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return errors.WithStack(p.Push(target, opts))
	}
	return http.ErrNotSupported
}

func (w *serverHeaderWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
