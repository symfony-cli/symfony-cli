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
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/local/pid"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/symfony-cli/local/proxy"
	"github.com/symfony-cli/symfony-cli/util"
)

// Local represents the local project
type Local struct {
	Dir       string
	Debug     bool
	Tunnel    string
	TunnelEnv bool
	DockerEnv bool
}

// NewLocal creates a new local project
func NewLocal(path string, debug bool) (*Local, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Local{
		Dir:   path,
		Debug: debug,
	}, nil
}

func (l *Local) FindRelationshipPrefix(frel, fscheme string) string {
	for key, allValues := range l.Relationships() {
		key = strings.ToUpper(key)
		for i, endpoint := range allValues {
			if _, ok := endpoint["scheme"]; !ok {
				continue
			}

			scheme := endpoint["scheme"].(string)
			rel := endpoint["rel"].(string)
			if scheme == fscheme && rel == frel {
				prefix := fmt.Sprintf("%s_", key)
				if i != 0 {
					prefix = fmt.Sprintf("%s_%d_", key, i)
				}
				return strings.Replace(prefix, "-", "_", -1)
			}
		}
	}
	return ""
}

// Path returns the project's path
func (l *Local) Path() string {
	return l.Dir
}

// Local returns true if the command is used on a local machine
func (l *Local) Local() bool {
	return true
}

// Relationships returns envs from Platform.sh relationships or a local Docker setup
func (l *Local) Relationships() Relationships {
	// we need to call it in all cases so that l.DockerEnv is set correctly
	dockerRel := l.RelationshipsFromDocker()

	tunnel := Tunnel{Dir: l.Dir, Debug: l.Debug}
	if !tunnel.IsExposed() {
		return dockerRel
	}

	if rels := l.relationshipsFromTunnel(); rels != nil {
		return rels
	}

	return dockerRel
}

// Mail catchers are handled like regular services
func (l *Local) Mailer() Envs {
	return nil
}

// Extra adds some env specific env vars
func (l *Local) Extra() Envs {
	docker := ""
	if l.DockerEnv {
		docker = "1"
	}
	sc := ""
	if l.TunnelEnv {
		sc = "1"
	}
	env := Envs{
		"SYMFONY_TUNNEL":     l.Tunnel,
		"SYMFONY_TUNNEL_ENV": sc,
		"SYMFONY_DOCKER_ENV": docker,
	}
	if _, err := os.Stat(filepath.Join(l.Dir, ".prod")); err == nil {
		env["APP_ENV"] = "prod"
		env["APP_DEBUG"] = "0"
	}

	for k, v := range l.webServer() {
		env[k] = v
	}

	return env
}

func (l *Local) Language() string {
	language := os.Getenv("APP_LANGUAGE")
	if language != "" {
		return language
	}
	projectRoot, err := util.GetProjectRoot(l.Debug)
	if err != nil {
		if l.Debug {
			fmt.Fprint(os.Stderr, "ERROR: unable to get project root\n")
		}
		return "php"
	}
	app := platformsh.GuessSelectedAppByWd(platformsh.FindLocalApplications(projectRoot))
	if app == nil {
		if l.Debug {
			fmt.Fprint(os.Stderr, "ERROR: unable to get project configuration\n")
		}
		return "php"
	}
	parts := strings.Split(app.Type, ":")
	return parts[0]
}

// domain associated with the directory?
func (l *Local) webServer() Envs {
	dir := l.Dir
	var pidFile *pid.PidFile
	for {
		pidFile = pid.New(dir, nil)
		if pidFile.IsRunning() {
			break
		}
		upDir := filepath.Dir(dir)
		if upDir == dir || upDir == "." {
			return nil
		}
		dir = upDir
	}

	port := fmt.Sprintf("%d", pidFile.Port)
	host := fmt.Sprintf("127.0.0.1:%s", port)

	if proxyConf, err := proxy.Load(util.GetHomeDir()); err == nil {
		for _, domain := range proxyConf.GetDomains(l.Dir) {
			// we get the first one only
			host = domain
			if pidFile.Scheme == "http" {
				port = "80"
			} else {
				port = "443"
			}
			break
		}
	}

	url := fmt.Sprintf("%s://%s/", pidFile.Scheme, host)
	env := Envs{}
	for _, prefix := range []string{"SYMFONY_APPLICATION_DEFAULT_ROUTE_", "SYMFONY_PROJECT_DEFAULT_ROUTE_", "SYMFONY_DEFAULT_ROUTE_"} {
		env[prefix+"SCHEME"] = pidFile.Scheme
		env[prefix+"HOST"] = host
		env[prefix+"PORT"] = port
		env[prefix+"URL"] = url
		env[prefix+"PATH"] = "/"
	}

	return env
}
