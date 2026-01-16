package gopher

import (
	"net"
	"time"
)

// A Client is a Gopher client. Its zero value ([DefaultClient]) is a
// usable client.
type Client struct {
	// Timeout specifies a time limit for requests made by this
	// Client. The timeout includes connection time and reading
	// the response body.
	//
	// A Timeout of zero means no timeout.
	Timeout time.Duration

	// Dialer optionally specifies an alternate dialer
	Dialer *net.Dialer
}
