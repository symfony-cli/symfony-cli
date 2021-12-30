package process

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Process struct {
	Dir           string
	Path          string
	Args          []string
	Logger        zerolog.Logger
	ProcessLogger func(string)
	Env           []string
}

func (p *Process) Run(ctx context.Context) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, p.Path, p.Args...)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if p.ProcessLogger == nil {
		p.ProcessLogger = func(t string) {
			p.Logger.Info().Msg(t)
		}
	}
	errReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	go func() {
		s := bufio.NewScanner(outReader)
		for s.Scan() {
			p.ProcessLogger(s.Text())
		}
	}()
	go func() {
		s := bufio.NewScanner(errReader)
		for s.Scan() {
			p.ProcessLogger(s.Text())
		}
	}()

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, p.Env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	deathsig(cmd.SysProcAttr)
	if err := cmd.Start(); err != nil {
		return nil, errors.WithStack(err)
	}
	go func() {
		p.Logger.Debug().Msg("started")
		<-ctx.Done()
		kill(cmd)
		p.Logger.Debug().Msg("stopped")
	}()
	return cmd, nil
}
