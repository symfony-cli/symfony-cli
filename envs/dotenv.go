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

// LoadDotEnv adds the variables of the project .env files to vars (the
// variables computed from Docker or tunnels) and returns vars. The goal is
// for PHP scripts that do not boot Symfony's Dotenv to see the values
// Dotenv::loadEnv() would load with its default arguments, without
// interfering with the scripts that do boot it.
//
// # Algorithm
//
//  1. The project directory is the first one containing a .env or .env.dist
//     file, starting from scriptDir and going up. When there is none, nothing
//     is loaded (not even APP_ENV).
//  2. A variable is exported when it is defined in the CLI environment (even
//     empty) and not listed in an inherited SYMFONY_DOTENV_VARS. A listed
//     variable was loaded from .env files by a parent process, so .env files
//     can override it, like Dotenv does.
//  3. The environment is the exported APP_ENV (even empty), else the APP_ENV
//     of .env (even empty), possibly changed by .env.local, else "dev".
//  4. Files are merged in this order, later ones winning: .env (or .env.dist
//     when .env does not exist), .env.local (skipped in the "test"
//     environment), .env.<env>, and .env.<env>.local (both skipped in the
//     "local" environment).
//  5. Exported variables are never taken from .env files.
//  6. Variables already in vars are never taken from .env files.
//  7. SYMFONY_DOTENV_VARS is the inherited value minus the variables already
//     in vars, followed by every variable taken from .env files that is not
//     listed yet, except APP_ENV.
//
// # Differences with Dotenv
//
// Values are parsed with godotenv, not with Dotenv's parser. Unquoted,
// single-quoted, double-quoted, and multiline values, comments, and the
// export prefix behave the same, but:
//
//   - $VAR and ${VAR} only resolve to a variable defined earlier in the same
//     file; Dotenv resolves them once all files are loaded, and also to
//     exported variables. A reference to a variable defined in another file,
//     exported, or computed from Docker, resolves to an empty string.
//   - ${VAR:-default} and ${VAR:=default} are not supported and produce
//     invalid values.
//   - $(command) is kept as is; Dotenv runs the command.
//   - Backslashes differ: godotenv turns "\t" into "t" in double-quoted
//     values and keeps "\\" in unquoted ones, while Dotenv keeps "\t" and
//     turns "\\" into "\".
//   - Adjacent quoted parts (like A='a'"b") make the file unparsable.
//   - Values Dotenv rejects (like unquoted values containing spaces) are
//     accepted.
//
// An unparsable .env (or .env.dist) file makes LoadDotEnv load nothing, and
// any other unparsable file is skipped; Dotenv throws an exception instead.
//
// Dotenv::bootEnv(), which Symfony applications call, also uses
// .env.local.php instead of the .env files when it exists (see composer
// dump-env), and sets APP_DEBUG from the final environment. LoadDotEnv
// ignores .env.local.php and never sets APP_DEBUG, so that Dotenv computes it
// from the environment PHP actually uses.
//
// # Interaction with Dotenv in PHP
//
// When the PHP script boots Dotenv, the variables passed by the CLI are
// either exported for Dotenv, which keeps them, or listed in
// SYMFONY_DOTENV_VARS, which lets Dotenv recompute them from the .env files:
//
//   - Variables computed from Docker or tunnels are never listed, even when
//     inherited, so the database of the Docker Compose project always wins
//     over the DATABASE_URL of the .env files.
//   - Variables taken from .env files are listed, so Dotenv recomputes them
//     with its own parser, which fixes the differences above for Symfony
//     applications.
//   - APP_ENV is never listed: PHP code sets it before booting Dotenv
//     (Symfony Runtime's --env option, PHPUnit's <server name="APP_ENV"
//     force="true"/> or <env> settings), and Dotenv would otherwise revert
//     it to the .env value. Dotenv then loads the files of the environment
//     PHP chose, which gives the same values as without the CLI, except for
//     variables only defined in files of the environment LoadDotEnv chose,
//     which Dotenv does not reset: with "symfony php bin/phpunit" in a
//     project using "dev" by default, a variable only defined in .env.local
//     or .env.dev is still visible in tests.
//   - A nested CLI run from such a PHP process (like a test running "symfony
//     php" in another project) sees APP_ENV as exported and keeps it.
//
// See LookupEnv for a single variable.
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
	var dotEnvVars map[string]string
	if dotEnvDir != "" {
		dotEnvVars = lookupDotEnv(dotEnvDir)
	}
	for k, v := range dotEnvVars {
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

// lookupDotEnv implements steps 2 to 5 of the LoadDotEnv algorithm for the
// project directory dir.
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
