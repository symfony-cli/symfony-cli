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

//go:generate sh generate_docker_version

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	compose "github.com/compose-spec/compose-go/cli"
	composeConsts "github.com/compose-spec/compose-go/consts"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/symfony-cli/terminal"
	"gopkg.in/yaml.v2"
)

var (
	dockerComposeNormalizeRegexp       = regexp.MustCompile("[^-_a-z0-9]")
	dockerComposeNormalizeRegexpLegacy = regexp.MustCompile("[^a-z0-9]")
	dockerUserAgent                    = "Docker-Client/unknown version"
)

func ComputeDockerUserAgent(appName, appVersion string) {
	dockerUserAgent = fmt.Sprintf("Docker-Client/%s %s/%s", dockerClientVersion, appName, appVersion)
}

type sortedPorts []container.Port

func (ps sortedPorts) Len() int           { return len(ps) }
func (ps sortedPorts) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps sortedPorts) Less(i, j int) bool { return ps[i].PrivatePort < ps[j].PrivatePort }

// Port of https://github.com/docker/compose/blob/615c01c50a51408a7fdfed66ecccf73781e87f2c/compose/cli/command.py#L153-L154
func normalizeDockerComposeProjectName(projectName string) string {
	return dockerComposeNormalizeRegexp.ReplaceAllString(strings.ToLower(projectName), "")
}

// Port of https://github.com/docker/compose/blob/4e0fdd70bdae4f8d85e4ef9d0129ce445f3ece3c/compose/cli/command.py#L129-L130
// (prior to 615c01c50a51408a7fdfed66ecccf73781e87f2c)
// This was used in Docker Compose prior to 1.21.0, some users might still use
// versions older though, so we keep this BC in the meantime.
func normalizeDockerComposeProjectNameLegacy(projectName string) string {
	return dockerComposeNormalizeRegexpLegacy.ReplaceAllString(strings.ToLower(projectName), "")
}

func (l *Local) RelationshipsFromDocker() Relationships {
	project := l.getComposeProjectName()
	if project == "" {
		return nil
	}

	client, err := docker.NewClientWithOpts(
		docker.FromEnv,
		dockerUseDesktopSocketIfAvailable,
		docker.WithAPIVersionNegotiation(),
		// we use a short timeout here because we don't want to impact
		// negatively performance when Docker is not reachable
		docker.WithTimeout(2*time.Second),
		// defining a User Agent to avoid having the Docker API being slow
		// see https://github.com/docker/for-mac/issues/7575
		docker.WithUserAgent(dockerUserAgent),
	)
	if err != nil {
		if l.Debug {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		}
		return nil
	}
	defer client.Close()

	containers, err := client.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		if docker.IsErrConnectionFailed(err) {
			terminal.Logger.Warn().Msg(err.Error())
		}
		if l.Debug {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		}
		return nil
	}

	// To be in sync with Docker compose behavior we also normalize project name
	// passed by the environment variable
	projectLegacy := normalizeDockerComposeProjectNameLegacy(project)
	project = normalizeDockerComposeProjectName(project)

	relationships := Relationships{}
	for _, container := range containers {
		p, ok := container.Labels["com.docker.compose.project"]
		if !ok {
			continue
		}
		if p != project && p != projectLegacy {
			continue
		}
		for suffix, relationship := range l.dockerServiceToRelationship(client, container) {
			// get the service name
			name, ok := container.Labels["com.symfony.server.service-prefix"]
			if !ok {
				name, ok = container.Labels["com.docker.compose.service"]
				if !ok {
					if l.Debug {
						fmt.Fprintf(os.Stderr, "ERROR: unable to find the service name of the Docker container with id \"%s\"\n", container.ID)
					}
					continue
				}
			}
			if l.Debug {
				fmt.Fprintf(os.Stderr, "  exposing service \"%s%s\"\n", name, suffix)
			}
			relationships[name+suffix] = append(relationships[name+suffix], relationship)
		}
	}

	if len(relationships) != 0 {
		l.DockerEnv = true
	}

	return relationships
}

