/*
 * Copyright (c) 2024-present Fabien Potencier <fabien@symfony.com>
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

package upsun

import (
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type PlatformSuite struct{}

var _ = Suite(&PlatformSuite{})

func (s *PlatformSuite) TestReadDBVersionFromDoctrineConfigYAML(c *C) {
	version, err := ReadDBVersionFromDoctrineConfigYAML("testdata/projectA")
	c.Assert(err, IsNil)
	c.Assert(version, Equals, "")
}

func (s *PlatformSuite) TestReadDBVersionFromDoctrineConfigYAMLConnections(c *C) {
	for name, tc := range map[string]struct {
		config   string
		expected string
	}{
		"top-level server version": {
			config:   "doctrine:\n    dbal:\n        server_version: '16'\n",
			expected: "16",
		},
		"implicit default connection": {
			config:   "doctrine:\n    dbal:\n        connections:\n            default:\n                server_version: '15'\n",
			expected: "15",
		},
		"custom default connection": {
			config:   "doctrine:\n    dbal:\n        default_connection: main\n        connections:\n            main:\n                server_version: '10.1'\n            legacy:\n                server_version: '5.7'\n",
			expected: "10.1",
		},
		"custom default connection without version": {
			config:   "doctrine:\n    dbal:\n        server_version: '14'\n        default_connection: main\n        connections:\n            main:\n                url: '%env(DATABASE_URL)%'\n",
			expected: "14",
		},
	} {
		dir := c.MkDir()
		c.Assert(os.MkdirAll(filepath.Join(dir, "config", "packages"), 0755), IsNil)
		c.Assert(os.WriteFile(filepath.Join(dir, "config", "packages", "doctrine.yaml"), []byte(tc.config), 0644), IsNil)

		version, err := ReadDBVersionFromDoctrineConfigYAML(dir)
		c.Assert(err, IsNil, Commentf(name))
		c.Assert(version, Equals, tc.expected, Commentf(name))
	}
}
