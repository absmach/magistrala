package mux

import (
	"context"
	"io"
	"net"

	"github.com/plgd-dev/go-coap/v2/message"
)

type Observation = interface {
	Cancel(ctx context.Context) error
}

type Client interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, path string, opts ...message.Option) (*message.Message, error)
	Delete(ctx context.Context, path string, opts ...message.Option) (*message.Message, error)
	Post(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*message.Message, error)
	Put(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*message.Message, error)
	Observe(ctx context.Context, path string, observeFunc func(notification *message.Message), opts ...message.Option) (Observation, error)
	ClientConn() interface{}

	RemoteAddr() net.Addr
	Context() context.Context
	WriteMessage(req *message.Message) error
	Do(req *message.Message) (*message.Message, error)
	Close() error
	Sequence() uint64
}
