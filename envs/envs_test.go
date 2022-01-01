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

package envs

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ScenvSuite struct{}

var _ = Suite(&ScenvSuite{})

func (s *ScenvSuite) TestAppID(c *C) {
	c.Assert(appID("testdata/project"), Equals, "01B7C15FQN1DFPQ4H2CJ66YFPM")
	c.Assert(appID("testdata/project_without_id"), Equals, "")
	c.Assert(appID("testdata/project_without_composer"), Equals, "")
	c.Assert(appID("testdata/project_with_borked_composer"), Equals, "")
}
