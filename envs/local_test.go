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

	docker "github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
	"github.com/symfony-cli/symfony-cli/local/platformsh"

	"github.com/testcontainers/testcontainers-go"
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
	ctx := context.Background()

	composeProject := os.Getenv("COMPOSE_PROJECT_NAME")
	defer os.Setenv("COMPOSE_PROJECT_NAME", composeProject)

	os.Setenv("COMPOSE_PROJECT_NAME", "TestGenericContainer")

	cmd := []string{"--providers.docker=false", "--entryPoints.web.address=:3000"}
	labels := testcontainers.GenericLabels()
	labels["com.docker.compose.project"] = "testgenericcontainer"
	labels["com.docker.compose.service"] = "generic"
	req := testcontainers.ContainerRequest{
		Image:        "traefik:v3.1",
		ExposedPorts: []string{"3000/tcp"},
		WaitingFor:   wait.ForListeningPort("3000/tcp"),
		Labels:       labels,
		Cmd:          cmd,
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		c.Errorf("Could not start container: %s", err)
		c.FailNow()
	}

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
		if err := container.Terminate(ctx); err != nil {
			c.Errorf("Could not stop redis: %s", err)
			c.FailNow()
		}
	}()
}

func (s *LocalSuite) TestGenericContainerWithServicePrefix(c *C) {
	ctx := context.Background()

	composeProject := os.Getenv("COMPOSE_PROJECT_NAME")
	defer os.Setenv("COMPOSE_PROJECT_NAME", composeProject)

	os.Setenv("COMPOSE_PROJECT_NAME", "TestGenericContainer")

	cmd := []string{"--providers.docker=false", "--entryPoints.web.address=:3000"}
	labels := testcontainers.GenericLabels()
	labels["com.docker.compose.project"] = "testgenericcontainer"
	labels["com.symfony.server.service-prefix"] = "CUSTOM"
	req := testcontainers.ContainerRequest{
		Image:        "traefik:v3.1",
		ExposedPorts: []string{"3000/tcp"},
		WaitingFor:   wait.ForListeningPort("3000/tcp"),
		Labels:       labels,
		Cmd:          cmd,
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		c.Errorf("Could not start container: %s", err)
		c.FailNow()
	}

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
		if err := container.Terminate(ctx); err != nil {
			c.Errorf("Could not stop redis: %s", err)
			c.FailNow()
		}
	}()
}

func (s *LocalSuite) TestRedisContainer(c *C) {
	ctx := context.Background()

	composeProject := os.Getenv("COMPOSE_PROJECT_NAME")
	defer os.Setenv("COMPOSE_PROJECT_NAME", composeProject)

	os.Setenv("COMPOSE_PROJECT_NAME", "TestRedisContainer")

	labels := testcontainers.GenericLabels()
	labels["com.docker.compose.project"] = "testrediscontainer"
	labels["com.docker.compose.service"] = "redis"
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
		Labels:       labels,
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		c.Errorf("Could not start container: %s", err)
		c.FailNow()
	}

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
		if err := container.Terminate(ctx); err != nil {
			c.Errorf("Could not stop redis: %s", err)
			c.FailNow()
		}
	}()
}

func (s *LocalSuite) TestDatabaseContainer(c *C) {
	ctx := context.Background()

	composeProject := os.Getenv("COMPOSE_PROJECT_NAME")
	defer os.Setenv("COMPOSE_PROJECT_NAME", composeProject)

	os.Setenv("COMPOSE_PROJECT_NAME", "TestDatabaseContainer")

	labels := testcontainers.GenericLabels()
	labels["com.docker.compose.project"] = "testdatabasecontainer"
	labels["com.docker.compose.service"] = "database"
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16.3-alpine",
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
		Labels:       labels,
		Env: map[string]string{
			"POSTGRES_USER":     "app",
			"POSTGRES_PASSWORD": "password",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		c.Errorf("Could not start container: %s", err)
		c.FailNow()
	}

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
		if err := container.Terminate(ctx); err != nil {
			c.Errorf("Could not stop redis: %s", err)
			c.FailNow()
		}
	}()
}
