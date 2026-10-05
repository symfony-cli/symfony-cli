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

package php

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/mitchellh/go-homedir"
	. "gopkg.in/check.v1"
)

type PHPSuite struct{}

var _ = Suite(&PHPSuite{})

func (s *PHPSuite) TestPhpAddslashes(c *C) {
	c.Assert(addslashes.Replace("foo"), Equals, "foo")
	c.Assert(addslashes.Replace("foo'bar"), Equals, "foo\\'bar")
	c.Assert(addslashes.Replace("foo\"bar"), Equals, "foo\"bar")
	c.Assert(addslashes.Replace("foo\\bar"), Equals, "foo\\\\bar")
	c.Assert(addslashes.Replace(`"hello"`), Equals, `"hello"`)
}

func (s *PHPSuite) TestServerCmdHookLoadsProjectPhpIni(c *C) {
	defer restoreExecCommand()
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "echo-arg", "/opt/test/conf.d")
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	home, err := filepath.Abs("testdata/executor")
	c.Assert(err, IsNil)
	homedir.Reset()
	os.Setenv("HOME", home)
	defer homedir.Reset()

	projectDir := filepath.Join(home, "project")
	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(projectDir)
	defer cleanupExecutorTempFiles()

	iniPath := filepath.Join(projectDir, "php.ini")
	c.Assert(os.WriteFile(iniPath, []byte("memory_limit = -1\n"), 0644), IsNil)
	defer os.Remove(iniPath)

	e := &Executor{BinName: "php", Args: []string{"php"}, scriptDir: projectDir}
	defer e.CleanupTemporaryDirectories()
	cmd := &exec.Cmd{}
	c.Assert(serverCmdHook(e, projectDir, []string{"FOO=bar"})(cmd), IsNil)

	c.Check(slices.Contains(cmd.Env, "PHP_INI_SCAN_DIR=/opt/test/conf.d"+string(os.PathListSeparator)+projectDir), Equals, true)
	c.Check(slices.Contains(cmd.Env, "FOO=bar"), Equals, true)
	c.Check(cmd.Dir, Equals, projectDir)
}
