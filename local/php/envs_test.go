package php

import (
	"net/http"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type PHPFPMSuite struct{}

var _ = Suite(&PHPFPMSuite{})

func (s *PHPFPMSuite) TestGenerateEnv(c *C) {
	testdataDir := "testdata"
	tests := []struct {
		uri      string
		passthru string
		expected map[string]string
	}{
		{
			passthru: "/index.php",
			uri:      "/",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/index.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/index.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/index.php/foo",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/index.php/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/app.PHP",
			uri:      "/app.PHP/foo",
			expected: map[string]string{
				"PATH_INFO":       "/foo",
				"REQUEST_URI":     "/app.PHP/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/app.PHP",
				"SCRIPT_NAME":     "/app.PHP",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/foo",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/update.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/update.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/update.php",
				"SCRIPT_NAME":     "/update.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/js/whitelist.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/js/whitelist.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/js/whitelist.php",
				"SCRIPT_NAME":     "/js/whitelist.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/update.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/update.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/update.php",
				"SCRIPT_NAME":     "/update.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/unknown.php",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/unknown.php",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/unknown.php/foo",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/unknown.php/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
		{
			passthru: "/index.php",
			uri:      "/unknown.PHP/foo",
			expected: map[string]string{
				"PATH_INFO":       "",
				"REQUEST_URI":     "/unknown.PHP/foo",
				"QUERY_STRING":    "",
				"SCRIPT_FILENAME": testdataDir + "/public/index.php",
				"SCRIPT_NAME":     "/index.php",
			},
		},
	}
	for _, test := range tests {
		process := &Server{
			projectDir:   testdataDir,
			documentRoot: testdataDir + "/public/",
			passthru:     test.passthru,
		}
		req, err := http.NewRequest("GET", test.uri, nil)
		c.Assert(err, IsNil)

		req.RequestURI = test.uri
		env := process.generateEnv(req)
		for k, v := range test.expected {
			vv, ok := env[k]
			c.Assert(ok, Equals, true)
			c.Assert(vv, DeepEquals, v)
		}
	}
}
