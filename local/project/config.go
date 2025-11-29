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
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/terminal"
	"gopkg.in/yaml.v2"
)

const (
	DockerComposeWorkerKey = "docker_compose"
)

type config struct {
	Logger     zerolog.Logger
	HomeDir    string
	ProjectDir string

	NoWorkers bool
	Daemon    bool

	HTTP struct {
		DocumentRoot  string
		Passthru      string
		Port          int
		PreferredPort int
		ListenIp      string
		AllowHTTP     bool
		NoTLS         bool
		PKCS12        string
		TlsKeyLogFile string
		UseGzip       bool
		AllowCORS     bool
	}
	Workers map[string]struct {
		Cmd   []string
		Watch []string
	}
	Proxy struct {
		Domains []string
	}
}

func NewConfigFromDirectory(logger zerolog.Logger, homeDir, projectDir string) (*config, error) {
	config := &config{
		Logger:     logger,
		HomeDir:    homeDir,
		ProjectDir: projectDir,
		Workers: make(map[string]struct {
			Cmd   []string
			Watch []string
		}),
	}

	// Only one nomenclature can be used at a time
	for _, prefix := range []string{".symfony.cli", ".symfony.local"} {
		found := false

		// first consider project configuration files in this specific order
		for _, suffix := range []string{".dist.yaml", ".yaml", ".override.yaml"} {
			fileConfig, err := newConfigFromFile(filepath.Join(projectDir, prefix+suffix))
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return nil, err
			} else if fileConfig == nil {
				continue
			}

			if prefix == ".symfony.local" {
				terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin).Warning(fmt.Sprintf(`The "%s" configuration file have been deprecated since v5.17.0,
please use "%s" instead.`, prefix+suffix, ".symfony.cli"+suffix))
			}

			config.mergeWithFileConfig(*fileConfig)
			found = true
		}

		if found {
			break
		}
	}

	for k, v := range config.Workers {
		if len(v.Cmd) == 0 {
			return nil, errors.Errorf(`The command for the "%s" worker entry cannot be empty.`, k)
		}
	}

	return config, nil
}

func NewConfigFromContext(c *console.Context, logger zerolog.Logger, homeDir, projectDir string) (*config, error) {
	config, err := NewConfigFromDirectory(logger, homeDir, projectDir)
	if err != nil {
		return nil, err
	}

	// then each option that can be overridden by command line flags
	config.mergeWithContext(c)

	return config, nil
}

func (config *config) mergeWithContext(c *console.Context) {
	if c.IsSet("allow-all-ip") {
		config.HTTP.ListenIp = ""
	} else {
		config.HTTP.ListenIp = c.String("listen-ip")
	}
	if c.IsSet("document-root") {
		config.HTTP.DocumentRoot = c.String("document-root")
	}
	if c.IsSet("passthru") {
		config.HTTP.Passthru = c.String("passthru")
	}
	if c.IsSet("port") {
		config.HTTP.Port = c.Int("port")
	}
	if config.HTTP.Port == 0 {
		config.HTTP.PreferredPort = 8000
	}
	if c.IsSet("allow-cors") {
		config.HTTP.AllowCORS = c.Bool("allow-cors")
	}
	if c.IsSet("allow-http") {
		config.HTTP.AllowHTTP = c.Bool("allow-http")
	}
	if c.IsSet("p12") {
		config.HTTP.PKCS12 = c.String("p12")
	}
	if c.IsSet("no-tls") {
		config.HTTP.NoTLS = c.Bool("no-tls")
	}
	if c.IsSet("tls-key-log-suffix") {
		config.HTTP.TlsKeyLogFile = c.String("tls-key-log-suffix")
	}
	if c.IsSet("use-gzip") {
		config.HTTP.UseGzip = c.Bool("use-gzip")
	}
	if c.IsSet("daemon") {
		config.Daemon = c.Bool("daemon")
	}
	if c.IsSet("no-workers") {
		config.NoWorkers = c.Bool("no-workers")
	}
}

