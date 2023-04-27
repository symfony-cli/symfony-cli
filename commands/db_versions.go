package commands

import (
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

func readDBVersionFromPlatformServiceYAML(projectDir string) (string, string, error) {
	servicesYAML, err := ioutil.ReadFile(filepath.Join(projectDir, ".platform", "services.yaml"))
	if err != nil {
		// no services.yaml or unreadable
		return "", "", err
	}
	var services map[string]struct {
		Type string `yaml:"type"`
	}
	if err := yaml.Unmarshal(servicesYAML, &services); err != nil {
		// services.yaml format is wrong
		return "", "", err
	}

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

func readDBVersionFromDotEnv(projectDir string) (string, error) {
	path := filepath.Join(projectDir, ".env")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	vars, err := godotenv.Read(path)
	if err != nil {
		return "", err
	}

	databaseUrl, defined := vars["DATABASE_URL"]
	if !defined {
		return "", nil
	}

	if !strings.Contains(databaseUrl, "serverVersion=") {
		return "", nil
	}

	url, err := url.Parse(databaseUrl)
	if err != nil {
		return "", err
	}

	return url.Query().Get("serverVersion"), nil
}

func readDBVersionFromDoctrineConfigYAML(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "config", "packages", "doctrine.yaml")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	doctrineConfigYAML, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	var doctrineConfig struct {
		Doctrine struct {
			Dbal struct {
				ServerVersion string `yaml:"server_version"`
			} `yaml:"dbal"`
		} `yaml:"doctrine"`
	}
	if err := yaml.Unmarshal(doctrineConfigYAML, &doctrineConfig); err != nil {
		// format is wrong
		return "", err
	}
	return doctrineConfig.Doctrine.Dbal.ServerVersion, nil
}
