package php

import (
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	fcgiclient "github.com/symfony-cli/symfony-cli/local/fcgi_client"
)

type cgiTransport struct{}

func (p *cgiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	env := req.Context().Value(environmentContextKey).(map[string]string)

	// as the process might have been just created, it might not be ready yet
	var fcgi *fcgiclient.FCGIClient
	var err error
	max := 10
	i := 0
	for {
		if fcgi, err = fcgiclient.Dial("tcp", "127.0.0.1:"+req.URL.Port()); err == nil {
			break
		}
		i++
		if i > max {
			return nil, errors.Wrapf(err, "unable to connect to the PHP FastCGI process")
		}
		time.Sleep(time.Millisecond * 50)
	}

	// fetching the response from the fastcgi backend, and check for errors
	resp, err := fcgi.Request(env, req.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch the response from the backend")
	}
	resp.Body = cgiBodyReadCloser{resp.Body, fcgi}
	resp.Request = req

	return resp, nil
}

// cgiBodyReadCloser is responsible for postponing the CGI connection
// termination when the client finished reading the response. This effectively
// allows to "stream" the CGI response from the server to the client by removing
// the requirement for an in-between buffer.
type cgiBodyReadCloser struct {
	io.Reader
	*fcgiclient.FCGIClient
}

func (f cgiBodyReadCloser) Close() error {
	f.FCGIClient.Close()
	return nil
}
