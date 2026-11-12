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
	"os/exec"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type DotEnvSuite struct {
	php         string
	previousEnv map[string]*string
}

var _ = Suite(&DotEnvSuite{})

func (s *DotEnvSuite) SetUpTest(c *C) {
	s.previousEnv = map[string]*string{}
	if s.php == "" {
		php, err := exec.LookPath("php")
		if err != nil {
			c.Skip("PHP is required to run Dotenv")
		}
		s.php = php
	}
	vendorDir, err := filepath.Abs("testdata/dotenv/vendor")
	c.Assert(err, IsNil)
	s.setEnv(c, map[string]string{"COMPOSER_VENDOR_DIR": vendorDir, "APP_ENV": "", "SYMFONY_DOTENV_VARS": ""})
	os.Unsetenv("APP_ENV")
}

func (s *DotEnvSuite) phpBinary() (string, error) {
	return s.php, nil
}

func (s *DotEnvSuite) TestLoadDotEnv(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{
		".env":       "APP_ENV=dev\nEXPORTED=dotenv\nFOO=env\nBAR=env\n",
		".env.local": "BAR=local\n",
		".env.dev":   "BAZ=dev\n",
	})
	s.setEnv(c, map[string]string{"EXPORTED": "exported"})

	vars, err := LoadDotEnv(map[string]string{}, dir, s.phpBinary)

	c.Assert(err, IsNil)
	c.Check(vars, DeepEquals, map[string]string{
		"APP_ENV":             "dev",
		"FOO":                 "env",
		"BAR":                 "local",
		"BAZ":                 "dev",
		"SYMFONY_DOTENV_VARS": "FOO,BAR,BAZ",
	})
}

func (s *DotEnvSuite) TestLoadDotEnvFromSubdirectory(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{".env": "FOO=env\n"})
	subdir := filepath.Join(dir, "bin")
	c.Assert(os.Mkdir(subdir, 0755), IsNil)

	vars, err := LoadDotEnv(map[string]string{}, subdir, s.phpBinary)

	c.Assert(err, IsNil)
	c.Check(vars["FOO"], Equals, "env")
}

func (s *DotEnvSuite) TestLoadDotEnvKeepsComputedVariables(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{".env": "DATABASE_URL=dotenv\nFOO=dotenv\n"})
	// a parent process loaded both from .env files
	s.setEnv(c, map[string]string{"DATABASE_URL": "parent", "FOO": "parent", "SYMFONY_DOTENV_VARS": "DATABASE_URL,FOO"})

	vars, err := LoadDotEnv(map[string]string{"DATABASE_URL": "docker"}, dir, s.phpBinary)

	c.Assert(err, IsNil)
	c.Check(vars["DATABASE_URL"], Equals, "docker")
	c.Check(vars["FOO"], Equals, "dotenv")
	c.Check(vars["SYMFONY_DOTENV_VARS"], Equals, "FOO")
}

func (s *DotEnvSuite) TestLoadDotEnvKeepsInheritedAppEnvListed(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{".env": "APP_ENV=dev\n", ".env.dev": "FOO=dev\n"})
	s.setEnv(c, map[string]string{"APP_ENV": "prod", "SYMFONY_DOTENV_VARS": "APP_ENV"})

	vars, err := LoadDotEnv(map[string]string{}, dir, s.phpBinary)

	c.Assert(err, IsNil)
	c.Check(vars["APP_ENV"], Equals, "dev")
	c.Check(vars["FOO"], Equals, "dev")
	c.Check(vars["SYMFONY_DOTENV_VARS"], Equals, "APP_ENV,FOO")
}

func (s *DotEnvSuite) TestLoadDotEnvWithoutDotEnvFiles(c *C) {
	vars, err := LoadDotEnv(map[string]string{}, c.MkDir(), failingPHPBinary(c))

	c.Assert(err, IsNil)
	c.Check(vars, DeepEquals, map[string]string{"SYMFONY_DOTENV_VARS": ""})
}

func (s *DotEnvSuite) TestLoadDotEnvWithoutDotenvComponent(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{".env": "FOO=env\n"})
	s.setEnv(c, map[string]string{"COMPOSER_VENDOR_DIR": c.MkDir()})

	vars, err := LoadDotEnv(map[string]string{}, dir, failingPHPBinary(c))

	c.Assert(err, IsNil)
	c.Check(vars, DeepEquals, map[string]string{"SYMFONY_DOTENV_VARS": ""})
}

func (s *DotEnvSuite) TestLoadDotEnvReportsDotenvErrors(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{".env": "FOO=env\nINVALID\n"})

	vars, err := LoadDotEnv(map[string]string{"DATABASE_URL": "docker"}, dir, s.phpBinary)

	c.Assert(err, ErrorMatches, `unable to load the \.env files: Missing = in ".*\.env"\.`)
	c.Check(vars, DeepEquals, map[string]string{"DATABASE_URL": "docker", "SYMFONY_DOTENV_VARS": ""})
}

func (s *DotEnvSuite) TestLookupEnv(c *C) {
	dir := writeDotEnvFiles(c, map[string]string{".env": "FOO=env\nEXPORTED=env\n"})
	s.setEnv(c, map[string]string{"EXPORTED": "exported", "UNRELATED": "exported"})

	for key, expected := range map[string]string{"FOO": "env", "EXPORTED": "exported"} {
		value, ok := LookupEnv(dir, key, s.phpBinary)
		c.Check(ok, Equals, true, Commentf(key))
		c.Check(value, Equals, expected, Commentf(key))
	}

	value, ok := LookupEnv(dir, "UNRELATED", failingPHPBinary(c))
	c.Check(ok, Equals, true)
	c.Check(value, Equals, "exported")
	_, ok = LookupEnv(dir, "UNDEFINED", failingPHPBinary(c))
	c.Check(ok, Equals, false)
}

func (s *DotEnvSuite) TestComposerVendorDir(c *C) {
	dir := c.MkDir()
	os.Unsetenv("COMPOSER_VENDOR_DIR")
	c.Check(composerVendorDir(dir), Equals, filepath.Join(dir, "vendor"))

	c.Assert(os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"config": {"vendor-dir": "libs"}}`), 0644), IsNil)
	c.Check(composerVendorDir(dir), Equals, filepath.Join(dir, "libs"))

	s.setEnv(c, map[string]string{"COMPOSER_VENDOR_DIR": "deps"})
	c.Check(composerVendorDir(dir), Equals, filepath.Join(dir, "deps"))
}

func writeDotEnvFiles(c *C, files map[string]string) string {
	dir := c.MkDir()
	for name, content := range files {
		c.Assert(os.WriteFile(filepath.Join(dir, name), []byte(content), 0644), IsNil)
	}
	return dir
}

func failingPHPBinary(c *C) PHPBinary {
	return func() (string, error) {
		c.Error("PHP must not run")
		return "", os.ErrNotExist
	}
}

// setEnv sets environment variables until the end of the test.
func (s *DotEnvSuite) setEnv(c *C, vars map[string]string) {
	for k, v := range vars {
		if _, recorded := s.previousEnv[k]; !recorded {
			if previous, ok := os.LookupEnv(k); ok {
				s.previousEnv[k] = &previous
			} else {
				s.previousEnv[k] = nil
			}
		}
		c.Assert(os.Setenv(k, v), IsNil)
	}
}

func (s *DotEnvSuite) TearDownTest(c *C) {
	for k, v := range s.previousEnv {
		if v == nil {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, *v)
		}
	}
}
