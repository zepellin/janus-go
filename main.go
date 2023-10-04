package main

import (
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/compute/metadata"
)

// This example demonstrates how to use your own transport when using this package.
func main() {
	client := metadata.NewClient(&http.Client{Timeout: 1 * time.Second, Transport: userAgentTransport{
		userAgent: "my-user-agent",
		base:      http.DefaultTransport,
	}})
	p, err := client.ProjectID()
	if err != nil {
		log.Fatal("Couldn't connect to metadata server")
	}
	_ = p // TODO: Use p.
}

// userAgentTransport sets the User-Agent header before calling base.
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req)
}
