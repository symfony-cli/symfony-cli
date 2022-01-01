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

package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/cert"
	"github.com/symfony-cli/symfony-cli/local/pid"
	. "gopkg.in/check.v1"
)

func (s *ProxySuite) TestProxy(c *C) {
	ca, err := cert.NewCA(filepath.Join("testdata/certs"))
	c.Assert(err, IsNil)
	c.Assert(ca.LoadCA(), IsNil)

	homedir.Reset()
	os.Setenv("HOME", "testdata")
	defer homedir.Reset()
	defer os.RemoveAll("testdata/.symfony5")

	p := New(&Config{
		domains: map[string]string{
			"symfony":        "symfony_com",
			"symfony-no-tls": "symfony_com_no_tls",
			"symfony2":       "symfony_com2",
		},
		TLD:  "wip",
		path: "testdata/.symfony5/proxy.json",
	}, ca, log.New(zerolog.New(os.Stderr), "", 0), true)
	os.MkdirAll("testdata/.symfony5", 0755)
	err = p.Save()
	c.Assert(err, IsNil)

	// Test the 404 fallback
	{
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/foo", nil)
		req.Host = "localhost"
		c.Assert(err, IsNil)
		p.proxy.ServeHTTP(rr, req)
		c.Check(rr.Code, Equals, http.StatusNotFound)
	}

	// Test serving the proxy.pac
	{
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/proxy.pac", nil)
		req.Host = "localhost"
		c.Assert(err, IsNil)
		p.proxy.ServeHTTP(rr, req)
		c.Assert(rr.Code, Equals, http.StatusOK)
		c.Check(rr.Header().Get("Content-type"), Equals, "application/x-ns-proxy-autoconfig")
	}

	// Test serving the index
	{
		rr := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/", nil)
		req.Host = "localhost"
		c.Assert(err, IsNil)
		p.proxy.ServeHTTP(rr, req)
		c.Assert(rr.Code, Equals, http.StatusOK)
		c.Check(strings.Contains(rr.Body.String(), "symfony.wip"), Equals, true)
	}

	// Test the proxy
	frontend := httptest.NewServer(p.proxy)
	defer frontend.Close()
	frontendUrl, _ := url.Parse(frontend.URL)
	cert, err := x509.ParseCertificate(ca.AsTLS().Certificate[0])
	c.Assert(err, IsNil)
	certpool := x509.NewCertPool()
	certpool.AddCert(cert)
	transport := &http.Transport{
		Proxy: http.ProxyURL(frontendUrl),
		TLSClientConfig: &tls.Config{
			RootCAs: certpool,
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   1 * time.Second,
	}

	// Test proxying a request to a non registered project
	{
		req, _ := http.NewRequest("GET", "https://foo.wip/", nil)
		req.Close = true

		res, err := client.Do(req)
		c.Assert(err, IsNil)
		c.Assert(res.StatusCode, Equals, http.StatusNotFound)
		body, _ := ioutil.ReadAll(res.Body)
		c.Check(strings.Contains(string(body), "not linked"), Equals, true)
	}

	// Test proxying a request to a registered project but not started
	{
		req, _ := http.NewRequest("GET", "https://symfony.wip/", nil)
		req.Close = true

		res, err := client.Do(req)
		c.Assert(err, IsNil)
		c.Assert(res.StatusCode, Equals, http.StatusNotFound)
		body, _ := ioutil.ReadAll(res.Body)
		c.Check(strings.Contains(string(body), "not started"), Equals, true)
	}
	/*
		// Test proxying a request to a registered project and started
		{
			backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte(`symfony.wip`))
			}))
			cert, err := ca.CreateCert([]string{"localhost", "127.0.0.1"})
			c.Assert(err, IsNil)
			backend.TLS = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
			backend.StartTLS()
			defer backend.Close()
			backendURL, err := url.Parse(backend.URL)
			c.Assert(err, IsNil)

			p := pid.New("symfony_com", nil)
			port, _ := strconv.Atoi(backendURL.Port())
			p.Write(os.Getpid(), port, "https")

			req, _ := http.NewRequest("GET", "https://symfony.wip/", nil)
			req.Close = true

			res, err := client.Do(req)
			c.Assert(err, IsNil)
			c.Assert(res.StatusCode, Equals, http.StatusOK)
			body, _ := ioutil.ReadAll(res.Body)
			c.Check(string(body), Equals, "symfony.wip")
		}
	*/
	// Test proxying a request to a registered project but no TLS
	{
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`http://symfony-no-tls.wip`))
		}))
		defer backend.Close()
		backendURL, err := url.Parse(backend.URL)
		c.Assert(err, IsNil)

		p := pid.New("symfony_com_no_tls", nil)
		port, _ := strconv.Atoi(backendURL.Port())
		p.Write(os.Getpid(), port, "http")

		req, _ := http.NewRequest("GET", "http://symfony-no-tls.wip/", nil)
		req.Close = true

		res, err := client.Do(req)
		c.Assert(err, IsNil)
		body, _ := ioutil.ReadAll(res.Body)
		c.Assert(res.StatusCode, Equals, http.StatusOK)
		c.Assert(string(body), Equals, "http://symfony-no-tls.wip")
	}

	// Test proxying a request to an outside backend
	{
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer backend.Close()
		req, _ := http.NewRequest("GET", backend.URL, nil)
		req.Close = true

		res, err := client.Do(req)
		c.Assert(err, IsNil)
		c.Assert(res.StatusCode, Equals, http.StatusOK)
	}
	/*
		// Test proxying a request over HTTP2
		http2.ConfigureTransport(transport)
		{
			backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				if r.Proto == "HTTP/2.0" {
					w.Write([]byte(`http2`))
					return
				}
				w.Write([]byte(`symfony.wip`))
			}))
			cert, err := ca.CreateCert([]string{"localhost", "127.0.0.1"})
			c.Assert(err, IsNil)
			backend.TLS = &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2", "http/1.1"},
			}
			backend.StartTLS()
			defer backend.Close()
			backendURL, err := url.Parse(backend.URL)
			c.Assert(err, IsNil)

			p := pid.New("symfony_com2", nil)
			port, _ := strconv.Atoi(backendURL.Port())
			p.Write(os.Getpid(), port, "https")

			req, _ := http.NewRequest("GET", "https://symfony2.wip/", nil)
			req.Close = true

			res, err := client.Do(req)
			c.Assert(err, IsNil)
			c.Assert(res.StatusCode, Equals, http.StatusOK)
			body, _ := ioutil.ReadAll(res.Body)
			c.Check(string(body), Equals, "http2")
		}
	*/
}
