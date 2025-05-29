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

package project

import (
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	. "gopkg.in/check.v1"
)

type ConfigSuite struct{}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) TestDefaultConfig(c *C) {
	config, err := NewConfigFromDirectory(
		zerolog.Nop(),
		"",
		"",
	)
	c.Assert(err, IsNil)
	c.Assert(config, NotNil)
}

func (s *ConfigSuite) TestConfigFromDirectory(c *C) {
	config, err := NewConfigFromDirectory(
		zerolog.Nop(),
		"testdata",
		"testdata",
	)
	c.Assert(err, IsNil)
	c.Assert(config, NotNil)

	c.Assert(config.NoWorkers, Equals, true)
	c.Assert(config.Daemon, Equals, false)

	c.Assert(config.Proxy.Domains, DeepEquals, []string{"foo"})
	c.Assert(config.Proxy.Domains, DeepEquals, []string{"foo"})

	c.Assert(config.HTTP.PreferredPort, Equals, 8181)

	c.Assert(config.Workers, HasLen, 3)
	c.Assert(config.Workers["docker_compose"].Cmd, NotNil)
	c.Assert(config.Workers["docker_compose"].Cmd, Not(Equals), []string{})

	c.Assert(config.Workers["messenger_consume_async"].Cmd, NotNil)
	c.Assert(config.Workers["messenger_consume_async"].Cmd, Not(Equals), []string{})

	c.Assert(config.Workers["my_node_process"].Cmd, DeepEquals, []string{"npx", "foo"})
	c.Assert(config.Workers["my_node_process"].Watch, DeepEquals, []string{".node_version"})
}

func (s *ConfigSuite) TestConfigFromContext(c *C) {
	app := console.Application{
		Flags: ConfigurationFlags,
		Action: func(context *console.Context) error {
			config, err := NewConfigFromContext(
				context,
				zerolog.Nop(),
				"testdata",
				"testdata",
			)
			c.Assert(err, IsNil)
			c.Assert(config, NotNil)

			//c.Assert(config.HTTP.PreferredPort, Equals, 8282)
			c.Assert(config.HTTP.AllowHTTP, Equals, true)

			c.Assert(config.NoWorkers, Equals, true)
			c.Assert(config.Daemon, Equals, false)

			c.Assert(config.Proxy.Domains, DeepEquals, []string{"foo"})
			c.Assert(config.Proxy.Domains, DeepEquals, []string{"foo"})

			c.Assert(config.Workers, HasLen, 3)
			c.Assert(config.Workers["docker_compose"].Cmd, NotNil)
			c.Assert(config.Workers["docker_compose"].Cmd, Not(Equals), []string{})

			c.Assert(config.Workers["messenger_consume_async"].Cmd, NotNil)
			c.Assert(config.Workers["messenger_consume_async"].Cmd, Not(Equals), []string{})

			c.Assert(config.Workers["my_node_process"].Cmd, DeepEquals, []string{"npx", "foo"})
			c.Assert(config.Workers["my_node_process"].Watch, DeepEquals, []string{".node_version"})

			return nil
		},
	}
	c.Check(app.Run([]string{"--port=8282", "--allow-http=true"}), IsNil)
}
