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
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/symfony-cli/local/html"
)

// MercureHubPath is the path of the Mercure hub
const MercureHubPath = "/.well-known/mercure"

// MercureHubURLFunc returns the URL of the Mercure hub, or an empty string when there is none
type MercureHubURLFunc func() (string, error)

type mercureProxy struct {
	hubURL    MercureHubURLFunc
	transport http.RoundTripper
	logger    zerolog.Logger
}

// NewMercureProxy forwards Mercure hub requests to the hub returned by hubURL
// so that browsers can reach it on the web server origin
func NewMercureProxy(hubURL MercureHubURLFunc, logger zerolog.Logger) http.Handler {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// hubs running in local containers use certificates from their own CA
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &mercureProxy{hubURL: hubURL, transport: transport, logger: logger}
}

func (p *mercureProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hubURL, err := p.hubURL()
	if err == nil && hubURL == "" {
		err = errors.New("make sure its Docker container is running")
	}
	var target *url.URL
	if err == nil {
		target, err = url.Parse(hubURL)
	}
	if err != nil {
		msg := "No Mercure hub detected: " + err.Error()
		p.logger.Error().Msg(msg)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(html.WrapHTML(msg, html.CreateErrorTerminal("# "+msg), "")))
		return
	}

	proxy := &httputil.ReverseProxy{
		Transport: p.transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			// SetURL also sets the Host to the hub one, as Mercure only serves
			// its site address and derives the token audience from it
			r.SetURL(&url.URL{Scheme: target.Scheme, Host: target.Host})
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.logger.Error().Err(err).Msg("unable to proxy the request to the Mercure hub")
			w.WriteHeader(http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(w, r)
}

func isMercureHubPath(path string) bool {
	return path == MercureHubPath ||
		strings.HasPrefix(path, MercureHubPath+"/") ||
		path == "/.well-known/oauth-protected-resource"+MercureHubPath
}

func withMercureProxy(h, mercure http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isMercureHubPath(r.URL.Path) {
			mercure.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}