func (l *Local) dockerServiceToRelationship(client *docker.Client, container container.Summary) map[string]map[string]interface{} {
	if l.Debug {
		fmt.Fprintf(os.Stderr, `found Docker container "%s" for project "%s" (image "%s")`+"\n", container.Labels["com.docker.compose.service"], container.Labels["com.docker.compose.project"], container.Image)
	}

	if v, _ := strconv.ParseBool(container.Labels["com.symfony.server.service-ignore"]); v {
		if l.Debug {
			fmt.Fprintln(os.Stderr, "  skipping as com.symfony.server.service-ignore is true")
		}
		return nil
	}

	var exposedPorts sortedPorts
	for _, port := range container.Ports {
		if port.PublicPort == 0 {
			continue
		}
		if l.Debug {
			fmt.Fprintf(os.Stderr, "  port %d for private port %d\n", port.PublicPort, port.PrivatePort)
		}
		exposedPorts = append(exposedPorts, port)
	}
	if len(exposedPorts) == 0 {
		if l.Debug && len(container.Ports) > 0 {
			fmt.Fprintf(os.Stderr, "WARNING: Container %s has none of its ports exposed.\n", container.Names[0][1:])
		}
		return nil
	}

	c, err := client.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		if l.Debug {
			fmt.Fprintf(os.Stderr, "  ERROR: error while inspecting container \"%s\": %s\n", container.Names[0][1:], err)
		}
		return nil
	}

	if l.Debug {
		for _, env := range c.Config.Env {
			fmt.Fprintf(os.Stderr, "  env  %s\n", env)
		}
	}

	host := os.Getenv(docker.EnvOverrideHost)
	if host == "" || strings.HasPrefix(host, "unix://") {
		host = "127.0.0.1"
	} else {
		u, err := url.Parse(host)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: unable to parse %s \"%s\", falling back to 127.0.0.1: %s\n", docker.EnvOverrideHost, host, err)
			host = "127.0.0.1"
		} else {
			host = u.Hostname()
		}
	}

	if len(exposedPorts) < 1 {
		return nil
	}

	sort.Sort(exposedPorts)
	p := exposedPorts[0]

	rels := make(map[string]map[string]interface{})
	if p.PrivatePort == 1025 {
		// recommended image: sj26/mailcatcher or axllent/mailpit (default now)
		for _, pw := range exposedPorts {
			if pw.PrivatePort == 1080 || pw.PrivatePort == 8025 {
				rels["-web"] = map[string]interface{}{
					"host":   host,
					"ip":     host,
					"port":   formatDockerPort(pw.PublicPort),
					"rel":    "mailer",
					"scheme": "http",
				}
				rels[""] = map[string]interface{}{
					"host":   host,
					"ip":     host,
					"port":   formatDockerPort(p.PublicPort),
					"rel":    "mailer",
					"scheme": "smtp",
				}
				return rels
			}
		}
	} else if p.PrivatePort == 25 {
		// recommended image: djfarrelly/maildev
		for _, pw := range exposedPorts {
			if pw.PrivatePort == 80 {
				rels["-web"] = map[string]interface{}{
					"host":   host,
					"ip":     host,
					"port":   formatDockerPort(pw.PublicPort),
					"rel":    "mailer",
					"scheme": "http",
				}
				rels[""] = map[string]interface{}{
					"host":   host,
					"ip":     host,
					"port":   formatDockerPort(p.PublicPort),
					"rel":    "mailer",
					"scheme": "smtp",
				}
				return rels
			}
		}
	} else if p.PrivatePort == 8707 || p.PrivatePort == 8307 {
		// Blackfire
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"rel":    "blackfire",
			"scheme": "tcp",
		}
		return rels
	} else if p.PrivatePort == 3306 {
		username := ""
		password := ""
		path := ""
		version := ""
		// MARIADB is used by bitnami/mariadb
		for _, prefix := range []string{"MYSQL", "MARIADB"} {
			for _, env := range c.Config.Env {
				if strings.HasPrefix(env, prefix+"_ROOT_PASSWORD") && password == "" {
					// *_PASSWORD has precedence over *_ROOT_PASSWORD
					password = getEnvValue(env, prefix+"_ROOT_PASSWORD")
					username = "root"
				} else if strings.HasPrefix(env, prefix+"_USER") {
					username = getEnvValue(env, prefix+"_USER")
				} else if strings.HasPrefix(env, prefix+"_PASSWORD") {
					password = getEnvValue(env, prefix+"_PASSWORD")
				} else if strings.HasPrefix(env, prefix+"_DATABASE") {
					path = getEnvValue(env, prefix+"_DATABASE")
				} else if strings.HasPrefix(env, prefix+"_VERSION") {
					version = getEnvValue(env, prefix+"_VERSION")
				}
			}
		}
		if path == "" {
			path = username
		}
		rels[""] = map[string]interface{}{
			"host":     host,
			"ip":       host,
			"username": username,
			"password": password,
			"path":     path,
			"version":  version,
			"port":     formatDockerPort(p.PublicPort),
			"query": map[string]bool{
				"is_master": true,
			},
			"rel":    "mysql",
			"scheme": "mysql",
		}
		return rels
	} else if p.PrivatePort == 5432 {
		username := ""
		password := ""
		path := ""
		version := ""
		for _, env := range c.Config.Env {
			if strings.HasPrefix(env, "POSTGRES_USER") {
				username = getEnvValue(env, "POSTGRES_USER")
			} else if strings.HasPrefix(env, "POSTGRES_PASSWORD") {
				password = getEnvValue(env, "POSTGRES_PASSWORD")
			} else if strings.HasPrefix(env, "POSTGRES_DB") {
				path = getEnvValue(env, "POSTGRES_DB")
			} else if strings.HasPrefix(env, "PG_VERSION") {
				version = getEnvValue(env, "PG_VERSION")
			}
		}
		if path == "" {
			path = username
		}
		rels[""] = map[string]interface{}{
			"host":     host,
			"ip":       host,
			"username": username,
			"password": password,
			"path":     path,
			"version":  version,
			"port":     formatDockerPort(p.PublicPort),
			"query": map[string]bool{
				"is_master": true,
			},
			"rel":    "pgsql",
			"scheme": "pgsql",
		}
		return rels
	} else if p.PrivatePort == 6379 {
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"rel":    "redis",
			"scheme": "redis",
		}
		return rels
	} else if p.PrivatePort == 11211 {
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"rel":    "memcached",
			"scheme": "memcached",
		}
		return rels
	} else if p.PrivatePort == 5672 {
		username := "guest"
		password := "guest"
		for _, env := range c.Config.Env {
			// that's our local convention
			if strings.HasPrefix(env, "RABBITMQ_DEFAULT_USER") {
				username = getEnvValue(env, "RABBITMQ_DEFAULT_USER")
			} else if strings.HasPrefix(env, "RABBITMQ_DEFAULT_PASS") {
				password = getEnvValue(env, "RABBITMQ_DEFAULT_PASS")
			}
		}
		rels[""] = map[string]interface{}{
			"host":     host,
			"ip":       host,
			"port":     formatDockerPort(p.PublicPort),
			"username": username,
			"password": password,
			"rel":      "amqp",
			"scheme":   "amqp",
		}
		// management plugin?
		for _, pw := range exposedPorts {
			if pw.PrivatePort == 15672 {
				rels["-management"] = map[string]interface{}{
					"host":   host,
					"ip":     host,
					"port":   formatDockerPort(pw.PublicPort),
					"rel":    "amqp",
					"scheme": "http",
				}
				break
			}
		}
		return rels
	} else if p.PrivatePort == 9200 {
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"path":   "/",
			"rel":    "elasticsearch",
			"scheme": "http",
		}
		return rels
	} else if p.PrivatePort == 5601 {
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"path":   "/",
			"rel":    "kibana",
			"scheme": "http",
		}
		return rels
	} else if p.PrivatePort == 27017 || p.PrivatePort == 27018 || p.PrivatePort == 27019 {
		username := ""
		password := ""
		path := ""
		for _, env := range c.Config.Env {
			// that's our local convention
			if strings.HasPrefix(env, "MONGO_DATABASE") {
				path = getEnvValue(env, "MONGO_DATABASE")
			} else if strings.HasPrefix(env, "MONGO_INITDB_DATABASE") {
				path = getEnvValue(env, "MONGO_INITDB_DATABASE")
			} else if strings.HasPrefix(env, "MONGO_INITDB_ROOT_USERNAME") {
				username = getEnvValue(env, "MONGO_INITDB_ROOT_USERNAME")
			} else if strings.HasPrefix(env, "MONGO_INITDB_ROOT_PASSWORD") {
				password = getEnvValue(env, "MONGO_INITDB_ROOT_PASSWORD")
			}
		}
		rels[""] = map[string]interface{}{
			"host":     host,
			"ip":       host,
			"username": username,
			"password": password,
			"path":     path,
			"port":     formatDockerPort(p.PublicPort),
			"rel":      "mongodb",
			"scheme":   "mongodb",
		}
		return rels
	} else if p.PrivatePort == 9092 {
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"rel":    "kafka",
			"scheme": "kafka",
		}
		return rels
	} else if p.PrivatePort == 80 && strings.Contains(container.Image, "dunglas/mercure") {
		// for podman the image name is docker.io/dunglas/mercure:latest
		rels[""] = map[string]interface{}{
			"host":   host,
			"ip":     host,
			"port":   formatDockerPort(p.PublicPort),
			"rel":    "mercure",
			"scheme": "http",
		}
		return rels
	}

	if l.Debug {
		fmt.Fprintln(os.Stderr, "  exposing port")
	}

	rels[""] = map[string]interface{}{
		"host": host,
		"ip":   host,
		"port": formatDockerPort(p.PublicPort),
		"rel":  "simple",
	}
	// Official HTTP(s) ports or well know alternatives
	switch p.PrivatePort {
	case 80, 8008, 8080, 8081:
		rels[""]["scheme"] = "http"
	case 443, 8443:
		rels[""]["scheme"] = "https"
	default:
		rels[""]["scheme"] = "tcp"
	}
	return rels
}

