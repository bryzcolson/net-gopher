package gopher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

// Handler responds to a Gopher request.
type Handler interface {
	ServeGopher(ResponseWriter, *Request)
}

// HandlerFunc type is an adapter to allow the use of
// ordinary functions as Gopher handlers.
type HandlerFunc func(ResponseWriter, *Request)

// ServeGopher calls f(w, r).
func (f HandlerFunc) ServeGopher(w ResponseWriter, r *Request) {
	f(w, r)
}

// ResponseWriter writes responses to a Gopher request.
type ResponseWriter interface {
	// Write writes raw bytes to the connection.
	Write([]byte) (int, error)

	// WriteItem writes a single menu item.
	WriteItem(item *Item) error

	// WriteInfo is a convenience method for writing an info item.
	WriteInfo(text string) error

	// WriteError is a convenience method for writing an error item.
	WriteError(text string) error

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

func (w *responseWriter) WriteError(text string) error {
	item := &Item{
		Type:     TypeError,
		Display:  text,
		Selector: "",
		Host:     "error.host",
		Port:     "70",
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

type ServeMux struct {
	mu sync.RWMutex
	m  map[string]Handler
}

func NewServeMux() *ServeMux {
	return &ServeMux{m: make(map[string]Handler)}
}

func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	mux.Handle(pattern, HandlerFunc(handler))
}

func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.m[pattern] = handler
}

func (mux *ServeMux) match(selector string) Handler {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	if h, ok := mux.m[selector]; ok {
		return h
	}
	var longest string
	var handler Handler
	for pattern, h := range mux.m {
		if strings.HasPrefix(selector, pattern) && len(pattern) > len(longest) {
			longest = pattern
			handler = h
		}
	}
	return handler
}

// ServeGopher dispatches the request to the handler whose
// pattern most closely matches the request selector.
func (mux *ServeMux) ServeGopher(w ResponseWriter, r *Request) {
	h := mux.match(r.Selector)
	if h == nil {
		w.WriteError("selector not found")
		return
	}
	h.ServeGopher(w, r)
}

// DefaultServeMux is the default [ServeMux] used by [Serve].
var DefaultServeMux = NewServeMux()

// Handle registers the handler for the given pattern in [DefaultServeMux].
func Handle(pattern string, handler Handler) {
	DefaultServeMux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern in [DefaultServeMux].
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}

// Server defines parameters for running a Gopher server.
type Server struct {
	Addr    string
	Handler Handler
}

func (srv *Server) serveConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return
	}
	line = strings.TrimRight(line, "\r\n")

	parts := strings.SplitN(line, "\t", 2)
	selector := parts[0]
	if selector == "" {
		selector = "/"
	}

	host, port, _ := net.SplitHostPort(conn.LocalAddr().String())
	req := &Request{
		Selector: selector,
		Host:     conn.LocalAddr().String(),
		ctx:      context.Background(),
	}
	if len(parts) > 1 {
		req.Query = parts[1]
	}

	w := &responseWriter{conn: conn, host: host, port: port}

	handler := srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
	handler.ServeGopher(w, req)

	fmt.Fprintf(conn, ".\r\n")
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each.
func (srv *Server) Serve(l net.Listener) error {
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go srv.serveConn(conn)
	}
}

// ListenAndServe listens on the TCP network address srv.Addr and then
// calls [Serve] to handle requests on incoming connections.
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":70"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

// ListenAndServe listens on the TCP network address addr and then calls
// [Serve] with handler to handle requests on incoming connections.
// The handler is typically nil, in which case [DefaultServeMux] is used.
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// Serve accepts incoming Gopher connections on the Listener l, creating a
// new service goroutine for each.
// The handler is typically nil, in which case [DefaultServeMux] is used.
func Serve(l net.Listener, handler Handler) error {
	server := &Server{Handler: handler}
	return server.Serve(l)
}
