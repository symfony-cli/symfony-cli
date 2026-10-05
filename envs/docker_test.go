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
	"sort"

	"github.com/docker/docker/api/types/container"
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

func (s *DockerSuite) TestSortedPortsPreferTCP(c *C) {
	ports := sortedPorts{
		{PrivatePort: 443, PublicPort: 30001, Type: "udp"},
		{PrivatePort: 443, PublicPort: 30002, Type: "tcp"},
		{PrivatePort: 80, PublicPort: 30003, Type: "tcp"},
	}
	sort.Sort(ports)

	c.Check(ports[0].PublicPort, Equals, uint16(30003))
	c.Check(ports[1].PublicPort, Equals, uint16(30002))

	_, _, port := mercureEndpoint("localhost", ports)
	c.Check(port.PublicPort, Equals, uint16(30002))
}

func (s *DockerSuite) TestMercureEndpoint(c *C) {
	ports := func(privatePorts ...uint16) []container.Port {
		var ps []container.Port
		for _, p := range privatePorts {
			ps = append(ps, container.Port{PrivatePort: p, PublicPort: 30000 + p})
		}
		return ps
	}

	for _, tc := range []struct {
		serverName       string
		ports            []container.Port
		expectedScheme   string
		expectedHostname string
		expectedPort     uint16
	}{
		// default SERVER_NAME (localhost) serves HTTPS on 443
		{"", ports(80, 443), "https", "localhost", 30443},
		// HTTPS port not published, fall back to the first one
		{"", ports(80), "http", "", 30080},
		{":80", ports(80), "http", "", 30080},
		{":9877", ports(9877), "http", "", 39877},
		{"mercure.superproject.localhost:80", ports(80), "http", "mercure.superproject.localhost", 30080},
		{"mercure.localhost", ports(80, 443), "https", "mercure.localhost", 30443},
		{"mercure.localhost:8443", ports(8443), "https", "mercure.localhost", 38443},
		{"http://mercure.localhost:8080", ports(8080), "http", "mercure.localhost", 38080},
		{"https://:8443", ports(8443), "https", "", 38443},
		{"*.example.com:80", ports(80), "http", "", 30080},
		// first address with a published port wins
		{"localhost, :80", ports(80), "http", "", 30080},
		{"localhost :80", ports(80, 443), "https", "localhost", 30443},
		// nothing matches, fall back to HTTP on the first published port
		{"", ports(9877), "http", "", 39877},
	} {
		scheme, hostname, port := mercureEndpoint(tc.serverName, tc.ports)
		comment := Commentf("SERVER_NAME=%q", tc.serverName)
		c.Check(scheme, Equals, tc.expectedScheme, comment)
		c.Check(hostname, Equals, tc.expectedHostname, comment)
		c.Check(port.PublicPort, Equals, tc.expectedPort, comment)
	}
}
