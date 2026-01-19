package gopher

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

type clientMockConn struct {
	readData  string
	writeData strings.Builder
}

func (m *clientMockConn) Read(b []byte) (int, error) {
	if m.readData == "" {
		return 0, io.EOF
	}
	n := copy(b, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *clientMockConn) Write(b []byte) (int, error)        { return m.writeData.Write(b) }
func (m *clientMockConn) Close() error                       { return nil }
func (m *clientMockConn) LocalAddr() net.Addr                { return nil }
func (m *clientMockConn) RemoteAddr() net.Addr               { return nil }
func (m *clientMockConn) SetDeadline(t time.Time) error      { return nil }
func (m *clientMockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *clientMockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		selector     string
		query        string
		wantWritten  string
		responseBody string
	}{
		{
			name:         "simple selector",
			selector:     "/hello",
			wantWritten:  "/hello\r\n",
			responseBody: "Hello, World!",
		},
		{
			name:         "empty selector",
			selector:     "",
			wantWritten:  "\r\n",
			responseBody: "root menu",
		},
		{
			name:         "selector with query",
			selector:     "/search",
			query:        "gopher",
			wantWritten:  "/search\tgopher\r\n",
			responseBody: "search results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &clientMockConn{readData: tt.responseBody}

			transport := &Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return conn, nil
				},
			}

			req := &Request{
				Selector: tt.selector,
				Query:    tt.query,
				Host:     "localhost:70",
				ctx:      context.Background(),
			}

			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}

			if got := conn.writeData.String(); got != tt.wantWritten {
				t.Errorf("written = %q, wanted %q", got, tt.wantWritten)
			}

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.responseBody {
				t.Errorf("body = %q, wanted %q", body, tt.responseBody)
			}
		})
	}
}

func TestTransport_RoundTrip_DialError(t *testing.T) {
	dialErr := errors.New("connection refused")
	transport := &Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, dialErr
		},
	}

	req := &Request{Host: "localhost:70", ctx: context.Background()}
	_, err := transport.RoundTrip(req)
	if !errors.Is(err, dialErr) {
		t.Errorf("got error %v, wanted %v", err, dialErr)
	}
}

func TestClient_Get(t *testing.T) {
	conn := &clientMockConn{readData: "test response"}

	client := &Client{
		Transport: &Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return conn, nil
			},
		},
	}

	resp, err := client.Get("gopher://localhost/0test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test response" {
		t.Errorf("body = %q, wanted %q", body, "test response")
	}
}

func TestClient_Get_InvalidURL(t *testing.T) {
	client := &Client{}

	_, err := client.Get("http://example.com")
	if !errors.Is(err, ErrInvalidScheme) {
		t.Errorf("got error %v, wanted ErrInvalidScheme", err)
	}
}

func TestClient_transport(t *testing.T) {
	t.Run("custom transport", func(t *testing.T) {
		custom := &Transport{}
		client := &Client{Transport: custom}
		if client.transport() != custom {
			t.Error("expected custom transport")
		}
	})

	t.Run("default transport", func(t *testing.T) {
		client := &Client{}
		if client.transport() != DefaultTransport {
			t.Error("expected DefaultTransport")
		}
	})
}

func TestClient_Timeout(t *testing.T) {
	client := &Client{
		Timeout: 50 * time.Millisecond,
		Transport: &Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return &clientMockConn{}, nil
				}
			},
		},
	}

	_, err := client.Get("gopher://localhost/")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got error %v, wanted context.DeadlineExceeded", err)
	}
}

