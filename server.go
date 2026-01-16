package gopher

import "net"

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
