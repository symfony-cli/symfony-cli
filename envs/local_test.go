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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"context"

	"github.com/docker/compose/v2/pkg/api"
	docker "github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
	"github.com/symfony-cli/symfony-cli/local/platformsh"

	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
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

func (s *LocalSuite) TestGenericContainer(c *C) {
	os.Setenv("COMPOSE_PROJECT_NAME", "generic")
	identifier := tc.StackIdentifier("generic")
	compose, err := tc.NewDockerComposeWith(tc.WithStackFiles("./testdata/local_project/generic_compose.yaml"), identifier)
	require.NoError(c, err, "NewDockerComposeAPIWith()")

	ctx := context.Background()

	require.NoError(c, compose.Up(ctx, tc.WithRecreate(api.RecreateNever), tc.Wait(true)), "compose.Up()")

	compose.WaitForService("generic", wait.ForListeningPort("3000/tcp"))

	host := os.Getenv(docker.EnvOverrideHost)
	if host == "" || strings.HasPrefix(host, "unix://") {
		host = "127.0.0.1"
	} else {
		u, err := url.Parse(host)
		if err != nil {
			host = "127.0.0.1"
		} else {
			host = u.Hostname()
		}
	}

	container, err := compose.ServiceContainer(ctx, "generic")
	if err != nil {
		c.Errorf("Could not get service with name %s", "generic")
		c.FailNow()
	}

	mappedPort, err := container.MappedPort(ctx, "3000/tcp")

	if err != nil {
		c.Errorf("Could not get mapped port of container: %s", err)
		c.FailNow()
	}

	c.Assert(extractRelationshipsEnvs(&Local{}), DeepEquals, Envs{
		"GENERIC_HOST":   host,
		"GENERIC_IP":     host,
		"GENERIC_PORT":   mappedPort.Port(),
		"GENERIC_URL":    fmt.Sprintf("tcp://127.0.0.1:%s", mappedPort.Port()),
		"GENERIC_SCHEME": "tcp",
	})

	defer func() {
		require.NoError(c, compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
	}()
}

func (s *LocalSuite) TestGenericContainerWithServicePrefix(c *C) {
	os.Setenv("COMPOSE_PROJECT_NAME", "genericcustom")
	identifier := tc.StackIdentifier("genericcustom")
	compose, err := tc.NewDockerComposeWith(tc.WithStackFiles("./testdata/local_project/generic_service_compose.yaml"), identifier)
	require.NoError(c, err, "NewDockerComposeAPIWith()")

	ctx := context.Background()

	require.NoError(c, compose.Up(ctx, tc.WithRecreate(api.RecreateNever), tc.Wait(true)), "compose.Up()")

	compose.WaitForService("generic", wait.ForListeningPort("3000/tcp"))

	host := os.Getenv(docker.EnvOverrideHost)
	if host == "" || strings.HasPrefix(host, "unix://") {
		host = "127.0.0.1"
	} else {
		u, err := url.Parse(host)
		if err != nil {
			host = "127.0.0.1"
		} else {
			host = u.Hostname()
		}
	}

	container, err := compose.ServiceContainer(ctx, "generic")
	if err != nil {
		c.Errorf("Could not get service with name %s", "generic")
		c.FailNow()
	}

	mappedPort, err := container.MappedPort(ctx, "3000/tcp")

	if err != nil {
		c.Errorf("Could not get mapped port of container: %s", err)
		c.FailNow()
	}

	c.Assert(extractRelationshipsEnvs(&Local{}), DeepEquals, Envs{
		"CUSTOM_HOST":   host,
		"CUSTOM_IP":     host,
		"CUSTOM_PORT":   mappedPort.Port(),
		"CUSTOM_URL":    fmt.Sprintf("tcp://127.0.0.1:%s", mappedPort.Port()),
		"CUSTOM_SCHEME": "tcp",
	})

	defer func() {
		require.NoError(c, compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
	}()
}

func (s *LocalSuite) TestRedisContainer(c *C) {

	os.Setenv("COMPOSE_PROJECT_NAME", "redis")
	identifier := tc.StackIdentifier("redis")
	compose, err := tc.NewDockerComposeWith(tc.WithStackFiles("./testdata/local_project/redis_compose.yaml"), identifier)
	require.NoError(c, err, "NewDockerComposeAPIWith()")

	ctx := context.Background()

	require.NoError(c, compose.Up(ctx, tc.WithRecreate(api.RecreateNever), tc.Wait(true)), "compose.Up()")

	compose.WaitForService("redis", wait.ForListeningPort("6379/tcp"))

	host := os.Getenv(docker.EnvOverrideHost)
	if host == "" || strings.HasPrefix(host, "unix://") {
		host = "127.0.0.1"
	} else {
		u, err := url.Parse(host)
		if err != nil {
			host = "127.0.0.1"
		} else {
			host = u.Hostname()
		}
	}

	container, err := compose.ServiceContainer(ctx, "redis")
	if err != nil {
		c.Errorf("Could not get service with name %s", "redis")
		c.FailNow()
	}

	mappedPort, err := container.MappedPort(ctx, "6379/tcp")

	if err != nil {
		c.Errorf("Could not get mapped port of container: %s", err)
		c.FailNow()
	}

	c.Assert(extractRelationshipsEnvs(&Local{}), DeepEquals, Envs{
		"REDIS_HOST":   host,
		"REDIS_PORT":   mappedPort.Port(),
		"REDIS_URL":    fmt.Sprintf("redis://127.0.0.1:%s", mappedPort.Port()),
		"REDIS_SCHEME": "redis",
	})

	defer func() {
		require.NoError(c, compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
	}()
}

func (s *LocalSuite) TestDatabaseContainer(c *C) {
	os.Setenv("COMPOSE_PROJECT_NAME", "postgres")
	identifier := tc.StackIdentifier("postgres")
	compose, err := tc.NewDockerComposeWith(tc.WithStackFiles("./testdata/local_project/postgres_compose.yaml"), identifier)
	require.NoError(c, err, "NewDockerComposeAPIWith()")

	ctx := context.Background()

	require.NoError(c, compose.Up(ctx, tc.WithRecreate(api.RecreateNever), tc.Wait(true)), "compose.Up()")

	compose.WaitForService("database", wait.ForListeningPort("5432/tcp"))

	host := os.Getenv(docker.EnvOverrideHost)
	if host == "" || strings.HasPrefix(host, "unix://") {
		host = "127.0.0.1"
	} else {
		u, err := url.Parse(host)
		if err != nil {
			host = "127.0.0.1"
		} else {
			host = u.Hostname()
		}
	}

	container, err := compose.ServiceContainer(ctx, "database")
	if err != nil {
		c.Errorf("Could not get service with name %s", "database")
		c.FailNow()
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")

	if err != nil {
		c.Errorf("Could not get mapped port of container: %s", err)
		c.FailNow()
	}

	c.Assert(extractRelationshipsEnvs(&Local{}), DeepEquals, Envs{
		"PGHOST":            host,
		"PGUSER":            "app",
		"PGDATABASE":        "app",
		"PGPASSWORD":        "password",
		"PGPORT":            mappedPort.Port(),
		"DATABASE_DRIVER":   "postgres",
		"DATABASE_VERSION":  "16.3",
		"DATABASE_USERNAME": "app",
		"DATABASE_USER":     "app",
		"DATABASE_NAME":     "app",
		"DATABASE_DATABASE": "app",
		"DATABASE_PASSWORD": "password",
		"DATABASE_SERVER":   fmt.Sprintf("postgres://127.0.0.1:%s", mappedPort.Port()),
		"DATABASE_HOST":     host,
		"DATABASE_PORT":     mappedPort.Port(),
		"DATABASE_URL":      fmt.Sprintf("postgres://%s:%s@127.0.0.1:%s/%s%s", "app", "password", mappedPort.Port(), "app", "?sslmode=disable&charset=utf8&serverVersion=16.3"),
	})

	defer func() {
		require.NoError(c, compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
	}()
}
