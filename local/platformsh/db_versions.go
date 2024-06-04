package platformsh

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

func ReadDBVersionFromPlatformServiceYAML(projectDir string) (string, string, string, error) {
	configFile := filepath.Join(projectDir, ".platform", "services.yaml")
	servicesYAML, err := os.ReadFile(configFile)
	if err != nil {
		// no services.yaml or unreadable
		return "", "", "", err
	}
	var services map[string]struct {
		Type string `yaml:"type"`
	}
	if err := yaml.Unmarshal(servicesYAML, &services); err != nil {
		// services.yaml format is wrong
		return "", "", "", err
	}

	dbName := ""
	dbVersion := ""
	for _, service := range services {
		if strings.HasPrefix(service.Type, "mysql") || strings.HasPrefix(service.Type, "mariadb") || strings.HasPrefix(service.Type, "postgresql") {
			if dbName != "" {
				// give up as there are multiple DBs
				return "", "", "", nil
			}

			parts := strings.Split(service.Type, ":")
			dbName = parts[0]
			dbVersion = parts[1]
		}
	}
	return configFile, dbName, dbVersion, nil
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
