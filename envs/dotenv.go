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

// LoadDotEnv adds the variables of the project .env files to vars, so that PHP
// scripts not booting Symfony's Dotenv see the same values as the ones that
// do. It reproduces Dotenv::loadEnv() with its default arguments:
//
//  1. The project directory is the first one containing a .env or .env.dist
//     file, starting from scriptDir and going up.
//  2. A variable is exported when it is defined in the CLI environment (even
//     empty) and not listed in an inherited SYMFONY_DOTENV_VARS. A listed
//     variable was loaded from .env files by a parent process, so .env files
//     can override it, like Dotenv does.
//  3. The environment is the exported APP_ENV (even empty), else the APP_ENV
//     of .env, possibly changed by .env.local, else "dev".
//  4. Files are merged in this order, later ones winning: .env (or .env.dist
//     when .env does not exist), .env.local (skipped in the "test"
//     environment), .env.<env>, and .env.<env>.local (both skipped in the
//     "local" environment).
//  5. Exported variables are never taken from .env files.
//  6. Variables already in vars (computed from Docker or tunnels) are never
//     taken from .env files.
//  7. SYMFONY_DOTENV_VARS starts with the inherited value, minus the variables
//     already in vars so that Dotenv cannot replace them; every variable
//     taken from .env files and not already listed is appended to it so that
//     Dotenv, when the script boots it, can recompute it (for instance after
//     --env changes the environment). APP_ENV is never appended: PHP code sets
//     it before booting Dotenv (Symfony Runtime's --env, PHPUnit's forced
//     APP_ENV), and Dotenv would revert it to the .env value. As a
//     consequence, a nested CLI run from such a PHP process sees APP_ENV as
//     exported.
//
// The returned map is vars itself. See LookupEnv for a single variable.
func LoadDotEnv(vars map[string]string, scriptDir string) map[string]string {
	dotEnvDir := findDotEnvDir(scriptDir)
	loaded := dotEnvLoadedVars()
	var listed []string
	for _, k := range strings.Split(os.Getenv("SYMFONY_DOTENV_VARS"), ",") {
		if _, computed := vars[k]; k != "" && !computed {
			listed = append(listed, k)
		}
	}
	vars["SYMFONY_DOTENV_VARS"] = ""
	for k, v := range lookupDotEnv(dotEnvDir) {
		if _, alreadyDefined := vars[k]; alreadyDefined {
			continue
		}

		vars[k] = v
		if k != "APP_ENV" && !loaded[k] {
			listed = append(listed, k)
		}
	}
	vars["SYMFONY_DOTENV_VARS"] = strings.Join(listed, ",")

	return vars
}

// LookupEnv looks up a single variable like os.LookupEnv would, but takes it
// from the .env files of dotEnvDir unless it is exported, following the
// LoadDotEnv algorithm (without computed variables).
func LookupEnv(dotEnvDir, key string) (string, bool) {
	if value, isDefined := lookupDotEnv(dotEnvDir)[key]; isDefined {
		return value, isDefined
	}

	return os.LookupEnv(key)
}

// lookupDotEnv implements steps 2 to 5 of the LoadDotEnv algorithm.
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

	if env != "local" {
		mergeDovEnvFile(vars, filepath.Join(dir, ".env."+env))
		mergeDovEnvFile(vars, filepath.Join(dir, ".env."+env+".local"))
	}

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
	if isExported("APP_ENV") {
		return os.Getenv("APP_ENV")
	}
	if env, ok := vars["APP_ENV"]; ok {
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
