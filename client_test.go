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
