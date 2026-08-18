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
	"net/http"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type PHPFPMSuite struct{}

var _ = Suite(&PHPFPMSuite{})

func (s *PHPFPMSuite) TestGenerateEnv(c *C) {
	testdataDir := "testdata"
	tests := []struct {
		uri      string
		passthru string
		expected map[string]string
	}{
		{
			passthru: "/index.php",
			uri:      "/",
			expected: map[string]string{
				"PATH_INFO":       "/",
				"REQUEST_URI":     "/",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/?foo=bar",
			expected: map[string]string{
				"PATH_INFO":       "/",
				"REQUEST_URI":     "/?foo=bar",
				"QUERY_STRING":    "foo=bar",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/index.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/index.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/index.php/foo",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/index.php/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/app.PHP",
			uri:      "/app.PHP/foo",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/app.PHP/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/app.PHP",
				"SCRIPT_NAME":     "/app.PHP",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/index.php/foo?foo=bar",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/index.php/foo?foo=bar",
				"QUERY_STRING":    "foo=bar",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/foo",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/update.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/update.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/update.php",
				"SCRIPT_NAME":     "/update.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/js/whitelist.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/js/whitelist.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/js/whitelist.php",
				"SCRIPT_NAME":     "/js/whitelist.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/update.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/update.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/update.php",
				"SCRIPT_NAME":     "/update.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/unknown.php",
			expected: map[string]string{
				"PATH_INFO":       "/unknown.php",
				"REQUEST_URI":     "/unknown.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/unknown.php/foo",
			expected: map[string]string{
				"PATH_INFO":       "/unknown.php/foo",
				"REQUEST_URI":     "/unknown.php/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/unknown.PHP/foo",
			expected: map[string]string{
				"PATH_INFO":       "/unknown.PHP/foo",
				"REQUEST_URI":     "/unknown.PHP/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/subdirectory",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/",
			expected: map[string]string{
				"PATH_INFO":       "/",
				"REQUEST_URI":     "/subdirectory/",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/unknown.php",
			expected: map[string]string{
				"PATH_INFO":       "/unknown.php",
				"REQUEST_URI":     "/subdirectory/unknown.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/unknown.php/",
			expected: map[string]string{
				"PATH_INFO":       "/unknown.php/",
				"REQUEST_URI":     "/subdirectory/unknown.php/",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/index.php/foo",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/subdirectory/index.php/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/subdirectory/",
			expected: map[string]string{
				"PATH_INFO":       "/",
				"REQUEST_URI":     "/subdirectory/subdirectory/",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "///subdirectory",
			expected: map[string]string{
				"PATH_INFO":       "///subdirectory",
				"REQUEST_URI":     "///subdirectory",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory///subdirectory//foo/",
			expected: map[string]string{
				"PATH_INFO":       "/subdirectory/foo/",
				"REQUEST_URI":     "/subdirectory/subdirectory/foo/",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/../index.php",
			expected: map[string]string{
				"PATH_INFO":       "/../index.php",
				"REQUEST_URI":     "/../index.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/../../index.php",
			expected: map[string]string{
				"PATH_INFO":       "/../../index.php",
				"REQUEST_URI":     "/subdirectory/../../index.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/subdirectory/subdirectory/foo/subdirectory/bar",
			expected: map[string]string{
				"PATH_INFO":       "/foo/subdirectory/bar",
				"REQUEST_URI":     "/subdirectory/subdirectory/foo/subdirectory/bar",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/subdirectory/subdirectory/index.php",
				"SCRIPT_NAME":     "/subdirectory/subdirectory/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/foo/../update.php",
			expected: map[string]string{
				"PATH_INFO":       "/foo/../update.php",
				"REQUEST_URI":     "/foo/../update.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
	}
	for _, test := range tests {
		process := &Server{
			projectDir:   testdataDir,
			documentRoot: testdataDir + "/public/",
			passthru:     test.passthru,
		}
		req, err := http.NewRequest("GET", test.uri, nil)
		c.Assert(err, IsNil)

		req.RequestURI = test.uri
		env := process.generateEnv(req)
		for k, v := range test.expected {
			vv, ok := env[k]
			c.Assert(ok, Equals, true)
			c.Assert(vv, DeepEquals, v, Commentf("#test uri:\"%s\" varName:\"%s\"", test.uri, k))
		}
	}
}
