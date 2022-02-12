package connections

import (
	"context"
	"net"
	"time"

	sync "github.com/plgd-dev/kit/v2/sync"
)

type Connections struct {
	data *sync.Map
}

func New() *Connections {
	return &Connections{
		data: sync.NewMap(),
	}
}

type Connection interface {
	Context() context.Context
	CheckExpirations(now time.Time)
	Close() error
	RemoteAddr() net.Addr
}

func (c *Connections) Store(conn Connection) {
	c.data.Store(conn.RemoteAddr().String(), conn)
}

func (c *Connections) copyConnections() []Connection {
	m := make([]Connection, 0, c.data.Length())
	c.data.Range(func(key, value interface{}) bool {
		m = append(m, value.(Connection))
		return true
	})
	return m
}

func (c *Connections) CheckExpirations(now time.Time) {
	for _, cc := range c.copyConnections() {
		select {
		case <-cc.Context().Done():
			continue
		default:
			cc.CheckExpirations(now)
		}
	}
}

func (c *Connections) Close() {
	for _, cc := range c.copyConnections() {
		_ = cc.Close()
	}
}

func (c *Connections) Delete(conn Connection) {
	c.data.Delete(conn.RemoteAddr().String())
}
