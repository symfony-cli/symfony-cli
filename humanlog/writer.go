package humanlog

import (
	"io"

	"github.com/pkg/errors"
)

type humanWriter struct {
	w       io.Writer
	handler Handler
}

func New(w io.Writer, opts *Options) *humanWriter {
	return &humanWriter{
		w:       w,
		handler: Handler{opts: opts},
	}
}

func (h *humanWriter) Write(p []byte) (int, error) {
	n, err := h.w.Write(h.handler.Prettify(p))
	if err != nil {
		return n, errors.WithStack(err)
	}
	n, err = h.w.Write([]byte{'\n'})
	if err != nil {
		return n, errors.WithStack(err)
	}
	return len(p), nil
}

func (h *humanWriter) WriteString(s string) (n int, err error) {
	return h.Write([]byte(s))
}
