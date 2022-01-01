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

package proxy

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ProxySuite struct{}

var _ = Suite(&ProxySuite{})

func (s *ProxySuite) TestGetDir(c *C) {
	p := &Config{
		domains: map[string]string{
			"symfony.com":        "symfony_com",
			"*.symfony.com":      "any_symfony_com",
			"*.live.symfony.com": "any_live_symfony_com",
		},
	}
	c.Assert(p.GetDir("symfony.com"), Equals, "symfony_com")
	c.Assert(p.GetDir("foo.symfony.com"), Equals, "any_symfony_com")
	c.Assert(p.GetDir("foo.live.symfony.com"), Equals, "any_live_symfony_com")
}
