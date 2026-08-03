package fcgiclient

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestRequestContentLength(t *testing.T) {
	tests := []struct {
		name    string
		headers string
		want    int64
	}{
		{
			name:    "unknown length",
			headers: "Content-Type: text/plain\r\n",
			want:    -1,
		},
		{
			name:    "known length",
			headers: "Content-Type: text/plain\r\nContent-Length: 5\r\n",
			want:    5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newResponseClient(t, test.headers+"\r\nhello")

			response, err := client.Request(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if response.ContentLength != test.want {
				t.Fatalf("got content length %d, want %d", response.ContentLength, test.want)
			}
		})
	}
}

func newResponseClient(t *testing.T, response string) *FCGIClient {
	t.Helper()

	var stream bytes.Buffer
	responseHeader := header{}
	responseHeader.init(FCGI_STDOUT, 1, len(response))
	if err := binary.Write(&stream, binary.BigEndian, responseHeader); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.WriteString(response); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Write(make([]byte, responseHeader.PaddingLength)); err != nil {
		t.Fatal(err)
	}

	return &FCGIClient{
		rwc:   &responseReadWriteCloser{Reader: &stream},
		reqId: 1,
	}
}

type responseReadWriteCloser struct {
	io.Reader
}

func (responseReadWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (responseReadWriteCloser) Close() error {
	return nil
}
