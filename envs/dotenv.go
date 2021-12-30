package envs

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// find .env in the script directory and up
// SHOULD ONLY be enabled on demand, so that Symfony has the priority
// so this feature is only useful for when you want to load a .env file that you are NOT loading yourself, so everything but PHP scripts
// and here, we only have PHP scripts anyway
func LoadDotEnv(vars map[string]string, scriptDir string) map[string]string {
	dotEnvDir := findDotEnvDir(scriptDir)
	vars["SYMFONY_DOTENV_VARS"] = os.Getenv("SYMFONY_DOTENV_VARS")
	for k, v := range lookupDotEnv(dotEnvDir) {
		if _, alreadyDefined := vars[k]; alreadyDefined {
			continue
		}

		vars[k] = v
		if k != "APP_ENV" {
			if vars["SYMFONY_DOTENV_VARS"] != "" {
				vars["SYMFONY_DOTENV_VARS"] += ","
			}
			vars["SYMFONY_DOTENV_VARS"] += k
		}
	}

	return vars
}

// algorithm is here: https://github.com/symfony/recipes/blob/master/symfony/framework-bundle/3.3/config/bootstrap.php
func lookupDotEnv(dir string) map[string]string {
	var err error
	vars := map[string]string{}

	// we prefer loading .env
	path := filepath.Join(dir, ".env")
	if _, err = os.Stat(path); err == nil {
		vars, err = godotenv.Read(path)
		if err != nil {
			return nil
		}
	} else if os.IsNotExist(err) {
		// if .env is not available, let's try to load .env.dist if it exists (for compat)
		path := filepath.Join(dir, ".env.dist")
		if _, err := os.Stat(path); err == nil {
			vars, err = godotenv.Read(path)
			if err != nil {
				return nil
			}
		}
	}

	// APP_ENV defined?
	env := os.Getenv("APP_ENV")
	if env == "" {
		if v, ok := vars["APP_ENV"]; ok {
			env = v
		}
	}
	if env == "" {
		env = "dev"
	}
	vars["APP_ENV"] = env

	if vars["APP_ENV"] != "test" {
		mergeDovEnvFile(vars, filepath.Join(dir, ".env.local"))
	}

	mergeDovEnvFile(vars, filepath.Join(dir, ".env."+vars["APP_ENV"]))

	mergeDovEnvFile(vars, filepath.Join(dir, ".env."+vars["APP_ENV"]+".local"))

	return vars
}

func mergeDovEnvFile(vars map[string]string, path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}

	locals, err := godotenv.Read(path)
	if err != nil {
		return
	}

	for k, v := range locals {
		if _, ok := vars[k]; !ok {
			vars[k] = v
		}
	}
}

func findDotEnvDir(dir string) string {
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return dir
		}
		path = filepath.Join(dir, ".env.dist")
		if _, err := os.Stat(path); err == nil {
			return dir
		}
		upDir := filepath.Dir(dir)
		if dir == upDir {
			return ""
		}
		dir = upDir
	}
}
