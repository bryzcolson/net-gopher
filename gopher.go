package gopher

import (
	"context"
	"fmt"
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

var (
	DefaultHost = "error.host"
	DefaultPort = "70"
)

func (i *Item) String() string {
	host := i.Host
	if host == "" {
		host = DefaultHost
	}

	port := i.Port
	if port == "" {
		port = DefaultPort
	}

	return fmt.Sprintf("%c%s\t%s\t%s\t%s\r\n",
		i.Type, i.Display, i.Selector, host, port)
}

// ResponseWriter writes responses to a Gopher request.
type ResponseWriter interface {
	// Write writes raw bytes to the connection.
	Write([]byte) (int, error)

	// WriteItem writes a single menu item.
	WriteItem(item *Item) error

	// WriteInfo is a convenience method for writing an info item.
	WriteInfo(text string) error

	// WriteDirectory is a convenience method for writing
	// multiple items to a menu/gophermap.
	WriteDirectory(items []*Item) error
}

type responseWriter struct {
	conn net.Conn
	host string
	port string
}

func (w *responseWriter) Write(p []byte) (int, error) {
	return w.conn.Write(p)
}

func (w *responseWriter) WriteItem(item *Item) error {
	if item.Host == "" {
		item.Host = w.host
	}
	if item.Port == "" {
		item.Port = w.port
	}

	_, err := w.Write([]byte(item.String()))
	return err
}

func (w *responseWriter) WriteInfo(text string) error {
	item := &Item{
		Type:     TypeInfo,
		Display:  text,
		Selector: "",
		Host:     w.host,
		Port:     w.port,
	}
	return w.WriteItem(item)
}

func (w *responseWriter) WriteDirectory(items []*Item) error {
	for _, item := range items {
		if err := w.WriteItem(item); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte(".\r\n"))
	return err
}