func (config *config) mergeWithFileConfig(fileConfig fileConfig) {
	config.Logger.Debug().Msgf("Loading configuration from %s", fileConfig.filename)

	if fileConfig.Daemon != nil {
		config.Daemon = *fileConfig.Daemon
	}
	if fileConfig.NoWorkers != nil {
		config.NoWorkers = *fileConfig.NoWorkers
	}

	if fileConfig.Proxy != nil {
		config.Proxy.Domains = fileConfig.Proxy.Domains
	}

	if fileConfig.Workers != nil {
		for workerName, fileWorker := range fileConfig.Workers {
			worker, hasWorkerDefined := config.Workers[workerName]

			if fileWorker == nil {
				if !hasWorkerDefined {
					continue
				}

				delete(config.Workers, workerName)
				continue
			}

			if fileWorker.Cmd != nil {
				worker.Cmd = fileWorker.Cmd
			}

			if fileWorker.Watch != nil {
				worker.Watch = fileWorker.Watch
			}

			config.Workers[workerName] = worker
		}
	}

	if fileConfig.HTTP != nil {
		if fileConfig.HTTP.DocumentRoot != nil {
			config.HTTP.DocumentRoot = *fileConfig.HTTP.DocumentRoot
		}
		if fileConfig.HTTP.Passthru != nil {
			config.HTTP.Passthru = *fileConfig.HTTP.Passthru
		}
		if fileConfig.HTTP.Port != nil {
			config.HTTP.Port = *fileConfig.HTTP.Port
		}
		if fileConfig.HTTP.PreferredPort != nil {
			config.HTTP.PreferredPort = *fileConfig.HTTP.PreferredPort
		}
		if fileConfig.HTTP.AllowCORS != nil {
			config.HTTP.AllowCORS = *fileConfig.HTTP.AllowCORS
		}
		if fileConfig.HTTP.AllowHTTP != nil {
			config.HTTP.AllowHTTP = *fileConfig.HTTP.AllowHTTP
		}
		if fileConfig.HTTP.NoTLS != nil {
			config.HTTP.NoTLS = *fileConfig.HTTP.NoTLS
		}
		if fileConfig.HTTP.PKCS12 != nil {
			config.HTTP.PKCS12 = *fileConfig.HTTP.PKCS12
		}
		if fileConfig.HTTP.TlsKeyLogFile != nil {
			config.HTTP.TlsKeyLogFile = *fileConfig.HTTP.TlsKeyLogFile
		}
		if fileConfig.HTTP.UseGzip != nil {
			config.HTTP.UseGzip = *fileConfig.HTTP.UseGzip
		}

		if fileConfig.HTTP.Daemon != nil {
			config.Daemon = *fileConfig.HTTP.Daemon
			config.Logger.Warn().Msgf(`"http.daemon" setting has been deprecated since v5.12.0, use the "daemon" (at root level) setting instead.`)
		}
		if fileConfig.HTTP.NoWorkers != nil {
			config.NoWorkers = *fileConfig.HTTP.NoWorkers
			config.Logger.Warn().Msgf(`"http.no_workers" setting has been deprecated since v5.12.0, use the "no_workers" (at root level) setting instead.`)
		}
	}
}

type fileConfig struct {
	filename string

	NoWorkers *bool `yaml:"no_workers"`
	Daemon    *bool `yaml:"daemon"`

	Proxy *struct {
		Domains []string `yaml:"domains"`
	} `yaml:"proxy"`
	HTTP *struct {
		DocumentRoot  *string `yaml:"document_root"`
		Passthru      *string `yaml:"passthru"`
		Port          *int    `yaml:"port"`
		PreferredPort *int    `yaml:"preferred_port"`
		AllowHTTP     *bool   `yaml:"allow_http"`
		NoTLS         *bool   `yaml:"no_tls"`
		PKCS12        *string `yaml:"p12"`
		TlsKeyLogFile *string `yaml:"tls_key_log_file"`
		UseGzip       *bool   `yaml:"use_gzip"`
		AllowCORS     *bool   `yaml:"allow_cors"`

		// BC-layer
		Daemon    *bool `yaml:"daemon"`
		NoWorkers *bool `yaml:"no_workers"`
	} `yaml:"http"`
	Workers map[string]*workerFileConfig `yaml:"workers"`
}

type workerFileConfig struct {
	Cmd   []string `yaml:"cmd"`
	Watch []string `yaml:"watch"`
}

// Should only be used when for customers
func newConfigFromFile(configFile string) (*fileConfig, error) {
	if _, err := os.Stat(configFile); err != nil {
		return nil, errors.Wrapf(err, "config file %s does not exist", configFile)
	}

	contents, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	fileConfig := fileConfig{
		filename: filepath.Base(configFile),
	}
	if err := yaml.Unmarshal(contents, &fileConfig); err != nil {
		return nil, err
	}

	if err := fileConfig.parseWorkers(); err != nil {
		return nil, err
	}

	return &fileConfig, nil
}

func (c *fileConfig) parseWorkers() error {
	if c.Workers == nil {
		return nil
	}

	if v, ok := c.Workers[DockerComposeWorkerKey]; ok && v == nil {
		c.Workers[DockerComposeWorkerKey] = &workerFileConfig{
			Cmd: []string{"docker", "compose", "up"},
			Watch: []string{
				"compose.yaml", "compose.override.yaml",
				"compose.yml", "compose.override.yml",
				"docker-compose.yml", "docker-compose.override.yml",
				"docker-compose.yaml", "docker-compose.override.yaml",
			},
		}
	}
	if v, ok := c.Workers["yarn_encore_watch"]; ok && v == nil {
		c.Workers["yarn_encore_watch"] = &workerFileConfig{
			Cmd: []string{"yarn", "encore", "dev", "--watch"},
		}
	}
	if v, ok := c.Workers["messenger_consume_async"]; ok && v == nil {
		c.Workers["messenger_consume_async"] = &workerFileConfig{
			Cmd:   []string{"symfony", "console", "messenger:consume", "async"},
			Watch: []string{"config", "src", "templates", "vendor/composer/installed.json"},
		}
	}

	return nil
}
