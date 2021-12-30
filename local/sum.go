package local

import (
	"crypto/sha1"
	"fmt"
	"io"
)

func Name(dir string) string {
	h := sha1.New()
	io.WriteString(h, dir)
	return fmt.Sprintf("%x", h.Sum(nil))
}
