/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

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

type fakeEnv struct {
	Rels     Relationships
	RootPath string
}

func (f fakeEnv) Path() string {
	if f.RootPath != "" {
		return f.RootPath
	}
	return "/dev/null"
}

func (f fakeEnv) Mailer() Envs {
	return nil
}

func (f fakeEnv) Language() string {
	return "php"
}

func (f fakeEnv) Relationships() Relationships {
	return f.Rels
}

func (f fakeEnv) Extra() Envs {
	return nil
}

func (f fakeEnv) Local() bool {
	return true
}

func (s *ScenvSuite) TestElasticsearchURLEndsWithTrailingSlash(c *C) {
	env := fakeEnv{
		Rels: map[string][]map[string]interface{}{
			"elasticsearch": {
				map[string]interface{}{
					"host":   "localhost",
					"ip":     "localhost",
					"port":   9200,
					"path":   "/",
					"rel":    "elasticsearch",
					"scheme": "http",
				},
			},
		},
	}

	rels := extractRelationshipsEnvs(env)
	c.Assert(rels["ELASTICSEARCH_URL"], Equals, "http://localhost:9200/")

	// We want to stay backward compatible with Platform.sh/SymfonyCloud
	env.Rels["elasticsearch"][0]["path"] = nil
	rels = extractRelationshipsEnvs(env)
	c.Assert(rels["ELASTICSEARCH_URL"], Equals, "http://localhost:9200")

	delete(env.Rels["elasticsearch"][0], "path")
	rels = extractRelationshipsEnvs(env)
	c.Assert(rels["ELASTICSEARCH_URL"], Equals, "http://localhost:9200")
}

func (s *ScenvSuite) TestDockerDatabaseURLs(c *C) {
	env := fakeEnv{
		Rels: map[string][]map[string]interface{}{
			"mysql": {
				map[string]interface{}{
					"host":     "127.0.0.1",
					"ip":       "127.0.0.1",
					"password": "!ChangeMe!",
					"path":     "root",
					"port":     "56614",
					"query":    map[string]bool{"is_master": true},
					"rel":      "mysql",
					"scheme":   "mysql",
					"username": "root",
					"version":  "1:10.0.38+maria-1~xenial",
				},
			},
			"postgresql": {
				map[string]interface{}{
					"host":     "127.0.0.1",
					"ip":       "127.0.0.1",
					"password": "main",
					"path":     "main",
					"port":     "63574",
					"query":    map[string]bool{"is_master": true},
					"rel":      "pgsql",
					"scheme":   "pgsql",
					"username": "main",
					"version":  "13.13",
				},
			},
		},
	}

	rels := extractRelationshipsEnvs(env)
	c.Assert(rels["MYSQL_URL"], Equals, "mysql://root:!ChangeMe!@127.0.0.1:56614/root?sslmode=disable&charset=utf8mb4&serverVersion=10.0.38+maria-1~xenial")
	c.Assert(rels["POSTGRESQL_URL"], Equals, "postgres://main:main@127.0.0.1:63574/main?sslmode=disable&charset=utf8&serverVersion=13.13")
}

func (s *ScenvSuite) TestCloudTunnelDatabaseURLs(c *C) {
	env := fakeEnv{
		Rels: map[string][]map[string]interface{}{
			"mysql": {
				{
					"cluster":      "d3xkaapt4cyik-main-bvxea6i",
					"epoch":        0,
					"fragment":     interface{}(nil),
					"host":         "127.0.0.1",
					"host_mapped":  false,
					"hostname":     "vd4wb3toqpyybms2qktcjmdng4.database.service._.eu-5.platformsh.site",
					"instance_ips": []interface{}{"249.175.144.213"},
					"ip":           "127.0.0.1",
					"password":     "",
					"path":         "main",
					"port":         "30001",
					"public":       false,
					"query":        map[string]interface{}{"is_master": true},
					"rel":          "mysql",
					"scheme":       "mysql",
					"service":      "database",
					"type":         "mysql:10.0",
					"username":     "user",
				},
			},
			"postgresql": {
				{
					"cluster":      "xxx-master-yyy",
					"epoch":        0,
					"fragment":     interface{}(nil),
					"host":         "127.0.0.1",
					"host_mapped":  false,
					"hostname":     "xxx.pgsqldb.service._.fr-4.platformsh.site",
					"instance_ips": []interface{}{"240.7.208.71"},
					"ip":           "127.0.0.1",
					"password":     "main",
					"path":         "main",
					"port":         "30000",
					"public":       false,
					"query":        map[string]interface{}{"is_master": true},
					"rel":          "postgresql",
					"scheme":       "pgsql",
					"service":      "pgsqldb",
					"type":         "postgresql:13",
					"username":     "main",
				},
			},
		},
	}

	rels := extractRelationshipsEnvs(env)
	c.Assert(rels["MYSQL_URL"], Equals, "mysql://user@127.0.0.1:30001/main?sslmode=disable&charset=utf8mb4&serverVersion=10.0.0-MariaDB")
	c.Assert(rels["POSTGRESQL_URL"], Equals, "postgres://main:main@127.0.0.1:30000/main?sslmode=disable&charset=utf8&serverVersion=13")
}

func (s *ScenvSuite) TestDoctrineConfigTakesPrecedenceDatabaseURLs(c *C) {
	env := fakeEnv{
		Rels: map[string][]map[string]interface{}{
			"mysql": {
				{
					"cluster":      "d3xkaapt4cyik-main-bvxea6i",
					"epoch":        0,
					"fragment":     interface{}(nil),
					"host":         "127.0.0.1",
					"host_mapped":  false,
					"hostname":     "vd4wb3toqpyybms2qktcjmdng4.database.service._.eu-5.platformsh.site",
					"instance_ips": []interface{}{"249.175.144.213"},
					"ip":           "127.0.0.1",
					"password":     "",
					"path":         "main",
					"port":         "30001",
					"public":       false,
					"query":        map[string]interface{}{"is_master": true},
					"rel":          "mysql",
					"scheme":       "mysql",
					"service":      "database",
					"type":         "mysql:10.0",
					"username":     "user",
				},
			},
		},
		RootPath: "testdata/doctrine-project",
	}

	rels := extractRelationshipsEnvs(env)
	c.Assert(rels["MYSQL_URL"], Equals, "mysql://user@127.0.0.1:30001/main?sslmode=disable&charset=utf8mb4&serverVersion=8.0.33")
}
