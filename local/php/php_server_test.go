package php

import (
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
