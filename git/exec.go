package git

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

func execGitQuiet(cwd string, args ...string) (*bytes.Buffer, error) {
	return doExecGit(cwd, args, true)
}

func execGit(cwd string, args ...string) error {
	_, err := doExecGit(cwd, args, false)
	return err
}

func doExecGit(cwd string, args []string, quiet bool) (*bytes.Buffer, error) {
	var out bytes.Buffer
	cmd := exec.Command("git", args...)
	if quiet {
		cmd.Stdout = &out
		cmd.Stderr = &out
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = &gitOutputWriter{output: terminal.Stdout}
		cmd.Stderr = os.Stderr
	}

	if cwd != "" {
		cmd.Dir = cwd
	}

	err := cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
			if waitStatus.ExitStatus() == 1 {
				return &out, errors.Errorf("Command failed")
			}
		}
		return &out, errors.WithStack(err)
	}

	return &out, nil
}

type gitOutputWriter struct {
	output io.Writer

	// Internal state
	buf bytes.Buffer
}

func (w *gitOutputWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if err != nil {
		return n, errors.WithStack(err)
	}

	return n, w.scan()
}

func (w *gitOutputWriter) scan() error {
	for {
		b := w.buf.Bytes()
		// no new line, let's reset the buffer to save some memory
		if len(b) == 0 {
			w.buf.Reset()
			return nil
		}

		pos := bytes.IndexAny(b, "\r\n")
		// incomplete line, let's discard everything we already read to save
		// some memory
		if pos == -1 {
			w.buf.Truncate(len(b))
			return nil
		}

		b = w.buf.Next(pos + 1)
		if len(b) > 1 {
			if _, err := w.output.Write([]byte("  ")); err != nil {
				return errors.WithStack(err)
			}
		}

		if _, err := w.output.Write(b); err != nil {
			return errors.WithStack(err)
		}
	}
}
