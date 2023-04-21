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

// most of the code from https://github.com/mholt/caddy

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type linkResource struct {
	uri    string
	params map[string]string
}

// servePreloadLinks parses Link headers from backend and pushes resources found in them.
// If resource has 'nopush' attribute then it will be omitted.
func (s *Server) servePreloadLinks(w http.ResponseWriter, r *http.Request) ([]string, error) {
	resources, exists := w.Header()["Link"]
	if !exists {
		return nil, nil
	}
	// check if this is a request for the pushed resource (avoid recursion)
	if _, exists := r.Header["X-Push"]; exists {
		return nil, nil
	}
	pusher, hasPusher := w.(http.Pusher)
	// no push possible, carry on
	if !hasPusher {
		return nil, nil
	}
	headers := filterProxiedHeaders(r.Header)
	rs := []string{}
	for _, resource := range resources {
		for _, resource := range parseLinkHeader(resource) {
			if _, exists := resource.params["nopush"]; exists {
				continue
			}
			if isRemoteResource(resource.uri) {
				continue
			}
			if err := errors.WithStack(pusher.Push(resource.uri, &http.PushOptions{
				Method: http.MethodGet,
				Header: headers,
			})); err != nil {
				return nil, errors.WithStack(err)
			}
			rs = append(rs, resource.uri)
		}
	}
	return rs, nil
}

func isRemoteResource(resource string) bool {
	return strings.HasPrefix(resource, "//") ||
		strings.HasPrefix(resource, "http://") ||
		strings.HasPrefix(resource, "https://")
}

func filterProxiedHeaders(headers http.Header) http.Header {
	filter := http.Header{}
	for _, header := range []string{
		"Accept-Encoding",
		"Accept-Language",
		"Cache-Control",
		"Host",
		"User-Agent",
	} {
		if val, ok := headers[header]; ok {
			filter[header] = val
		}
	}
	return filter
}

// parseLinkHeader is responsible for parsing Link header and returning list of found resources.
//
// Accepted formats are:
// Link: </resource>; as=script
// Link: </resource>; as=script,</resource2>; as=style
// Link: </resource>;</resource2>
func parseLinkHeader(header string) []linkResource {
	resources := []linkResource{}

	if header == "" {
		return resources
	}

	for _, link := range strings.Split(header, ",") {
		l := linkResource{params: make(map[string]string)}

		li, ri := strings.Index(link, "<"), strings.Index(link, ">")

		if li == -1 || ri == -1 {
			continue
		}

		l.uri = strings.TrimSpace(link[li+1 : ri])

		for _, param := range strings.Split(strings.TrimSpace(link[ri+1:]), ";") {
			parts := strings.SplitN(strings.TrimSpace(param), "=", 2)
			key := strings.TrimSpace(parts[0])

			if key == "" {
				continue
			}

			if len(parts) == 1 {
				l.params[key] = key
			}

			if len(parts) == 2 {
				l.params[key] = strings.TrimSpace(parts[1])
			}
		}

		resources = append(resources, l)
	}

	return resources
}
