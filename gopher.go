// Package gopher provides Gopher client and server implementations.
//
// [Get] makes Gopher requests:
//
//	resp, err := gopher.Get("gopher://example.com/")
//
// The caller must close the response when finished with it:
//
//	resp, err := gopher.Get("gopher://example.com/")
//	if err != nil {
//		// handle error
//	}
//	defer resp.Body.Close()
//	body, err := io.ReadAll(resp.Body)
//
// For control over client behavior, create a [Client]:
//
//	client := &gopher.Client{
//		Timeout: 10 * time.Second,
//	}
//	resp, err := client.Get("gopher://example.com/")
//
// [ListenAndServe] starts a Gopher server with a given address and handler.
// The handler is usually nil, which means to use [DefaultServeMux].
// [Handle] and [HandleFunc] add handlers to [DefaultServeMux]:
//
//	gopher.HandleFunc("/", func(w gopher.ResponseWriter, r *gopher.Request) {
//	    w.WriteInfo("Welcome to the Gopher server!")
//	    w.WriteItem(&gopher.Item{Type: gopher.TypeDirectory, Display: "About", Selector: "/about"})
//	    w.WriteItem(&gopher.Item{Type: gopher.TypeText, Display: "README", Selector: "/readme"})
//	})
//
//	gopher.HandleFunc("/about", func(w gopher.ResponseWriter, r *gopher.Request) {
//	    w.WriteInfo("This is an example Gopher server")
//	    w.WriteInfo("Built with net-gopher")
//	})
//
//	gopher.HandleFunc("/readme", func(w gopher.ResponseWriter, r *gopher.Request) {
//	    fmt.Fprintln(w, "net-gopher: A Gopher protocol library for Go")
//	})
//
//	log.Fatal(gopher.ListenAndServe(":7070", nil))
package gopher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
)

// ItemType represents a Gopher item type.
type ItemType byte

// Gopher item type constants.
const (
	TypeText          ItemType = '0'
	TypeDirectory              = '1'
	TypeCCSOPhonebook          = '2'
	TypeError                  = '3'
	TypeBinHex                 = '4'
	TypeDOS                    = '5'
	TypeUUEncoded              = '6'
	TypeSearch                 = '7'
	TypeTelnet                 = '8'
	TypeBinary                 = '9'
	TypeGIF                    = 'g'
	TypeImage                  = 'I'
	TypeHTML                   = 'h'
	TypeInfo                   = 'i'
)

// Item represents a single line in a Gopher menu/directory.
type Item struct {
	// Type is the Gopher item type (0, 1, i, etc.)
	Type ItemType

	// Display is the user-visible text.
	Display string

	// Selector is the path to the resource
	Selector string

	// Host is the server hostname
	Host string

	// Port is the server port
	Port string
}

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

var (
	DefaultHost = "error.host"
	DefaultPort = "70"

	ErrInvalidScheme = errors.New("invalid scheme")
	ErrInvalidURL    = errors.New("invalid URL")
)

// Request represents a Gopher request
// received by a server or to be sent by a client.
type Request struct {
	// Type is the Gopher item type (0, 1, 7, etc.)
	Type ItemType

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

func NewRequest(rawURL string) (*Request, error) {
	return NewRequestWithContext(context.Background(), rawURL)
}

func NewRequestWithContext(ctx context.Context, rawURL string) (*Request, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}

	if u.Scheme != "gopher" {
		return nil, ErrInvalidScheme
	}

	path := u.Path
	var itemType ItemType = TypeDirectory
	var selector string

	if len(path) >= 2 && path[0] == '/' {
		itemType = ItemType(path[1])
		selector = path[2:]
	} else if path == "/" || path == "" {
		selector = ""
	} else {
		selector = path
	}

	return &Request{
		Type:     itemType,
		Selector: selector,
		Query:    u.RawQuery,
		Host:     u.Host,
		URL:      u,
		ctx:      ctx,
	}, nil
}

func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
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
