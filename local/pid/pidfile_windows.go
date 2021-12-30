package pid

import (
	"os"

	"github.com/pkg/errors"
)

func kill(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(p.Kill())
}
