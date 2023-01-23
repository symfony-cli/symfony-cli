package php

import (
	"path/filepath"

	. "gopkg.in/check.v1"
)

type ComposerSuite struct{}

var _ = Suite(&ComposerSuite{})

func (s *ComposerSuite) TestIsComposerPHPScript(c *C) {
	dir, err := filepath.Abs("testdata/php_scripts")
	c.Assert(err, IsNil)

	c.Assert(isPHPScript(filepath.Join(dir, "unknown")), Equals, false)
	c.Assert(isPHPScript(filepath.Join(dir, "invalid")), Equals, false)

	for _, validScripts := range []string{
		"usual-one",
		"debian-style",
		"custom-one",
	} {
		c.Assert(isPHPScript(filepath.Join(dir, validScripts)), Equals, true)
	}
}
