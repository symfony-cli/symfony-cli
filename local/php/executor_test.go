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
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	. "gopkg.in/check.v1"
)

type ExecutorSuite struct{}

var _ = Suite(&ExecutorSuite{})

// Modify os.Stdout to write to the given buffer.
func testStdoutCapture(c *C, dst io.Writer) func() {
	r, w, err := os.Pipe()
	if err != nil {
		c.Fatalf("err: %s", err)
	}

	// Modify stdout
	old := os.Stdout
	os.Stdout = w

	// Copy
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		defer r.Close()
		io.Copy(dst, r)
	}()

	return func() {
		// Close the writer end of the pipe
		if err := w.Close(); err != nil {
			c.Errorf("err: %s", err)
		}

		// Reset stdout
		os.Stdout = old

		// Wait for the data copy to complete to avoid a race reading data
		<-doneCh
	}
}

func restoreExecCommand() {
	execCommand = exec.Command
}

func fakeExecCommand(cmd string, args ...string) {
	execCommand = func(string, ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", cmd}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		// Set the working directory right now so that it can be changed by
		// calling test case
		cmd.Dir, _ = os.Getwd()
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Unsetenv("GO_WANT_HELPER_PROCESS")

	switch os.Args[3] {
	case "dump-env":
		fmt.Println(os.Getwd())
		for _, v := range os.Environ() {
			fmt.Println(v)
		}
		os.Exit(0)
	case "exit-code":
		code, _ := strconv.Atoi(os.Args[4])
		os.Exit(code)
	}
	os.Exit(1)
}

func (s *ExecutorSuite) TestNotEnoughArgs(c *C) {
	defer cleanupExecutorTempFiles()

	c.Assert((&Executor{BinName: "php"}).Execute(true), Equals, 1)
}

func (s *ExecutorSuite) TestCommandLineFormatting(c *C) {
	c.Assert((&Executor{}).CommandLine(), Equals, "")

	c.Assert((&Executor{Args: []string{"php"}}).CommandLine(), Equals, "php")

	c.Assert((&Executor{Args: []string{"php", "-dmemory_limit=-1", "/path/to/composer.phar"}}).CommandLine(), Equals, "php -dmemory_limit=-1 /path/to/composer.phar")
}

func (s *ExecutorSuite) TestForwardExitCode(c *C) {
	defer restoreExecCommand()
	fakeExecCommand("exit-code", "5")

	home, err := filepath.Abs("testdata/executor")
	c.Assert(err, IsNil)

	homedir.Reset()
	os.Setenv("HOME", home)
	defer homedir.Reset()

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(filepath.Join(home, "project"))
	defer cleanupExecutorTempFiles()

	c.Assert((&Executor{BinName: "php", Args: []string{"php"}}).Execute(true), Equals, 5)
}

func (s *ExecutorSuite) TestExecutorRunsPHP(c *C) {
	defer restoreExecCommand()
	execCommand = func(name string, arg ...string) *exec.Cmd {
		c.Assert(name, Equals, "../bin/php")

		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "exit-code", "0")
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		// Set the working directory right now so that it can be changed by
		// calling test case
		cmd.Dir, _ = os.Getwd()
		return cmd
	}

	home, err := filepath.Abs("testdata/executor")
	c.Assert(err, IsNil)

	homedir.Reset()
	os.Setenv("HOME", home)
	defer homedir.Reset()

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(filepath.Join(home, "project"))
	defer cleanupExecutorTempFiles()

	c.Assert((&Executor{BinName: "php", Args: []string{"php"}}).Execute(true), Equals, 0)

}

func (s *ExecutorSuite) TestBinaryOtherThanPhp(c *C) {
	defer restoreExecCommand()
	execCommand = func(name string, arg ...string) *exec.Cmd {
		c.Assert(name, Equals, "not-php")

		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "exit-code", "0")
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		// Set the working directory right now so that it can be changed by
		// calling test case
		cmd.Dir, _ = os.Getwd()
		return cmd
	}

	home, err := filepath.Abs("testdata/executor")
	c.Assert(err, IsNil)

	homedir.Reset()
	os.Setenv("HOME", home)
	defer homedir.Reset()

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(filepath.Join(home, "project"))
	defer cleanupExecutorTempFiles()

	c.Assert((&Executor{BinName: "php", Args: []string{"not-php"}}).Execute(true), Equals, 0)
}

