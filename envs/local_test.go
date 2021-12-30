package envs

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	. "gopkg.in/check.v1"
)

type LocalSuite struct{}

var _ = Suite(&LocalSuite{})

func (s *LocalSuite) TestExtra(c *C) {
	l := &Local{}
	c.Assert(l.Extra(), DeepEquals, Envs{
		"SYMFONY_TUNNEL":     "",
		"SYMFONY_TUNNEL_ENV": "",
		"SYMFONY_DOCKER_ENV": "",
	})
}

func (s *LocalSuite) TestTunnelFilePath(c *C) {
	l := &Local{Dir: "testdata/project"}
	os.Rename("testdata/project/git", "testdata/project/.git")
	defer func() {
		os.Rename("testdata/project/.git", "testdata/project/git")
	}()
	tunnel := Tunnel{Dir: l.Dir}
	tunnelPath, _ := tunnel.computePath()
	c.Assert(filepath.Base(tunnelPath), Equals, "ism4mega7wpx6-toto--security.json")
}

func (s *LocalSuite) TestRelationships(c *C) {
	os.Rename("testdata/project/git", "testdata/project/.git")
	defer os.Rename("testdata/project/.git", "testdata/project/git")
	homedir.Reset()
	os.Setenv("HOME", "testdata/project")
	defer homedir.Reset()
	l := &Local{Dir: "testdata/project"}
	c.Assert(extractRelationshipsEnvs(l), DeepEquals, Envs{
		"SECURITY_SERVER_HOST":   "127.0.0.1",
		"SECURITY_SERVER_URL":    "http://127.0.0.1:30000",
		"SECURITY_SERVER_SERVER": "http://127.0.0.1:30000",
		"SECURITY_SERVER_PORT":   "30000",
		"SECURITY_SERVER_SCHEME": "http",
		"SECURITY_SERVER_IP":     "127.0.0.1",
		"DATABASE_URL":           "postgres://main:main@127.0.0.1:30001/main?sslmode=disable&charset=utf8&serverVersion=13",
		"DATABASE_HOST":          "127.0.0.1",
		"DATABASE_PORT":          "30001",
		"DATABASE_USER":          "main",
		"DATABASE_USERNAME":      "main",
		"DATABASE_PASSWORD":      "main",
		"DATABASE_SERVER":        "postgres://127.0.0.1:30001",
		"DATABASE_DRIVER":        "postgres",
		"DATABASE_NAME":          "main",
		"DATABASE_DATABASE":      "main",
		"DATABASE_VERSION":       "13",
		"PGPORT":                 "30001",
		"PGPASSWORD":             "main",
		"PGDATABASE":             "main",
		"PGUSER":                 "main",
		"PGHOST":                 "127.0.0.1",
	})
}
