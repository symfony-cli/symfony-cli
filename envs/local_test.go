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
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	. "gopkg.in/check.v1"
)

type LocalSuite struct{}

var _ = Suite(&LocalSuite{})

func (s *LocalSuite) TestExtra(c *C) {
	l := &Local{}
	c.Assert(l.Extra(), DeepEquals, Envs{
		"SYMFONY_TUNNEL":       "",
		"SYMFONY_TUNNEL_ENV":   "",
		"SYMFONY_TUNNEL_BRAND": "",
		"SYMFONY_DOCKER_ENV":   "",
	})

	l = &Local{
		Dir: "testdata/upsun",
	}
	c.Assert(l.Extra(), DeepEquals, Envs{
		"SYMFONY_TUNNEL":       "",
		"SYMFONY_TUNNEL_ENV":   "",
		"SYMFONY_TUNNEL_BRAND": "Upsun",
		"SYMFONY_DOCKER_ENV":   "",
	})
}

func (s *LocalSuite) TestTunnelFilePath(c *C) {
	l := &Local{Dir: "testdata/project"}
	os.Rename("testdata/project/git", "testdata/project/.git")
	defer func() {
		os.Rename("testdata/project/.git", "testdata/project/git")
	}()
	project, err := platformsh.ProjectFromDir(l.Dir, true)
	if err != nil {
		panic(err)
	}
	tunnel := Tunnel{Project: project}
	c.Assert(filepath.Base(tunnel.path()), Equals, "ism4mega7wpx6-toto--security-expose.json")
}

func (s *LocalSuite) TestRelationships(c *C) {
	os.Rename("testdata/project/git", "testdata/project/.git")
	defer os.Rename("testdata/project/.git", "testdata/project/git")
	homedir.Reset()
	os.Setenv("HOME", "testdata/project")
	defer homedir.Reset()
	l := &Local{Dir: "testdata/project"}
	c.Assert(extractRelationshipsEnvs(l), DeepEquals, Envs{
		"SECURITY_SERVER_HOST":   "127.0.0.1",
		"SECURITY_SERVER_URL":    "http://127.0.0.1:30000",
		"SECURITY_SERVER_SERVER": "http://127.0.0.1:30000",
		"SECURITY_SERVER_PORT":   "30000",
		"SECURITY_SERVER_SCHEME": "http",
		"SECURITY_SERVER_IP":     "127.0.0.1",
		"DATABASE_URL":           "postgres://main:main@127.0.0.1:30001/main?sslmode=disable&charset=utf8&serverVersion=13",
		"DATABASE_HOST":          "127.0.0.1",
		"DATABASE_PORT":          "30001",
		"DATABASE_USER":          "main",
		"DATABASE_USERNAME":      "main",
		"DATABASE_PASSWORD":      "main",
		"DATABASE_SERVER":        "postgres://127.0.0.1:30001",
		"DATABASE_DRIVER":        "postgres",
		"DATABASE_NAME":          "main",
		"DATABASE_DATABASE":      "main",
		"DATABASE_VERSION":       "13",
		"PGPORT":                 "30001",
		"PGPASSWORD":             "main",
		"PGDATABASE":             "main",
		"PGUSER":                 "main",
		"PGHOST":                 "127.0.0.1",
	})
}

func (s *LocalSuite) TestProjectDirGuessingMissingGitAndConfig(c *C) {
	l, err := NewLocal("testdata/project", false)
	expectedLocalDir, err := filepath.Abs(".")
	expectedLocalDir = filepath.Dir(expectedLocalDir)
	c.Assert(err, IsNil)
	c.Assert(l.Dir, Equals, expectedLocalDir)
}

func (s *LocalSuite) TestGitProjectDirGuessing(c *C) {
	os.Rename("testdata/project/git", "testdata/project/.git")
	defer os.Rename("testdata/project/.git", "testdata/project/git")
	homedir.Reset()
	os.Setenv("HOME", "testdata/project")
	defer homedir.Reset()

	l, err := NewLocal("testdata/project", false)

	expectedLocalDir, err := filepath.Abs("testdata/project")
	c.Assert(err, IsNil)
	c.Assert(l.Dir, Equals, expectedLocalDir)
}

func (s *LocalSuite) TestConfigProjectDirGuessing(c *C) {
	configFilePath := "testdata/project/.symfony.local.yaml"
	os.WriteFile(configFilePath, make([]byte, 0), 0644)
	defer os.Remove(configFilePath)
	homedir.Reset()
	os.Setenv("HOME", "testdata/project")
	defer homedir.Reset()

	l, err := NewLocal("testdata/project", false)

	expectedLocalDir, err := filepath.Abs("testdata/project")
	c.Assert(err, IsNil)
	c.Assert(l.Dir, Equals, expectedLocalDir)
}
