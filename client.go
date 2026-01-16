package gopher

import (
	"context"
	"fmt"
	"net"
	"time"
)

// RoundTripper is an interface representing the ability to execute a
// single Gopher transaction, obtaining the [Response] for a given [Request].
//
// A RoundTripper must be safe for concurrent use by multiple
// goroutines.
type RoundTripper interface {
	// RoundTrip executes a single Gopher transaction, returning
	// a Response for the provided Request.
	RoundTrip(*Request) (*Response, error)
}

// Transport is an implementation of [RoundTripper] that supports Gopher.
type Transport struct {
	// DialContext specifies the dial function for creating TCP connections.
	// If DialContext is nil, then the transport dials using package net.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)
}

// DefaultTransport is the default [RoundTripper] used by [DefaultClient].
var DefaultTransport = &Transport{}

func (t *Transport) dial(ctx context.Context, addr string) (net.Conn, error) {
	if t.DialContext != nil {
		return t.DialContext(ctx, "tcp", addr)
	}
	var d net.Dialer
	return d.DialContext(ctx, "tcp", addr)
}

// RoundTrip implements the [RoundTripper] interface.
func (t *Transport) RoundTrip(req *Request) (*Response, error) {
	conn, err := t.dial(req.ctx, req.Host)
	if err != nil {
		return nil, err
	}

	selector := req.Selector
	if req.Query != "" {
		selector += "\t" + req.Query
	}
	if _, err := fmt.Fprintf(conn, "%s\r\n", selector); err != nil {
		conn.Close()
		return nil, err
	}

	return &Response{Request: req, Body: conn}, nil
}

// A Client is a Gopher client. Its zero value ([DefaultClient]) is a
// usable client.
type Client struct {
	Transport RoundTripper

	// Timeout specifies a time limit for requests made by this
	// Client. The timeout includes connection time and reading
	// the response body.
	//
	// A Timeout of zero means no timeout.
	Timeout time.Duration
}

// DefaultClient is the default [Client] and is used by [Get].
var DefaultClient = &Client{}

func (c *Client) transport() RoundTripper {
	if c.Transport != nil {
		return c.Transport
	}
	return DefaultTransport
}

// Do sends a Gopher request and returns a Gopher response.
//
// An error is returned if caused by client policy or failure to
// speak Gopher (such as a network connectivity problem).
//
// If the returned error is nil, the [Response] will contain a non-nil
// Body which the user is expected to close.
func (c *Client) Do(req *Request) (*Response, error) {
	ctx := req.Context()
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	return c.transport().RoundTrip(req)
}

// Get issues a request to the specified URL.
//
// When err is nil, resp always contains a non-nil resp.Body.
// Caller should close resp.Body when done reading from it.
func (c *Client) Get(rawURL string) (*Response, error) {
	req, err := NewRequest(rawURL)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Get issues a request to the specified URL using [DefaultClient].
//
// When err is nil, resp always contains a non-nil resp.Body.
// Caller should close resp.Body when done reading from it.
func Get(rawURL string) (*Response, error) {
	return DefaultClient.Get(rawURL)
}