func (s *ExecutorSuite) TestEnvInjection(c *C) {
	defer restoreExecCommand()
	fakeExecCommand("dump-env")

	home, err := filepath.Abs("testdata/executor")
	c.Assert(err, IsNil)

	homedir.Reset()
	os.Setenv("HOME", home)
	defer homedir.Reset()

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(filepath.Join(home, "project"))

	os.Rename("git", ".git")
	defer func() {
		// handling error is not really worth it here: we could not really recover it anyway and the original directory
		// is commited
		_ = os.Rename(".git", "git")
	}()
	defer cleanupExecutorTempFiles()

	var output bytes.Buffer
	outCloser := testStdoutCapture(c, &output)
	c.Assert((&Executor{BinName: "php", Args: []string{"php"}}).Execute(true), Equals, 0)
	outCloser()
	// Nothing should be injected by default as tunnel is not open
	c.Check(false, Equals, strings.Contains(output.String(), "DATABASE_URL=pgsql://127.0.0.1:30000"))
	// But .env should be properly loaded
	c.Check(true, Equals, strings.Contains(output.String(), "USER_DEFINED_ENVVAR=foobar"))
	c.Check(true, Equals, strings.Contains(output.String(), "DATABASE_URL=mysql://127.0.0.1"))
	// Checks local properly feed Symfony with SYMFONY_DOTENV_VARS
	c.Check(true, Equals, strings.Contains(output.String(), "SYMFONY_DOTENV_VARS=DATABASE_URL,USER_DEFINED_ENVVAR") || strings.Contains(output.String(), "SYMFONY_DOTENV_VARS=USER_DEFINED_ENVVAR,DATABASE_URL"))

	// change the project name to get exposed env vars
	projectFile := filepath.Join(".platform", "local", "project.yaml")
	contents, err := os.ReadFile(projectFile)
	c.Assert(err, IsNil)
	defer func() {
		// handling error is not really worth it here: we could not really recover it and anyway the original file
		// content is commited
		_ = os.WriteFile(projectFile, contents, 0644)
	}()
	c.Assert(os.WriteFile(projectFile, bytes.Replace(contents, []byte("bew7pfa7t2ut2"), []byte("aew7pfa7t2ut2"), 1), 0644), IsNil)

	output.Reset()
	outCloser = testStdoutCapture(c, &output)
	c.Assert((&Executor{BinName: "php", Args: []string{"php"}}).Execute(true), Equals, 0)
	outCloser()

	// Now overridden, check tunnel information is properly loaded
	c.Check(true, Equals, strings.Contains(output.String(), "DATABASE_URL=postgres://main:main@127.0.0.1:30001/main?sslmode=disable&charset=utf8&serverVersion=13"))
	// And checks .env keeps being properly loaded
	c.Check(true, Equals, strings.Contains(output.String(), "USER_DEFINED_ENVVAR=foobar"))
	// But do not override tunnel information
	c.Check(false, Equals, strings.Contains(output.String(), "DATABASE_URL=mysql://127.0.0.1"))
	// Checks local properly feed Symfony with SYMFONY_DOTENV_VARS
	c.Check(true, Equals, strings.Contains(output.String(), "SYMFONY_DOTENV_VARS=USER_DEFINED_ENVVAR"))

	// When a variable is already set, the value should be kept
	os.Setenv("USER_DEFINED_ENVVAR", "custom")
	defer os.Unsetenv("USER_DEFINED_ENVVAR")

	// Exception: Variables relating to the selected PHP version and INI scan directory are overridden
	os.Setenv("PHP_INI_SCAN_DIR", "test")
	defer os.Unsetenv("PHP_INI_SCAN_DIR")

	os.Setenv("PHP_PATH", "test")
	defer os.Unsetenv("PHP_PATH")

	iniScanDir := filepath.Join(home, "project")
	_, err = os.Create(filepath.Join(iniScanDir, "php.ini"))
	c.Assert(err, IsNil)
	defer os.Remove(filepath.Join(iniScanDir, "php.ini"))

	output.Reset()
	outCloser = testStdoutCapture(c, &output)
	c.Assert((&Executor{BinName: "php", Args: []string{"php"}}).Execute(true), Equals, 0)
	outCloser()

	c.Check(true, Equals, strings.Contains(output.String(), "USER_DEFINED_ENVVAR=custom"))
	c.Check(true, Equals, strings.Contains(output.String(), "PHP_INI_SCAN_DIR=test"+string(os.PathListSeparator)+iniScanDir))
	c.Check(true, Equals, strings.Contains(output.String(), "PHP_PATH="+filepath.FromSlash("../bin/php")))
}

func (s *PHPSuite) TestDetectScript(c *C) {
	phpgo, err := filepath.Abs("php.go")
	c.Assert(err, IsNil)
	phpgo = filepath.Dir(phpgo)
	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{"php.go"}, phpgo},
		{[]string{"-fphp.go"}, phpgo},
		{[]string{"-f", "php.go"}, phpgo},
		{[]string{"-c", "php.ini", "php.go"}, phpgo},
		{[]string{"-c", "php.ini", "-f", "php.go"}, phpgo},
		{[]string{"-c", "php.ini", "-fphp.go"}, phpgo},
		{[]string{"-cphp.ini", "php.go"}, phpgo},
		{[]string{"-l", "php.go"}, phpgo},
		{[]string{"-cphp.ini", "-l", "-z", "foo", "php.go"}, phpgo},
		{[]string{"php.go", "--", "foo.go"}, phpgo},
	}
	for _, test := range tests {
		c.Assert(detectScriptDir(test.args), Equals, test.expected)
	}
}

func cleanupExecutorTempFiles() {
	os.RemoveAll("testdata/executor/.symfony5/tmp")
	os.RemoveAll("testdata/executor/.symfony5/var")
}
