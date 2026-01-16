package gopher

import (
	"context"
	"io"
	"net"
	"net/url"
	"time"
)

// Gopher item type constants.
const (
	TypeText          = '0'
	TypeDirectory     = '1'
	TypeCCSOPhonebook = '2'
	TypeError         = '3'
	TypeBinHex        = '4'
	TypeDOS           = '5'
	TypeUUEncoded     = '6'
	TypeSearch        = '7'
	TypeTelnet        = '8'
	TypeBinary        = '9'
	TypeGIF           = 'g'
	TypeImage         = 'I'
	TypeHTML          = 'h'
	TypeInfo          = 'i'
)

// Request represents a Gopher request
// received by a server or to be sent by a client.
type Request struct {
	// Selector is the path/resource being requested.
	Selector string

	// Query contains search terms for search servers (Type 7).
	Query string

	// Host specifies the host on which the URL is sought.
	Host string

	// URL specifies either the URI being requested (for server
	// requests) or the URL to access (for client requests).
	URL *url.URL

	// ctx is the request context for cancellation
	ctx context.Context
}

// Response represents a Gopher response.
type Response struct {
	// Body represents the response body.
	Body io.ReadCloser

	// Request is the request that was sent to obtain this response.
	Request *Request

	// conn is the underlying network connection
	conn net.Conn
}

// Client is a Gopher client.
type Client struct {
	// Timeout specifies a time limit for requests
	Timeout time.Duration

	// Dialer optionally specifies an alternate dialer
	Dialer *net.Dialer
}

// Item represents a singe line in a Gopher menu/directory.
type Item struct {
	// Type is the Gopher item type (0, 1, i, etc.)
	Type byte

	// Dislpay is the user-visible text.
	Display string

	// Selector is the path to the resource
	Selector string

	// Host is the server hostname
	Host string

	// Port is the server port
	Port string
}