func formatDockerPort(port uint16) string {
	return strconv.FormatInt(int64(port), 10)
}

func dockerUseDesktopSocketIfAvailable(c *docker.Client) error {
	if c.DaemonHost() != docker.DefaultDockerHost {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	socketPath := filepath.Join(homeDir, ".docker/run/docker.sock")
	if _, err := os.Stat(socketPath); err != nil {
		return nil
	}

	return docker.WithHost(`unix://` + socketPath)(c)
}

func getEnvValue(env string, key string) string {
	if len(key) == len(env) {
		return ""
	}

	return env[len(key)+1:]
}

func (l *Local) getComposeProjectName() string {
	// https://docs.docker.com/compose/reference/envvars/#compose_project_name
	if project := os.Getenv(composeConsts.ComposeProjectName); project != "" {
		return project
	}

	composeDir := l.getComposeDir()
	if composeDir == "" {
		if l.Debug {
			fmt.Fprintln(os.Stderr, "ERROR: unable to find a compose directory for the current directory")
		}
		return ""
	}

	// COMPOSE_PROJECT_NAME can be set in a .env file
	if _, err := os.Stat(filepath.Join(composeDir, ".env")); err == nil {
		if contents, err := os.ReadFile(filepath.Join(composeDir, ".env")); err == nil {
			for _, line := range bytes.Split(contents, []byte("\n")) {
				if bytes.HasPrefix(line, []byte(composeConsts.ComposeProjectName+"=")) {
					return string(line[len(composeConsts.ComposeProjectName)+1:])
				}
			}
		}
	}

	// Compose project name can be set in every Docker Compose file
	for index, filename := range compose.DefaultFileNames {
		if _, err := os.Stat(filepath.Join(composeDir, filename)); err != nil {
			continue
		}

		for _, filename := range []string{compose.DefaultOverrideFileNames[index], filename} {
			buf, err := os.ReadFile(filepath.Join(composeDir, filename))
			if err != nil {
				continue
			}

			config := struct {
				ProjectName string `yaml:"name"`
			}{}

			// unmarshall the content of the file to get the project name
			if err := yaml.Unmarshal(buf, &config); err == nil && config.ProjectName != "" {
				return config.ProjectName
			}
		}
	}

	return filepath.Base(composeDir)
}

func (l *Local) getComposeDir() string {
	// https://docs.docker.com/compose/reference/envvars/#compose_file
	if os.Getenv("COMPOSE_FILE") != "" {
		return l.Dir
	}

	// look for the first dir up with a docker-composer.ya?ml file (in case of a multi-project)
	dir := l.Dir
	for {
		for _, filename := range compose.DefaultFileNames {
			if _, err := os.Stat(filepath.Join(dir, filename)); err == nil {
				return dir
			}
		}
		upDir := filepath.Dir(dir)
		if upDir == dir || upDir == "." {
			if l.Debug {
				fmt.Fprintln(os.Stderr, "ERROR: unable to find a docker-compose.ya?ml or compose.ya?ml for the current directory")
			}
			return ""
		}
		dir = upDir
	}
}
