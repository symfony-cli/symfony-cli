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

// LookupEnv allows one to lookup for a single environment variable in the same
// way os.LookupEnv would. It automatically let the environment variable take
// over if defined.
func LookupEnv(dotEnvDir, key string) (string, bool) {
	// first check if the user defined it in its environment
	if value, isUserDefined := os.LookupEnv(key); isUserDefined {
		return value, isUserDefined
	}

	dotEnvEnv := lookupDotEnv(dotEnvDir)
	if value, isDefined := dotEnvEnv[key]; isDefined {
		return value, isDefined
	}

	return "", false
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

	// When the user has set environment variables, we inherit them instead of overwrite it
	for k, _ := range vars {
		if os.Getenv(k) != "" {
			delete(vars, k)
		}
	}

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
