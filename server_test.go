package gopher

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func (m *mockConn) Read(b []byte) (int, error)         { return m.readBuf.Read(b) }
func (m *mockConn) Write(b []byte) (int, error)        { return m.writeBuf.Write(b) }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 70} }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func newMockConn(input string) (*mockConn, *bytes.Buffer) {
	writeBuf := &bytes.Buffer{}
	return &mockConn{readBuf: bytes.NewBufferString(input), writeBuf: writeBuf}, writeBuf
}

type TestConfig struct {
	defaultHost string
	defaultPort string
}

func newTestResponseWriterWithConfig(config *TestConfig) (*responseWriter, *bytes.Buffer) {
	conn, writeBuf := newMockConn("")
	w := &responseWriter{
		conn: conn,
		host: config.defaultHost,
		port: config.defaultPort,
	}
	return w, writeBuf
}

func newTestResponseWriter() (*responseWriter, *bytes.Buffer) {
	return newTestResponseWriterWithConfig(&TestConfig{"localhost", "70"})
}

func TestResponseWriter_Write(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		wantString string
		wantErr    error
	}{
		{
			name:       "happy path",
			input:      []byte("hello"),
			wantString: "hello",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, buf := newTestResponseWriter()
			n, err := w.Write(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Write() error = %v, wanted %v", err, tt.wantErr)
			}
			if n != len(tt.input) {
				t.Errorf("Write() wrote %d bytes, wanted %d", n, len(tt.input))
			}
			if got := buf.String(); got != tt.wantString {
				t.Errorf("Write() output = %q, wanted %q", got, tt.wantString)
			}
		})
	}
}

func TestResponseWriter_WriteItem(t *testing.T) {
	tests := []struct {
		name       string
		item       *Item
		wantString string
		wantErr    error
	}{
		{
			name: "happy path",
			item: &Item{
				Type:     TypeText,
				Display:  "Hello",
				Selector: "/hello",
				Host:     "localhost",
				Port:     "70",
			},
			wantString: "0Hello\t/hello\tlocalhost\t70\r\n",
			wantErr:    nil,
		},
		{
			name: "default host",
			item: &Item{
				Type:     TypeText,
				Display:  "Default Host",
				Selector: "/default-host",
				Port:     "70",
			},
			wantString: "0Default Host\t/default-host\texample.com\t70\r\n",
			wantErr:    nil,
		},
		{
			name: "default port",
			item: &Item{
				Type:     TypeText,
				Display:  "Default Port",
				Selector: "/default-port",
				Host:     "localhost",
			},
			wantString: "0Default Port\t/default-port\tlocalhost\t7070\r\n",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, buf := newTestResponseWriterWithConfig(&TestConfig{"example.com", "7070"})
			err := w.WriteItem(tt.item)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("WriteItem() error = %v, wanted %v", err, tt.wantErr)
			}
			if got := buf.String(); got != tt.wantString {
				t.Errorf("WriteItem() wrote = %q, wanted %q", got, tt.wantString)
			}
		})
	}
}

func TestResponseWriter_WriteInfo(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantString string
		wantErr    error
	}{
		{
			name:       "happy path",
			text:       "Hello",
			wantString: "iHello\t\tlocalhost\t70\r\n",
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, buf := newTestResponseWriter()
			err := w.WriteInfo(tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("WriteInfo() error = %v, wanted %v", err, tt.wantErr)
			}
			if got := buf.String(); got != tt.wantString {
				t.Errorf("WriteInfo() wrote = %q, wanted %q", got, tt.wantString)
			}
		})
	}
}

func TestResponseWriter_WriteDirectory(t *testing.T) {
	tests := []struct {
		name       string
		items      []*Item
		wantString string
		wantErr    error
	}{
		{
			name: "happy path",
			items: []*Item{
				{Type: TypeInfo, Display: "Hello"},
				{Type: TypeText, Display: "About", Selector: "/about"},
				{Type: TypeDirectory, Display: "Subdirectory", Selector: "/nested"},
			},
			wantString: "iHello\t\tlocalhost\t70\r\n" +
				"0About\t/about\tlocalhost\t70\r\n" +
				"1Subdirectory\t/nested\tlocalhost\t70\r\n" +
				".\r\n",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, buf := newTestResponseWriter()
			err := w.WriteDirectory(tt.items)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("WriteDirectory() error = %v, wanted %v", err, tt.wantErr)
			}
			if got := buf.String(); got != tt.wantString {
				t.Errorf("WriteDirectory() output mismatch")
			}
		})
	}
}

