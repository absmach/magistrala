package tcp

import (
	"context"
	"io"
	"net"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

type ClientTCP struct {
	cc *ClientConn
}

func NewClientTCP(cc *ClientConn) *ClientTCP {
	return &ClientTCP{
		cc: cc,
	}
}

func (c *ClientTCP) Ping(ctx context.Context) error {
	return c.cc.Ping(ctx)
}

func (c *ClientTCP) Delete(ctx context.Context, path string, opts ...message.Option) (*message.Message, error) {
	resp, err := c.cc.Delete(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	defer c.cc.session.messagePool.ReleaseMessage(resp)
	return pool.ConvertTo(resp)
}

func (c *ClientTCP) Put(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*message.Message, error) {
	resp, err := c.cc.Put(ctx, path, contentFormat, payload, opts...)
	if err != nil {
		return nil, err
	}
	defer c.cc.session.messagePool.ReleaseMessage(resp)
	return pool.ConvertTo(resp)
}

func (c *ClientTCP) Post(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*message.Message, error) {
	resp, err := c.cc.Post(ctx, path, contentFormat, payload, opts...)
	if err != nil {
		return nil, err
	}
	defer c.cc.session.messagePool.ReleaseMessage(resp)
	return pool.ConvertTo(resp)
}

func (c *ClientTCP) Get(ctx context.Context, path string, opts ...message.Option) (*message.Message, error) {
	resp, err := c.cc.Get(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	defer c.cc.session.messagePool.ReleaseMessage(resp)
	return pool.ConvertTo(resp)
}

func (c *ClientTCP) Close() error {
	return c.cc.Close()
}

func (c *ClientTCP) RemoteAddr() net.Addr {
	return c.cc.RemoteAddr()
}

func (c *ClientTCP) Context() context.Context {
	return c.cc.Context()
}

func (c *ClientTCP) SetContextValue(key interface{}, val interface{}) {
	c.cc.Session().SetContextValue(key, val)
}

func (c *ClientTCP) WriteMessage(req *message.Message) error {
	r, err := c.cc.session.messagePool.ConvertFrom(req)
	if err != nil {
		return err
	}
	defer c.cc.session.messagePool.ReleaseMessage(r)
	return c.cc.WriteMessage(r)
}

func (c *ClientTCP) Do(req *message.Message) (*message.Message, error) {
	r, err := c.cc.session.messagePool.ConvertFrom(req)
	if err != nil {
		return nil, err
	}
	defer c.cc.session.messagePool.ReleaseMessage(r)
	resp, err := c.cc.Do(r)
	if err != nil {
		return nil, err
	}
	defer c.cc.session.messagePool.ReleaseMessage(resp)
	return pool.ConvertTo(resp)
}

func createClientConnObserveHandler(observeFunc func(notification *message.Message)) func(n *pool.Message) {
	return func(n *pool.Message) {
		muxn, err := pool.ConvertTo(n)
		if err != nil {
			return
		}
		observeFunc(muxn)
	}
}

func (c *ClientTCP) Observe(ctx context.Context, path string, observeFunc func(notification *message.Message), opts ...message.Option) (mux.Observation, error) {
	return c.cc.Observe(ctx, path, createClientConnObserveHandler(observeFunc), opts...)
}

// Sequence acquires sequence number.
func (c *ClientTCP) Sequence() uint64 {
	return c.cc.Sequence()
}

// ClientConn get's underlaying client connection.
func (c *ClientTCP) ClientConn() interface{} {
	return c.cc
}

// Done signalizes that connection is not more processed.
func (c *ClientTCP) Done() <-chan struct{} {
	return c.cc.Done()
}
