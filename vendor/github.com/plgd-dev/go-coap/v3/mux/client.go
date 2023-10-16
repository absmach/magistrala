package mux

import (
	"context"
	"io"
	"net"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type Observation = interface {
	Cancel(ctx context.Context, opts ...message.Option) error
	Canceled() bool
}

type Conn interface {
	// create message from pool
	AcquireMessage(ctx context.Context) *pool.Message
	// return back the message to the pool for next use
	ReleaseMessage(m *pool.Message)

	Ping(ctx context.Context) error
	Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	Delete(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	Post(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error)
	Put(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error)
	Observe(ctx context.Context, path string, observeFunc func(notification *pool.Message), opts ...message.Option) (Observation, error)

	RemoteAddr() net.Addr
	// NetConn returns the underlying connection that is wrapped by client. The Conn returned is shared by all invocations of NetConn, so do not modify it.
	NetConn() net.Conn
	Context() context.Context
	SetContextValue(key interface{}, val interface{})
	WriteMessage(req *pool.Message) error
	// used for GET,PUT,POST,DELETE
	Do(req *pool.Message) (*pool.Message, error)
	// used for observation (GET with observe 0)
	DoObserve(req *pool.Message, observeFunc func(req *pool.Message)) (Observation, error)
	Close() error
	Sequence() uint64
	// Done signalizes that connection is not more processed.
	Done() <-chan struct{}
	AddOnClose(func())

	NewGetRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	NewObserveRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	NewPutRequest(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error)
	NewPostRequest(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error)
	NewDeleteRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
}
