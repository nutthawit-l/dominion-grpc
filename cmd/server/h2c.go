package main

import (
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// h2cMux wraps the mux so it speaks HTTP/2 cleartext (required for
// Connect's grpc wire format on insecure listeners).
func h2cMux(mux *http.ServeMux) http.Handler {
	return h2c.NewHandler(mux, &http2.Server{})
}
