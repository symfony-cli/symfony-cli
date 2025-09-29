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

package upsun

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

type serviceConfigs map[string]struct {
	Type string `yaml:"type"`
}

func ReadDBVersionFromPlatformServiceYAML(projectDir string, logger zerolog.Logger) (string, string, string) {
	// Platform.sh
	configFile := filepath.Join(".platform", "services.yaml")
	if servicesYAML, err := os.ReadFile(filepath.Join(projectDir, configFile)); err == nil {
		var services serviceConfigs
		if err := yaml.Unmarshal(servicesYAML, &services); err == nil {
			if dbName, dbVersion, err := extractCloudDatabaseType(services); err == nil {
				logger.Debug().Msg("DB configured in .platform/services.yaml")
				return configFile, dbName, dbVersion
			} else {
				logger.Debug().Msg("No DB configured in .platform/services.yaml")
			}
		} else {
			logger.Debug().Msg("Unable to parse .platform/services.yaml file")
		}
	} else {
		logger.Debug().Msg("No .platform/services.yaml file found or not readable")
	}

	// Upsun
	upsunDir := filepath.Join(projectDir, ".upsun")
	if _, err := os.Stat(upsunDir); err == nil {
		if files, err := os.ReadDir(upsunDir); err == nil {
			for _, file := range files {
				configFile := filepath.Join(".upsun", file.Name())
				if servicesYAML, err := os.ReadFile(filepath.Join(projectDir, configFile)); err == nil {
					var config struct {
						Services serviceConfigs `yaml:"services"`
					}
					if err := yaml.Unmarshal(servicesYAML, &config); err == nil {
						if dbName, dbVersion, err := extractCloudDatabaseType(config.Services); err == nil {
							logger.Debug().Msgf("DB configured in %s", configFile)
							return configFile, dbName, dbVersion
						} else {
							logger.Debug().Msgf("No DB configured in %s", configFile)
						}
					} else {
						logger.Debug().Msgf("Unable to parse the %s file", configFile)
					}
				} else {
					logger.Debug().Msgf("Unable to read the %s file", configFile)
				}
			}
		} else {
			logger.Debug().Msg("Unable to list files under the .upsun directory")
		}
	} else {
		logger.Debug().Msg("No .upsun directory found")
	}
	logger.Debug().Msg("No DB configured")
	return "", "", ""
}

func extractCloudDatabaseType(services serviceConfigs) (string, string, error) {
	dbName := ""
	dbVersion := ""
	for _, service := range services {
		if strings.HasPrefix(service.Type, "mysql") || strings.HasPrefix(service.Type, "mariadb") || strings.HasPrefix(service.Type, "postgresql") {
			if dbName != "" {
				// give up as there are multiple DBs
				return "", "", nil
			}

			parts := strings.Split(service.Type, ":")
			dbName = parts[0]
			dbVersion = parts[1]
		}
	}
	return dbName, dbVersion, nil
}

func ReadDBVersionFromDotEnv(projectDir string) (string, error) {
	path := filepath.Join(projectDir, ".env")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	vars, err := godotenv.Read(path)
	if err != nil {
		return "", err
	}

	databaseURL, defined := vars["DATABASE_URL"]
	if !defined {
		return "", nil
	}

	if !strings.Contains(databaseURL, "serverVersion=") {
		return "", nil
	}

	url, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}

	return url.Query().Get("serverVersion"), nil
}

func ReadDBVersionFromDoctrineConfigYAML(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "config", "packages", "doctrine.yaml")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	doctrineConfigYAML, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var doctrineConfig struct {
		Doctrine struct {
			Dbal struct {
				ServerVersion string `yaml:"server_version"`
				Connections   struct {
					Default struct {
						ServerVersion string `yaml:"server_version"`
					} `yaml:"default"`
				}
			} `yaml:"dbal"`
		} `yaml:"doctrine"`
	}
	if err := yaml.Unmarshal(doctrineConfigYAML, &doctrineConfig); err != nil {
		// format is wrong
		return "", err
	}

	version := doctrineConfig.Doctrine.Dbal.Connections.Default.ServerVersion
	if version == "" {
		version = doctrineConfig.Doctrine.Dbal.ServerVersion
	}
	if version == "" {
		// empty version
		return "", nil
	}
	if version[0] == '%' && version[len(version)-1] == '%' {
		// references an env var, ignore
		return "", nil
	}
	return version, nil
}

func DatabaseVersiondUnsynced(providedVersion, dbVersion string) bool {
	providedVersion = strings.Replace(providedVersion, "mariadb-", "", 1)

	return providedVersion != "" && !strings.HasPrefix(providedVersion, dbVersion)
}
