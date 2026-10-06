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
	"bytes"
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

//go:embed dotenv.php
var dotEnvScript string

// PHPBinary returns the path of the PHP binary to run Dotenv with. It is only
// called when Dotenv needs to run.
type PHPBinary func() (string, error)

// LoadDotEnv adds the variables of the project .env files to vars (the
// variables computed from Docker or tunnels) and returns vars, so that PHP
// scripts see them even when they do not boot Symfony's Dotenv themselves.
//
// # Algorithm
//
//  1. The project directory is the first one containing a .env or .env.dist
//     file, starting from scriptDir and going up. Nothing is loaded when there
//     is none, or when the project does not depend on symfony/dotenv
//     (vendor/symfony/dotenv/Dotenv.php, honoring COMPOSER_VENDOR_DIR and the
//     composer.json vendor-dir option).
//  2. PHP runs the project's Dotenv::bootEnv() on the project .env file, in
//     the CLI environment where vars override exported variables and are
//     removed from an inherited SYMFONY_DOTENV_VARS: Dotenv sees vars as
//     exported, so it never overrides them. Dotenv versions without bootEnv()
//     fall back to loadEnv(), then to load().
//  3. The variables Dotenv lists in SYMFONY_DOTENV_VARS are added to vars,
//     and SYMFONY_DOTENV_VARS is set to that list, except APP_ENV when Dotenv
//     set it (from the .env files or as its "dev" default).
//
// The values are thus exactly the ones the project would load on its own:
// same cascade of .env, .env.local, .env.<env>, and .env.<env>.local files,
// same .env.local.php support, same parser, same precedence of exported
// variables, and $(command) values are run when symfony/process is
// installed. APP_DEBUG, which bootEnv() sets without listing it, is left for
// Dotenv to compute from the environment PHP actually uses.
//
// When Dotenv fails (like on a syntax error), vars is returned without any
// .env variable, with the error.
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
//   - APP_ENV is never listed when it comes from the .env files: PHP code sets
//     it before booting Dotenv (Symfony Runtime's --env option, PHPUnit's
//     <server name="APP_ENV" force="true"/> or <env> settings), and Dotenv
//     would otherwise revert it to the .env value. Dotenv then loads the files
//     of the environment PHP chose, which gives the same values as without the
//     CLI, except for variables only defined in files of the environment
//     LoadDotEnv chose, which Dotenv does not reset: with "symfony php
//     bin/phpunit" in a project using "dev" by default, a variable only
//     defined in .env.local or .env.dev is still visible in tests.
//   - A nested CLI run from such a PHP process (like a test running "symfony
//     php" in another project) sees APP_ENV as exported and keeps it.
//
// See LookupEnv for a single variable.
func LoadDotEnv(vars map[string]string, scriptDir string, phpBinary PHPBinary) (map[string]string, error) {
	inherited := dotEnvLoadedVars()
	var listed []string
	for _, k := range strings.Split(os.Getenv("SYMFONY_DOTENV_VARS"), ",") {
		if _, computed := vars[k]; k != "" && !computed {
			listed = append(listed, k)
		}
	}
	vars["SYMFONY_DOTENV_VARS"] = strings.Join(listed, ",")

	dir := findDotEnvDir(scriptDir)
	if dir == "" {
		return vars, nil
	}
	environ := os.Environ()
	for k, v := range vars {
		environ = append(environ, k+"="+v)
	}
	values, dotEnvListed, err := runDotEnv(dir, phpBinary, environ)
	if err != nil {
		return vars, err
	}

	listed = listed[:0]
	for _, k := range dotEnvListed {
		if k != "APP_ENV" || inherited[k] {
			listed = append(listed, k)
		}
		if v, ok := values[k]; ok {
			vars[k] = v
		}
	}
	vars["SYMFONY_DOTENV_VARS"] = strings.Join(listed, ",")

	return vars, nil
}

// LookupEnv looks up a single variable like os.LookupEnv would, but takes it
// from the .env files of dotEnvDir when Dotenv would load it from there (see
// LoadDotEnv, without computed variables). Dotenv only runs when one of the
// .env files mentions the variable.
func LookupEnv(dotEnvDir, key string, phpBinary PHPBinary) (string, bool) {
	if dotEnvMentions(dotEnvDir, key) {
		if values, _, err := runDotEnv(dotEnvDir, phpBinary, os.Environ()); err == nil {
			if value, ok := values[key]; ok {
				return value, true
			}
		}
	}

	return os.LookupEnv(key)
}

// runDotEnv returns the variables listed by Dotenv in SYMFONY_DOTENV_VARS
// with their values, and the list itself.
func runDotEnv(dir string, phpBinary PHPBinary, environ []string) (map[string]string, []string, error) {
	vendorDir := composerVendorDir(dir)
	if _, err := os.Stat(filepath.Join(vendorDir, "symfony", "dotenv", "Dotenv.php")); err != nil {
		return nil, nil, nil
	}
	bin, err := phpBinary()
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to load the .env files")
	}

	var stderr bytes.Buffer
	cmd := exec.Command(bin, "-d", "display_errors=stderr", "--", vendorDir, filepath.Join(dir, ".env"))
	cmd.Dir = dir
	cmd.Env = append(append([]string{}, environ...), "XDEBUG_MODE=off")
	cmd.Stdin = strings.NewReader(dotEnvScript)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, nil, errors.Errorf("unable to load the .env files: %s", msg)
		}
		return nil, nil, errors.Wrap(err, "unable to load the .env files")
	}

	parts := strings.Split(string(out), "\x00")
	var listed []string
	for _, k := range strings.Split(parts[0], ",") {
		if k != "" {
			listed = append(listed, k)
		}
	}
	values := map[string]string{}
	for i := 1; i+1 < len(parts); i += 2 {
		values[parts[i]] = parts[i+1]
	}

	return values, listed, nil
}

func composerVendorDir(dir string) string {
	vendorDir := os.Getenv("COMPOSER_VENDOR_DIR")
	if vendorDir == "" {
		var composer struct {
			Config struct {
				VendorDir string `json:"vendor-dir"`
			} `json:"config"`
		}
		if content, err := os.ReadFile(filepath.Join(dir, "composer.json")); err == nil && json.Unmarshal(content, &composer) == nil {
			vendorDir = composer.Config.VendorDir
		}
	}
	if vendorDir == "" {
		vendorDir = "vendor"
	}
	if !filepath.IsAbs(vendorDir) {
		vendorDir = filepath.Join(dir, vendorDir)
	}
	return vendorDir
}

func dotEnvMentions(dir, key string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() || (entry.Name() != ".env" && !strings.HasPrefix(entry.Name(), ".env.")) {
			continue
		}
		if content, err := os.ReadFile(filepath.Join(dir, entry.Name())); err == nil && bytes.Contains(content, []byte(key)) {
			return true
		}
	}
	return false
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
