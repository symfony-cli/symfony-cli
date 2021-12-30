package proxy

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ProxySuite struct{}

var _ = Suite(&ProxySuite{})

func (s *ProxySuite) TestGetDir(c *C) {
	p := &Config{
		domains: map[string]string{
			"symfony.com":        "symfony_com",
			"*.symfony.com":      "any_symfony_com",
			"*.live.symfony.com": "any_live_symfony_com",
		},
	}
	c.Assert(p.GetDir("symfony.com"), Equals, "symfony_com")
	c.Assert(p.GetDir("foo.symfony.com"), Equals, "any_symfony_com")
	c.Assert(p.GetDir("foo.live.symfony.com"), Equals, "any_live_symfony_com")
}
