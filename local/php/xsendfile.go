package php

import (
	"net/http"
	"os"
)

func (p *Server) processXSendFile(resp *http.Response) (error, bool) {
	// X-SendFile
	sendFilename := resp.Header.Get("X-SendFile")
	if sendFilename == "" {
		return nil, false
	} else if _, err := os.Stat(sendFilename); err != nil {
		return nil, false
	}

	req := resp.Request
	w := req.Context().Value(responseWriterContextKey).(http.ResponseWriter)

	http.ServeFile(w, req, sendFilename)

	return nil, true
}
