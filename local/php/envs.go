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

package php

import (
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/symfony-cli/symfony-cli/envs"
)

func (p *Server) generateEnv(req *http.Request) map[string]string {
	scriptName := p.passthru
	https := ""
	if req.TLS != nil {
		https = "On"
	}

	pathInfo := req.URL.Path
	if pos := strings.Index(strings.ToLower(pathInfo), ".php"); pos != -1 {
		file := path.Clean(pathInfo[:pos+4])
		if _, err := os.Stat(filepath.Join(p.documentRoot, file)); err == nil {
			scriptName = file
			pathInfo = pathInfo[pos+4:]
		}
	}

	remoteAddr := req.Header.Get("X-Client-IP")
	remotePort := ""
	if remoteAddr == "" {
		remoteAddr, remotePort, _ = net.SplitHostPort(req.RemoteAddr)
	}

	env := map[string]string{
		"CONTENT_LENGTH":    req.Header.Get("Content-Length"),
		"CONTENT_TYPE":      req.Header.Get("Content-Type"),
		"DOCUMENT_URI":      scriptName,
		"DOCUMENT_ROOT":     p.documentRoot,
		"GATEWAY_INTERFACE": "CGI/1.1",
		"HTTP_HOST":         req.Host,
		"HTTP_MOD_REWRITE":  "On", // because Pagekit relies on it
		"HTTPS":             https,
		"PATH_INFO":         pathInfo,
		"QUERY_STRING":      req.URL.RawQuery,
		"REDIRECT_STATUS":   "200", // required if PHP was built with --enable-force-cgi-redirect
		"REMOTE_ADDR":       remoteAddr,
		"REMOTE_PORT":       remotePort,
		"REQUEST_METHOD":    req.Method,
		"REQUEST_URI":       req.RequestURI,
		"SCRIPT_FILENAME":   filepath.Join(p.documentRoot, scriptName),
		"SCRIPT_NAME":       scriptName,
	}

	if local, err := envs.NewLocal(p.projectDir, false); err == nil {
		for k, v := range envs.AsMap(local) {
			env[k] = v
		}
	}

	// iterate over request headers and append them to the environment variables in the valid format
	for k, v := range req.Header {
		key := strings.Replace(strings.ToUpper(k), "-", "_", -1)
		// ignore HTTP_HOST -- see https://httpoxy.org/
		if key == "HOST" {
			continue
		}
		env["HTTP_"+key] = strings.Join(v, ";")
	}
	return env
}
