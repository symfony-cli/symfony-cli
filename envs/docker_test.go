package envs

import (
	. "gopkg.in/check.v1"
)

type DockerSuite struct{}

var _ = Suite(&DockerSuite{})

func (s *DockerSuite) TestNormalizeDockerComposeProjectName(c *C) {
	for _, testCase := range []struct {
		ProjectName, Expected, ExpectedLegacy string
	}{
		{"foo", "foo", "foo"},
		{"simple-composefile", "simple-composefile", "simplecomposefile"},
		{"multiple-compose-files", "multiple-compose-files", "multiplecomposefiles"},
		{"MyProject", "myproject", "myproject"},
		{"MyProject2", "myproject2", "myproject2"},
		{"symfony.com", "symfonycom", "symfonycom"},
		{"oss-websites", "oss-websites", "osswebsites"},
		{"symfony-dev", "symfony-dev", "symfonydev"},
	} {
		c.Check(normalizeDockerComposeProjectName(testCase.ProjectName), Equals, testCase.Expected)
		c.Check(normalizeDockerComposeProjectNameLegacy(testCase.ProjectName), Equals, testCase.ExpectedLegacy)
	}
}
