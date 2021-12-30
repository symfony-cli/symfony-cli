package envs

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ScenvSuite struct{}

var _ = Suite(&ScenvSuite{})

func (s *ScenvSuite) TestAppID(c *C) {
	c.Assert(appID("testdata/project"), Equals, "01B7C15FQN1DFPQ4H2CJ66YFPM")
	c.Assert(appID("testdata/project_without_id"), Equals, "")
	c.Assert(appID("testdata/project_without_composer"), Equals, "")
	c.Assert(appID("testdata/project_with_borked_composer"), Equals, "")
}
