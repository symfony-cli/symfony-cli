package php

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
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

	// The CGI spec doesn't allow chunked requests. Go is already assembling the
	// chunks from the request to a usable Reader (see net/http.readTransfer and
	// net/http/internal.NewChunkedReader), so the only thing we have to
	// do to is get the content length and add it to the header but to do so we
	// have to read and buffer the body content.
	if len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked" {
		bodyBuffer := &bytes.Buffer{}
		bodyBytes, err := io.Copy(bodyBuffer, req.Body)
		if err != nil {
			return nil, err
		}

		req.Body = io.NopCloser(bodyBuffer)
		req.TransferEncoding = nil
		env["CONTENT_LENGTH"] = strconv.FormatInt(bodyBytes, 10)
		env["HTTP_CONTENT_LENGTH"] = env["CONTENT_LENGTH"]
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