func TestHandlerFunc(t *testing.T) {
	called := false
	f := HandlerFunc(func(w ResponseWriter, r *Request) {
		called = true
	})

	f.ServeGopher(nil, nil)
	if !called {
		t.Error("HandlerFunc did not call underlying function")
	}
}

func TestServer_serveConn(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantSelector string
		wantQuery    string
	}{
		{"simple selector", "/hello\r\n", "/hello", ""},
		{"empty selector", "\r\n", "/", ""},
		{"selector with query", "/search\tgopher\r\n", "/search", "gopher"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq *Request
			srv := &Server{
				Handler: HandlerFunc(func(w ResponseWriter, r *Request) {
					gotReq = r
				}),
			}

			conn, _ := newMockConn(tt.input)
			srv.serveConn(conn)

			if gotReq == nil {
				t.Fatal("handler not called")
			}
			if gotReq.Selector != tt.wantSelector {
				t.Errorf("selector = %q, want %q", gotReq.Selector, tt.wantSelector)
			}
			if gotReq.Query != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotReq.Query, tt.wantQuery)
			}
		})
	}
}

func TestServer_Serve(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var gotSelector string
	srv := &Server{
		Handler: HandlerFunc(func(w ResponseWriter, r *Request) {
			gotSelector = r.Selector
			w.WriteInfo("hello")
		}),
	}

	go srv.Serve(ln)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "/test\r\n")

	buf := make([]byte, 256)
	n, _ := conn.Read(buf)

	if gotSelector != "/test" {
		t.Errorf("selector = %q, want /test", gotSelector)
	}
	if !bytes.Contains(buf[:n], []byte("hello")) {
		t.Errorf("response missing 'hello': %q", buf[:n])
	}
}

func TestServeMux_Handle(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		selector string
		wantCall bool
	}{
		{"exact match", "/test", "/test", true},
		{"no match", "/test", "/other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := NewServeMux()
			called := false
			mux.Handle(tt.pattern, HandlerFunc(func(w ResponseWriter, r *Request) {
				called = true
			}))

			w, _ := newTestResponseWriter()
			mux.ServeGopher(w, &Request{Selector: tt.selector})

			if called != tt.wantCall {
				t.Errorf("Handle() called = %v, wanted %v", called, tt.wantCall)
			}
		})
	}
}

func TestServeMux_HandleFunc(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		selector string
		wantCall bool
	}{
		{"exact match", "/test", "/test", true},
		{"no match", "/test", "/other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := NewServeMux()
			called := false
			mux.HandleFunc(tt.pattern, func(w ResponseWriter, r *Request) {
				called = true
			})

			w, _ := newTestResponseWriter()
			mux.ServeGopher(w, &Request{Selector: tt.selector})

			if called != tt.wantCall {
				t.Errorf("HandleFunc() called = %v, wanted %v", called, tt.wantCall)
			}
		})
	}
}

func TestServeMux_Match(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		selector string
		want     string
	}{
		{"exact match", []string{"/", "/about"}, "/about", "/about"},
		{"prefix match", []string{"/", "/files"}, "/files/doc.txt", "/files"},
		{"longest prefix", []string{"/", "/a", "/a/b"}, "/a/b/c", "/a/b"},
		{"root fallback", []string{"/"}, "/unknown", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := NewServeMux()
			var matched string
			for _, p := range tt.patterns {
				pattern := p
				mux.HandleFunc(pattern, func(w ResponseWriter, r *Request) {
					matched = pattern
				})
			}

			w, _ := newTestResponseWriter()
			mux.ServeGopher(w, &Request{Selector: tt.selector})

			if matched != tt.want {
				t.Errorf("match() matched = %q, wanted %q", matched, tt.want)
			}
		})
	}
}

