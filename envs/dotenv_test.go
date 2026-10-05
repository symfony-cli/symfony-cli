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
	"slices"
	"strings"

	. "gopkg.in/check.v1"
)

type DotEnvSuite struct{}

var _ = Suite(&DotEnvSuite{})

func (s *DotEnvSuite) TestLoadDotEnvKeepsExportedVariables(c *C) {
	dir := c.MkDir()
	c.Assert(os.WriteFile(filepath.Join(dir, ".env"), []byte("EXPORTED=dotenv\nEXPORTED_EMPTY=dotenv\nDOTENV_ONLY=dotenv\n"), 0644), IsNil)

	for k, v := range map[string]string{"EXPORTED": "exported", "EXPORTED_EMPTY": ""} {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}
	for _, k := range []string{"APP_ENV", "SYMFONY_DOTENV_VARS"} {
		if v, ok := os.LookupEnv(k); ok {
			os.Unsetenv(k)
			defer os.Setenv(k, v)
		}
	}

	vars := LoadDotEnv(map[string]string{}, dir)

	c.Check(vars["DOTENV_ONLY"], Equals, "dotenv")
	c.Check(sortedDotEnvVars(vars), DeepEquals, []string{"APP_ENV", "DOTENV_ONLY"})
	_, ok := vars["EXPORTED"]
	c.Check(ok, Equals, false)
	_, ok = vars["EXPORTED_EMPTY"]
	c.Check(ok, Equals, false)
}

func (s *DotEnvSuite) TestLoadDotEnvOverridesVariablesLoadedByParentProcess(c *C) {
	dir := c.MkDir()
	c.Assert(os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_ENV=dev\nINHERITED=dotenv\nEXPORTED=dotenv\n"), 0644), IsNil)

	// INHERITED was loaded from a .env file by a parent process, EXPORTED by the user
	for k, v := range map[string]string{"INHERITED": "parent", "EXPORTED": "exported", "APP_ENV": "prod", "SYMFONY_DOTENV_VARS": "INHERITED,APP_ENV"} {
		if old, ok := os.LookupEnv(k); ok {
			defer os.Setenv(k, old)
		} else {
			defer os.Unsetenv(k)
		}
		os.Setenv(k, v)
	}

	vars := LoadDotEnv(map[string]string{}, dir)

	c.Check(vars["INHERITED"], Equals, "dotenv")
	c.Check(vars["APP_ENV"], Equals, "dev")
	_, ok := vars["EXPORTED"]
	c.Check(ok, Equals, false)
	c.Check(vars["SYMFONY_DOTENV_VARS"], Equals, "INHERITED,APP_ENV")
}

func (s *DotEnvSuite) TestLoadDotEnvInNestedRunUsesInnerProjectEnvironment(c *C) {
	outer := c.MkDir()
	c.Assert(os.WriteFile(filepath.Join(outer, ".env"), []byte("APP_ENV=prod\nFOO=outer\n"), 0644), IsNil)
	c.Assert(os.WriteFile(filepath.Join(outer, ".env.prod"), []byte("BAR=outer-prod\n"), 0644), IsNil)
	inner := c.MkDir()
	c.Assert(os.WriteFile(filepath.Join(inner, ".env"), []byte("APP_ENV=dev\nFOO=inner\n"), 0644), IsNil)
	c.Assert(os.WriteFile(filepath.Join(inner, ".env.dev"), []byte("BAR=inner-dev\n"), 0644), IsNil)
	c.Assert(os.WriteFile(filepath.Join(inner, ".env.prod"), []byte("BAR=inner-prod\n"), 0644), IsNil)

	for _, k := range []string{"APP_ENV", "FOO", "BAR", "SYMFONY_DOTENV_VARS"} {
		if old, ok := os.LookupEnv(k); ok {
			defer os.Setenv(k, old)
		} else {
			defer os.Unsetenv(k)
		}
		os.Unsetenv(k)
	}

	for k, v := range LoadDotEnv(map[string]string{}, outer) {
		os.Setenv(k, v)
	}
	vars := LoadDotEnv(map[string]string{}, inner)

	c.Check(vars["APP_ENV"], Equals, "dev")
	c.Check(vars["FOO"], Equals, "inner")
	c.Check(vars["BAR"], Equals, "inner-dev")
	c.Check(sortedDotEnvVars(vars), DeepEquals, []string{"APP_ENV", "BAR", "FOO"})
}

func (s *DotEnvSuite) TestLookupDotEnvCascade(c *C) {
	for _, tc := range []struct {
		name     string
		appEnv   string
		files    map[string]string
		expected map[string]string
	}{
		{
			name: "later files override earlier ones",
			files: map[string]string{
				".env":           "A=env\nB=env\nC=env\nD=env\n",
				".env.local":     "B=local\nC=local\nD=local\n",
				".env.dev":       "C=dev\nD=dev\n",
				".env.dev.local": "D=dev.local\n",
			},
			expected: map[string]string{"APP_ENV": "dev", "A": "env", "B": "local", "C": "dev", "D": "dev.local"},
		},
		{
			name: ".env.local can change the environment",
			files: map[string]string{
				".env":       "APP_ENV=dev\nA=env\n",
				".env.local": "APP_ENV=prod\n",
				".env.dev":   "A=dev\n",
				".env.prod":  "A=prod\n",
			},
			expected: map[string]string{"APP_ENV": "prod", "A": "prod"},
		},
		{
			name:   "an exported environment wins over .env.local",
			appEnv: "dev",
			files: map[string]string{
				".env":       "A=env\n",
				".env.local": "APP_ENV=prod\n",
				".env.dev":   "A=dev\n",
				".env.prod":  "A=prod\n",
			},
			expected: map[string]string{"A": "dev"},
		},
		{
			name: ".env.local is ignored in the test environment",
			files: map[string]string{
				".env":       "APP_ENV=test\nA=env\n",
				".env.local": "A=local\n",
			},
			expected: map[string]string{"APP_ENV": "test", "A": "env"},
		},
	} {
		dir := c.MkDir()
		for name, content := range tc.files {
			c.Assert(os.WriteFile(filepath.Join(dir, name), []byte(content), 0644), IsNil)
		}
		withAppEnv(tc.appEnv, func() {
			c.Check(lookupDotEnv(dir), DeepEquals, tc.expected, Commentf(tc.name))
		})
	}
}

func sortedDotEnvVars(vars map[string]string) []string {
	keys := strings.Split(vars["SYMFONY_DOTENV_VARS"], ",")
	slices.Sort(keys)
	return keys
}

func withAppEnv(value string, fn func()) {
	previous, wasSet := os.LookupEnv("APP_ENV")
	defer func() {
		if wasSet {
			os.Setenv("APP_ENV", previous)
		} else {
			os.Unsetenv("APP_ENV")
		}
	}()
	if value == "" {
		os.Unsetenv("APP_ENV")
	} else {
		os.Setenv("APP_ENV", value)
	}
	fn()
}
