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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"gopkg.in/yaml.v2"
)

// Config is the struct taken by New (should not be used for anything else)
type Config struct {
	HomeDir       string
	ProjectDir    string
	DocumentRoot  string `yaml:"document_root"`
	Passthru      string `yaml:"passthru"`
	Port          int    `yaml:"port"`
	PreferredPort int    `yaml:"preferred_port"`
	PKCS12        string `yaml:"p12"`
	Logger        zerolog.Logger
	AppVersion    string
	AllowHTTP     bool `yaml:"allow_http"`
	NoTLS         bool `yaml:"no_tls"`
	Daemon        bool `yaml:"daemon"`
}

type FileConfig struct {
	Proxy struct {
		Domains []string `yaml:"domains"`
	} `yaml:"proxy"`
	HTTP    *Config            `yaml:"http"`
	Workers map[string]*Worker `yaml:"workers"`
}

type Worker struct {
	Cmd   []string `yaml:"cmd"`
	Watch []string `yaml:"watch"`
}

func NewConfigFromContext(c *console.Context, projectDir string) (*Config, *FileConfig, error) {
	config := &Config{}
	var fileConfig *FileConfig
	var err error
	fileConfig, err = newConfigFromFile(filepath.Join(projectDir, ".symfony.local.yaml"))
	if err != nil {
		return nil, nil, err
	}
	if fileConfig != nil {
		if fileConfig.HTTP == nil {
			fileConfig.HTTP = &Config{}
		} else {
			config = fileConfig.HTTP
		}
		if fileConfig.Workers == nil {
			fileConfig.Workers = make(map[string]*Worker)
		}
	}
	config.AppVersion = c.App.Version
	config.ProjectDir = projectDir
	if c.IsSet("document-root") {
		config.DocumentRoot = c.String("document-root")
	}
	if c.IsSet("passthru") {
		config.Passthru = c.String("passthru")
	}
	if c.IsSet("port") {
		config.Port = c.Int("port")
	}
	if config.Port == 0 {
		config.PreferredPort = 8000
	}
	if c.IsSet("allow-http") {
		config.AllowHTTP = c.Bool("allow-http")
	}
	if c.IsSet("p12") {
		config.PKCS12 = c.String("p12")
	}
	if c.IsSet("no-tls") {
		config.NoTLS = c.Bool("no-tls")
	}
	if c.IsSet("daemon") {
		config.Daemon = c.Bool("daemon")
	}
	return config, fileConfig, nil
}

// Should only be used when for customers
func newConfigFromFile(configFile string) (*FileConfig, error) {
	if _, err := os.Stat(configFile); err != nil {
		return nil, nil
	}

	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var fileConfig FileConfig
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return nil, err
	}

	if err := fileConfig.parseWorkers(); err != nil {
		return nil, err
	}

	return &fileConfig, nil
}

func (c *FileConfig) parseWorkers() error {
	if c.Workers == nil {
		c.Workers = make(map[string]*Worker)
		return nil
	}

	if v, ok := c.Workers["yarn_encore_watch"]; ok && v == nil {
		c.Workers["yarn_encore_watch"] = &Worker{
			Cmd: []string{"yarn", "encore", "dev", "--watch"},
		}
	}
	if v, ok := c.Workers["messenger_consume_async"]; ok && v == nil {
		c.Workers["messenger_consume_async"] = &Worker{
			Cmd:   []string{"symfony", "console", "messenger:consume", "async"},
			Watch: []string{"config", "src", "templates", "vendor"},
		}
	}

	for k, v := range c.Workers {
		if v == nil {
			return errors.Errorf("The \"%s\" worker entry in \".symfony.local.yaml\" cannot be empty.", k)
		}
	}

	return nil
}
