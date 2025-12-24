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

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ddevDatabase struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
}

type ddevConfig struct {
	Name                string       `yaml:"name"`
	Type                string       `yaml:"type"`
	Docroot             string       `yaml:"docroot"`
	PHPVersion          string       `yaml:"php_version"`
	WebserverType       string       `yaml:"webserver_type"`
	XdebugEnabled       bool         `yaml:"xdebug_enabled"`
	AdditionalHostnames []string     `yaml:"additional_hostnames,omitempty"`
	AdditionalFQDNs     []string     `yaml:"additional_fqdns,omitempty"`
	Database            ddevDatabase `yaml:"database,omitempty"`
}

type semver struct {
	major int
	minor int
	patch int
}

var semverRe = regexp.MustCompile(`(?i)\bv?(\d+)\.(\d+)\.(\d+)\b`)

func isDdevAvailable() error {
	// Prefer a cheap check first
	if _, err := exec.LookPath("ddev"); err != nil {
		return fmt.Errorf("ddev CLI not found in PATH: %w", err)
	}

	_, err := getDdevVersion()
	if err != nil {
		return err
	}

	return nil
}

func getDdevVersion() (string, error) {
	cmd := exec.Command("ddev", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get DDEV version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func getLatestDdevPHPVersion() (string, error) {
	ddevVersion, err := getDdevVersion()
	if err != nil {
		return "", err
	}

	v, err := parseSemver(ddevVersion)
	if err != nil {
		return "", fmt.Errorf("could not parse ddev version %q: %w", ddevVersion, err)
	}

	switch {
	case v.gte(semver{major: 1, minor: 24, patch: 0}):
		return "8.4", nil
	case v.gte(semver{major: 1, minor: 22, patch: 5}):
		return "8.3", nil
	default:
		return "", fmt.Errorf("DDEV version %q is not supported, please upgrade to at least 1.22.5", ddevVersion)
	}
}

func createDdevConfigFile(dir string, phpVersion string, services []*Service) error {
	if err := os.MkdirAll(filepath.Join(dir, ".ddev"), 0o755); err != nil {
		return err
	}

	phpVersion = strings.TrimSpace(phpVersion)
	if phpVersion == "" {
		return fmt.Errorf("phpVersion is required")
	}

	projectName := filepath.Base(filepath.Clean(dir))
	if projectName == "." || projectName == string(filepath.Separator) || projectName == "" {
		projectName = "project"
	}

	cfg := ddevConfig{
		Name:          projectName,
		Type:          "php",
		Docroot:       "public",
		PHPVersion:    phpVersion,
		WebserverType: "nginx-fpm",
		XdebugEnabled: false,
	}

	if dbService := getDatabaseService(services); dbService != nil {
		cfg.Database = ddevDatabase{
			Type:    dbService.Type,
			Version: dbService.Version,
		}
	}

	outPath := filepath.Join(dir, ".ddev", "config.yaml")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(&cfg); err != nil {
		_ = enc.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}

	return nil
}

func (v semver) gte(o semver) bool {
	if v.major != o.major {
		return v.major > o.major
	}
	if v.minor != o.minor {
		return v.minor > o.minor
	}
	return v.patch >= o.patch
}

func parseSemver(s string) (semver, error) {
	s = strings.TrimSpace(s)
	m := semverRe.FindStringSubmatch(s)
	if len(m) != 4 {
		return semver{}, fmt.Errorf("expected major.minor.patch, got %q", s)
	}

	maj, err := strconv.Atoi(m[1])
	if err != nil {
		return semver{}, err
	}
	min, err := strconv.Atoi(m[2])
	if err != nil {
		return semver{}, err
	}
	pat, err := strconv.Atoi(m[3])
	if err != nil {
		return semver{}, err
	}

	return semver{major: maj, minor: min, patch: pat}, nil
}

func getDatabaseService(services []*Service) *Service {
	for _, service := range services {
		if service.Name == "database" {
			return service
		}
	}
	return nil
}
