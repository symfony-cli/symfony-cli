package project

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ProjectSuite struct{}

var _ = Suite(&ProjectSuite{})

func (s *ProjectSuite) TestGuessDocumentRoot(c *C) {
	c.Assert(guessDocumentRoot("testdata"), Equals, "testdata/foobar")
}
