package connections

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type Connections struct {
	data *sync.Map
}

func New() *Connections {
	return &Connections{
		data: &sync.Map{},
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

func (c *Connections) length() int {
	var l int
	c.data.Range(func(k, v interface{}) bool {
		l++
		return true
	})
	return l
}

func (c *Connections) copyConnections() []Connection {
	m := make([]Connection, 0, c.length())
	c.data.Range(func(key, value interface{}) bool {
		con, ok := value.(Connection)
		if !ok {
			panic(fmt.Errorf("invalid type %T in connections map", con))
		}
		m = append(m, con)
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
