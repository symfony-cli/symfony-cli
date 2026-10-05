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
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// find .env in the script directory and up
// SHOULD ONLY be enabled on demand, so that Symfony has the priority
// so this feature is only useful for when you want to load a .env file that you are NOT loading yourself, so everything but PHP scripts
// and here, we only have PHP scripts anyway
func LoadDotEnv(vars map[string]string, scriptDir string) map[string]string {
	dotEnvDir := findDotEnvDir(scriptDir)
	vars["SYMFONY_DOTENV_VARS"] = os.Getenv("SYMFONY_DOTENV_VARS")
	loaded := dotEnvLoadedVars()
	for k, v := range lookupDotEnv(dotEnvDir) {
		if _, alreadyDefined := vars[k]; alreadyDefined {
			continue
		}

		vars[k] = v
		if !loaded[k] {
			if vars["SYMFONY_DOTENV_VARS"] != "" {
				vars["SYMFONY_DOTENV_VARS"] += ","
			}
			vars["SYMFONY_DOTENV_VARS"] += k
		}
	}

	return vars
}

// LookupEnv allows one to lookup for a single environment variable in the same
// way os.LookupEnv would. Exported variables win over .env files.
func LookupEnv(dotEnvDir, key string) (string, bool) {
	if value, isDefined := lookupDotEnv(dotEnvDir)[key]; isDefined {
		return value, isDefined
	}

	return os.LookupEnv(key)
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

	env := resolveAppEnv(vars)
	vars["APP_ENV"] = env

	if env != "test" {
		mergeDovEnvFile(vars, filepath.Join(dir, ".env.local"))
		env = resolveAppEnv(vars)
		vars["APP_ENV"] = env
	}

	mergeDovEnvFile(vars, filepath.Join(dir, ".env."+env))

	mergeDovEnvFile(vars, filepath.Join(dir, ".env."+env+".local"))

	// Exported variables win, as with Symfony's Dotenv component
	for k := range vars {
		if isExported(k) {
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

	maps.Copy(vars, locals)
}

func resolveAppEnv(vars map[string]string) string {
	if env := os.Getenv("APP_ENV"); env != "" && isExported("APP_ENV") {
		return env
	}
	if env := vars["APP_ENV"]; env != "" {
		return env
	}
	return "dev"
}

// isExported mirrors Symfony's Dotenv: variables loaded from .env files by a
// parent process (listed in SYMFONY_DOTENV_VARS) can be overridden.
func isExported(key string) bool {
	if _, ok := os.LookupEnv(key); !ok {
		return false
	}
	return !dotEnvLoadedVars()[key]
}

func dotEnvLoadedVars() map[string]bool {
	loaded := map[string]bool{}
	for _, k := range strings.Split(os.Getenv("SYMFONY_DOTENV_VARS"), ",") {
		if k != "" {
			loaded[k] = true
		}
	}
	return loaded
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
