package websocket

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsWrapper is a websocket wrapper so it satisfies the net.Conn interface.
type wsWrapper struct {
	*websocket.Conn
	r   io.Reader
	rio sync.Mutex
	wio sync.Mutex
}

func newConn(ws *websocket.Conn) net.Conn {
	wrapper := &wsWrapper{
		Conn: ws,
	}
	return wrapper

}

// SetDeadline sets both the read and write deadlines
func (c *wsWrapper) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	err := c.SetWriteDeadline(t)
	return err
}

// Write writes data to the websocket
func (c *wsWrapper) Write(p []byte) (int, error) {
	c.wio.Lock()
	defer c.wio.Unlock()

	err := c.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Read reads the current websocket frame
func (c *wsWrapper) Read(p []byte) (int, error) {
	c.rio.Lock()
	defer c.rio.Unlock()
	for {
		if c.r == nil {
			// Advance to next message.
			var err error
			_, c.r, err = c.NextReader()
			if err != nil {
				return 0, err
			}
		}
		n, err := c.r.Read(p)
		if err == io.EOF {
			// At end of message.
			c.r = nil
			if n > 0 {
				return n, nil
			}
			// No data read, continue to next message.
			continue
		}
		return n, err
	}
}

func (c *wsWrapper) Close() error {
	return c.Conn.Close()
}
