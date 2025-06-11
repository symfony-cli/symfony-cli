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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/symfony-cli/phpstore"
	. "gopkg.in/check.v1"
)

type ToolbarSuite struct{}

var _ = Suite(&ToolbarSuite{})

func (s *ToolbarSuite) TestToolbarTweakPre73(c *C) {
	testToolbarTweak(c, "pre-7.3.html")
}

func (s *ToolbarSuite) TestToolbarTweakPost73(c *C) {
	testToolbarTweak(c, "post-7.3.html")
}

func testToolbarTweak(c *C, filename string) {
	localServer := &Server{
		Version: &phpstore.Version{
			Version: "8.4.0",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.OpenFile("testdata/toolbar/"+filename, os.O_RDONLY, 0644)
		if err != nil {
			c.Fatal(err)
		}

		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		io.Copy(w, file)
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req = req.WithContext(context.WithValue(req.Context(), environmentContextKey, map[string]string{}))
	if err != nil {
		c.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Fatal(err)
	}

	err, processed := localServer.processToolbarInResponse(res)
	c.Assert(err, IsNil)
	c.Assert(processed, Equals, true)

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		c.Fatal(err)
	}
	res.Body.Close()

	c.Assert(string(responseBody), Matches, `(\n|.)*<!-- START of Symfony CLI Toolbar -->(\n|.)*`)
}
