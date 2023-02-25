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
	"path/filepath"
	"strings"

	"github.com/symfony-cli/symfony-cli/envs"
)

func (p *Server) resolveScriptName(pathInfo string) (string, string) {
	if pos := strings.Index(strings.ToLower(pathInfo), ".php"); pos != -1 {
		file := pathInfo[:pos+4]
		if file == filepath.Clean(file) {
			if _, err := os.Stat(filepath.Join(p.documentRoot, file)); err == nil {
				return file, pathInfo[pos+4:]
			}
		}
	}
	// quick return if it's short or path starts with //
	if len(pathInfo) <= 1 || pathInfo[0:2] == "//" {
		return p.passthru, pathInfo
	}

	// removes first slash to make sure we don't loop through it as it always need to be there.
	paths := strings.Split(pathInfo[1:], "/")

	for n := len(paths); n > 0; n-- {
		pathPart := paths[n-1]
		if pathPart == "" {
			continue
		}

		// we on purpose don't use filepath join as it resolves the paths. This way if clean filepath is different we break
		folder := string(filepath.Separator) + strings.Join(paths[:n], string(filepath.Separator))

		if folder != filepath.Clean(folder) {
			continue
		}

		file := filepath.Join(folder, p.passthru)
		path := strings.Join(paths[n:], "/")

		if _, err := os.Stat(filepath.Join(p.documentRoot, file)); err == nil {
			// I am not sure how we can get rid of this if statements. It's complete abomination, but it's because subdirectory and subdirectory/ should go to this same file, but have different pathinfo
			if path == "" && pathInfo[len(pathInfo)-1:] != "/" {
				return file, ""
			}
			return file, "/" + path
		}

	}

	return p.passthru, pathInfo
}

func (p *Server) generateEnv(req *http.Request) map[string]string {
	scriptName, pathInfo := p.resolveScriptName(req.URL.Path)

	//fmt.Println(req.URL.Path + " | " + scriptName + " | " + pathInfo + " | " + filepath.Clean(scriptName))

	https := ""
	if req.TLS != nil {
		https = "On"
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
