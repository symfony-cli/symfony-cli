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
	. "gopkg.in/check.v1"
)

type DockerSuite struct{}

var _ = Suite(&DockerSuite{})

func (s *DockerSuite) TestNormalizeDockerComposeProjectName(c *C) {
	for _, testCase := range []struct {
		ProjectName, Expected, ExpectedLegacy string
	}{
		{"foo", "foo", "foo"},
		{"simple-composefile", "simple-composefile", "simplecomposefile"},
		{"multiple-compose-files", "multiple-compose-files", "multiplecomposefiles"},
		{"MyProject", "myproject", "myproject"},
		{"MyProject2", "myproject2", "myproject2"},
		{"symfony.com", "symfonycom", "symfonycom"},
		{"oss-websites", "oss-websites", "osswebsites"},
		{"symfony-dev", "symfony-dev", "symfonydev"},
	} {
		c.Check(normalizeDockerComposeProjectName(testCase.ProjectName), Equals, testCase.Expected)
		c.Check(normalizeDockerComposeProjectNameLegacy(testCase.ProjectName), Equals, testCase.ExpectedLegacy)
	}
}
