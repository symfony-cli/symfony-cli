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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/local/platformsh"
	"github.com/symfony-cli/symfony-cli/util"
)

type pshtunnel struct {
	EnvironmentID string                 `json:"environmentId"`
	AppName       string                 `json:"appName"`
	ProjectID     string                 `json:"projectId"`
	Relationship  string                 `json:"relationship"`
	LocalPort     int                    `json:"localPort"`
	Service       map[string]interface{} `json:"service"`
}

func (l *Local) relationshipsFromTunnel() Relationships {
	project, err := platformsh.ProjectFromDir(l.Dir, l.Debug)
	if err != nil {
		if l.Debug {
			fmt.Fprintf(os.Stderr, "WARNING: unable to detect Platform.sh project: %s\n", err)
		}
		return nil
	}

	userHomeDir, err := homedir.Dir()
	if err != nil {
		userHomeDir = ""
	}
	tunnelFile := filepath.Join(userHomeDir, ".platformsh", "tunnel-info.json")
	data, err := os.ReadFile(tunnelFile)
	if err != nil {
		if l.Debug {
			fmt.Fprintf(os.Stderr, "WARNING: unable to read relationships from %s: %s\n", tunnelFile, err)
		}
		return nil
	}
	var tunnels []pshtunnel
	if err := json.Unmarshal(data, &tunnels); err != nil {
		// For some reasons, psh sometimes dump the tunnel file as a map
		var alttunnels map[string]pshtunnel
		if err := json.Unmarshal(data, &alttunnels); err != nil {
			if l.Debug {
				fmt.Fprintf(os.Stderr, "ERROR: unable to unmarshal tunnel data: %s: %s\n", tunnelFile, err)
			}
			return nil
		}
		for _, config := range alttunnels {
			tunnels = append(tunnels, config)
		}
	}
	rels := make(Relationships)
	for _, config := range tunnels {
		if config.ProjectID == project.ID && config.EnvironmentID == project.Env && config.AppName == project.App {
			config.Service["port"] = strconv.Itoa(config.LocalPort)
			config.Service["host"] = "127.0.0.1"
			config.Service["ip"] = "127.0.0.1"
			rels[config.Relationship] = append(rels[config.Relationship], config.Service)
		}
	}

	if len(rels) > 0 {
		l.Tunnel = project.Env
		l.TunnelEnv = true
		return rels
	}

	return nil
}

var pathCleaningRegex = regexp.MustCompile(`[^a-zA-Z0-9-\.]+`)

type Tunnel struct {
	Project *platformsh.Project
	Worker  string
	Debug   bool
}

func (t *Tunnel) IsExposed() bool {
	if _, err := os.Stat(t.path()); err != nil {
		return false
	}
	return true
}

func (t *Tunnel) Expose(expose bool) error {
	path := t.path()
	if expose {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return errors.WithStack(err)
		}
		file, err := os.Create(path)
		if err != nil {
			return errors.WithStack(err)
		}
		return errors.WithStack(file.Close())
	}

	return errors.WithStack(os.Remove(path))
}

// Path returns the path to the Platform.sh local tunnel state file
func (t *Tunnel) path() string {
	var filename bytes.Buffer

	filename.WriteString(t.Project.ID)
	filename.WriteRune('-')
	filename.WriteString(t.Project.Env)

	if t.Project.App != "" {
		filename.WriteString("--")
		filename.WriteString(t.Project.App)
	}

	if t.Worker != "" {
		filename.WriteString("--")
		filename.WriteString(t.Worker)
	}

	filename.WriteString("-expose.json")

	return filepath.Join(filepath.Join(util.GetHomeDir(), "tunnels"), pathCleaningRegex.ReplaceAllString(path.Clean(filename.String()), "-"))
}
