package gopher

import (
	"bytes"
	"errors"
	"net"
	"testing"
	"time"
)

func TestItem_String(t *testing.T) {
	tests := []struct {
		name       string
		item       Item
		wantString string
	}{
		{
			name: "happy path",
			item: Item{
				Type:     TypeText,
				Display:  "Hello",
				Selector: "/hello",
				Host:     "localhost",
				Port:     "70",
			},
			wantString: "0Hello\t/hello\tlocalhost\t70\r\n",
		},
		{
			name: "default host",
			item: Item{
				Type:     TypeText,
				Display:  "Default Host",
				Selector: "/default-host",
				Port:     "70",
			},
			wantString: "0Default Host\t/default-host\terror.host\t70\r\n",
		},
		{
			name: "default port",
			item: Item{
				Type:     TypeText,
				Display:  "Default Port",
				Selector: "/default-port",
				Host:     "localhost",
			},
			wantString: "0Default Port\t/default-port\tlocalhost\t70\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.item.String() != tt.wantString {
				t.Errorf("got %q, wanted %q", tt.item.String(), tt.wantString)
			}
		})
	}
}

type mockConn struct {
	*bytes.Buffer
}

func (m mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m mockConn) Close() error                       { return nil }
func (m mockConn) LocalAddr() net.Addr                { return nil }
func (m mockConn) RemoteAddr() net.Addr               { return nil }
func (m mockConn) SetDeadline(t time.Time) error      { return nil }
func (m mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m mockConn) SetWriteDeadline(t time.Time) error { return nil }

type TestConfig struct {
	defaultHost string
	defaultPort string
}

// Helper to create a test ResponseWriter
func newTestResponseWriterWithConfig(config *TestConfig) (*responseWriter, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	w := &responseWriter{
		conn: mockConn{buf},
		host: config.defaultHost,
		port: config.defaultPort,
	}
	return w, buf
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