func TestServeMux_NotFound(t *testing.T) {
	tests := []struct {
		name       string
		selector   string
		wantString string
	}{
		{"missing selector", "/missing", "selector not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := NewServeMux()
			w, buf := newTestResponseWriter()
			mux.ServeGopher(w, &Request{Selector: tt.selector})

			if got := buf.String(); !strings.Contains(got, tt.wantString) {
				t.Errorf("ServeGopher() output = %q, wanted %q", got, tt.wantString)
			}
		})
	}
}

func TestDefaultServeMux_HandleFunc(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		selector string
		wantCall bool
	}{
		{"registers handler", "/test", "/test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := DefaultServeMux
			DefaultServeMux = NewServeMux()
			defer func() { DefaultServeMux = original }()

			called := false
			HandleFunc(tt.pattern, func(w ResponseWriter, r *Request) {
				called = true
			})

			w, _ := newTestResponseWriter()
			DefaultServeMux.ServeGopher(w, &Request{Selector: tt.selector})

			if called != tt.wantCall {
				t.Errorf("HandleFunc() called = %v, wanted %v", called, tt.wantCall)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name         string
		rawURL       string
		wantType     ItemType
		wantSelector string
		wantQuery    string
		wantHost     string
		wantErr      error
	}{
		{
			name:         "directory with selector",
			rawURL:       "gopher://example.com/1/foo/bar",
			wantType:     TypeDirectory,
			wantSelector: "/foo/bar",
			wantHost:     "example.com",
		},
		{
			name:         "text file",
			rawURL:       "gopher://example.com/0/docs/readme.txt",
			wantType:     TypeText,
			wantSelector: "/docs/readme.txt",
			wantHost:     "example.com",
		},
		{
			name:         "type only no selector",
			rawURL:       "gopher://example.com/1",
			wantType:     TypeDirectory,
			wantSelector: "",
			wantHost:     "example.com",
		},
		{
			name:         "empty path defaults to directory",
			rawURL:       "gopher://example.com",
			wantType:     TypeDirectory,
			wantSelector: "",
			wantHost:     "example.com",
		},
		{
			name:         "root path defaults to directory",
			rawURL:       "gopher://example.com/",
			wantType:     TypeDirectory,
			wantSelector: "",
			wantHost:     "example.com",
		},
		{
			name:         "search with query",
			rawURL:       "gopher://example.com/7/search?hello+world",
			wantType:     TypeSearch,
			wantSelector: "/search",
			wantQuery:    "hello+world",
			wantHost:     "example.com",
		},
		{
			name:         "custom port",
			rawURL:       "gopher://example.com:7070/1/files",
			wantType:     TypeDirectory,
			wantSelector: "/files",
			wantHost:     "example.com:7070",
		},
		{
			name:         "binary file",
			rawURL:       "gopher://example.com/9/archive.tar.gz",
			wantType:     TypeBinary,
			wantSelector: "/archive.tar.gz",
			wantHost:     "example.com",
		},
		{
			name:         "image file",
			rawURL:       "gopher://example.com/I/photo.jpg",
			wantType:     TypeImage,
			wantSelector: "/photo.jpg",
			wantHost:     "example.com",
		},
		{
			name:    "invalid scheme",
			rawURL:  "http://example.com/foo",
			wantErr: ErrInvalidScheme,
		},
		{
			name:    "invalid URL",
			rawURL:  "://bad-url",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest(tt.rawURL)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewRequest() error = %v, wanted %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRequest() unexpected error: %v", err)
			}
			if req.Type != tt.wantType {
				t.Errorf("NewRequest() type = %q, wanted %q", req.Type, tt.wantType)
			}
			if req.Selector != tt.wantSelector {
				t.Errorf("NewRequest() selector = %q, wanted %q", req.Selector, tt.wantSelector)
			}
			if req.Query != tt.wantQuery {
				t.Errorf("NewRequest() query = %q, wanted %q", req.Query, tt.wantQuery)
			}
			if req.Host != tt.wantHost {
				t.Errorf("NewRequest() host = %q, wanted %q", req.Host, tt.wantHost)
			}
			if req.URL == nil {
				t.Error("NewRequest() URL is nil")
			}
			if req.ctx == nil {
				t.Error("NewRequest() ctx is nil")
			}
		})
	}
}