func TestTextReader_PeriodUnstuffing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		itemType ItemType
		want     string
	}{
		{
			name:     "simple text with terminator",
			input:    "Hello, World!\r\n.\r\n",
			itemType: TypeText,
			want:     "Hello, World!\r\n",
		},
		{
			name:     "period unstuffing at line start",
			input:    "..starts with period\r\n.\r\n",
			itemType: TypeText,
			want:     ".starts with period\r\n",
		},
		{
			name:     "period unstuffing multiple lines",
			input:    "..first\r\n..second\r\n..third\r\n.\r\n",
			itemType: TypeText,
			want:     ".first\r\n.second\r\n.third\r\n",
		},
		{
			name:     "period alone on line unstuffed",
			input:    "before\r\n..\r\nafter\r\n.\r\n",
			itemType: TypeText,
			want:     "before\r\n.\r\nafter\r\n",
		},
		{
			name:     "no unstuffing for single period mid-line",
			input:    "hello.world\r\n.\r\n",
			itemType: TypeText,
			want:     "hello.world\r\n",
		},
		{
			name:     "terminator with just LF",
			input:    "test\n.\n",
			itemType: TypeText,
			want:     "test\n",
		},
		{
			name:     "directory listing",
			input:    "1Menu Item\t/path\tlocalhost\t70\r\niInfo line\t\tlocalhost\t70\r\n.\r\n",
			itemType: TypeDirectory,
			want:     "1Menu Item\t/path\tlocalhost\t70\r\niInfo line\t\tlocalhost\t70\r\n",
		},
		{
			name:     "binary type no unstuffing",
			input:    "..binary data\r\n.\r\nmore data",
			itemType: TypeBinary,
			want:     "..binary data\r\n.\r\nmore data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &clientMockConn{readData: tt.input}
			transport := &Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return conn, nil
				},
			}

			req := &Request{
				Type:     tt.itemType,
				Selector: "/test",
				Host:     "localhost:70",
				ctx:      context.Background(),
			}

			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			if string(body) != tt.want {
				t.Errorf("body = %q, wanted %q", string(body), tt.want)
			}
		})
	}
}

func TestTextReader_EmptyResponse(t *testing.T) {
	conn := &clientMockConn{readData: ".\r\n"}
	transport := &Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return conn, nil
		},
	}

	req := &Request{
		Type:     TypeText,
		Selector: "/empty",
		Host:     "localhost:70",
		ctx:      context.Background(),
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(body) != 0 {
		t.Errorf("body = %q, wanted empty", string(body))
	}
}

func TestTextReader_NoTerminator(t *testing.T) {
	// Server doesn't send terminator - should read until EOF
	conn := &clientMockConn{readData: "Hello\r\nWorld\r\n"}
	transport := &Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return conn, nil
		},
	}

	req := &Request{
		Type:     TypeText,
		Selector: "/test",
		Host:     "localhost:70",
		ctx:      context.Background(),
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	want := "Hello\r\nWorld\r\n"
	if string(body) != want {
		t.Errorf("body = %q, wanted %q", string(body), want)
	}
}

func TestIsTextType(t *testing.T) {
	tests := []struct {
		itemType ItemType
		want     bool
	}{
		// RFC 1436: Only types 0, 1, and 7 use period-terminated format
		{TypeText, true},        // 0 - uses TextFile format with period terminator
		{TypeDirectory, true},   // 1 - uses Menu format with period terminator
		{TypeSearch, true},      // 7 - sends Menu Entity with period terminator

		// All other types do not use period stuffing
		{TypeCCSOPhonebook, false}, // 2 - uses external CSO protocol
		{TypeError, false},         // 3 - error indicator, not a response format
		{TypeBinHex, false},        // 4
		{TypeDOS, false},           // 5 - binary, explicitly no terminator
		{TypeUUEncoded, false},     // 6
		{TypeTelnet, false},        // 8
		{TypeBinary, false},        // 9 - binary, explicitly no terminator
		{TypeGIF, false},           // g
		{TypeImage, false},         // I
		{TypeHTML, false},          // h
		{TypeInfo, false},          // i - info line in menus, not a response type
	}

	for _, tt := range tests {
		t.Run(string(tt.itemType), func(t *testing.T) {
			if got := isTextType(tt.itemType); got != tt.want {
				t.Errorf("isTextType(%c) = %v, wanted %v", tt.itemType, got, tt.want)
			}
		})
	}
}
