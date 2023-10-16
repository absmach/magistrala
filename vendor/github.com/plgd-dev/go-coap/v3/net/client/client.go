package client

import (
	"context"
	"fmt"
	"io"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	limitparallelrequests "github.com/plgd-dev/go-coap/v3/net/client/limitParallelRequests"
	"github.com/plgd-dev/go-coap/v3/net/observation"
)

type (
	GetTokenFunc = func() (message.Token, error)
)

type Conn interface {
	// create message from pool
	AcquireMessage(ctx context.Context) *pool.Message
	// return back the message to the pool for next use
	ReleaseMessage(m *pool.Message)
	WriteMessage(req *pool.Message) error
	AsyncPing(receivedPong func()) (func(), error)
	Context() context.Context
}

type Client[C Conn] struct {
	cc                 Conn
	observationHandler *observation.Handler[C]
	getToken           GetTokenFunc
	*limitparallelrequests.LimitParallelRequests
}

func New[C Conn](cc C, observationHandler *observation.Handler[C], getToken GetTokenFunc, limitParallelRequests *limitparallelrequests.LimitParallelRequests) *Client[C] {
	return &Client[C]{
		cc:                    cc,
		observationHandler:    observationHandler,
		getToken:              getToken,
		LimitParallelRequests: limitParallelRequests,
	}
}

func (c *Client[C]) GetToken() (message.Token, error) {
	return c.getToken()
}

// NewGetRequest creates get request.
//
// Use ctx to set timeout.
func (c *Client[C]) NewGetRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req := c.cc.AcquireMessage(ctx)
	token, err := c.GetToken()
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	err = req.SetupGet(path, token, opts...)
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	return req, nil
}

// Get issues a GET to the specified path.
//
// Use ctx to set timeout.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
func (c *Client[C]) Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req, err := c.NewGetRequest(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create get request: %w", err)
	}
	defer c.cc.ReleaseMessage(req)
	return c.Do(req)
}

type Observation = interface {
	Cancel(ctx context.Context, opts ...message.Option) error
	Canceled() bool
}

// NewObserveRequest creates observe request.
//
// Use ctx to set timeout.
func (c *Client[C]) NewObserveRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req, err := c.NewGetRequest(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	req.SetObserve(0)
	return req, nil
}

// Observe subscribes for every change of resource on path.
func (c *Client[C]) Observe(ctx context.Context, path string, observeFunc func(req *pool.Message), opts ...message.Option) (Observation, error) {
	req, err := c.NewObserveRequest(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	defer c.cc.ReleaseMessage(req)
	return c.DoObserve(req, observeFunc)
}

func (c *Client[C]) GetObservationRequest(token message.Token) (*pool.Message, bool) {
	return c.observationHandler.GetObservationRequest(token)
}

// NewPostRequest creates post request.
//
// Use ctx to set timeout.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
//
// If payload is nil then content format is not used.
func (c *Client[C]) NewPostRequest(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req := c.cc.AcquireMessage(ctx)
	token, err := c.GetToken()
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	err = req.SetupPost(path, token, contentFormat, payload, opts...)
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	return req, nil
}

// Post issues a POST to the specified path.
//
// Use ctx to set timeout.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
//
// If payload is nil then content format is not used.
func (c *Client[C]) Post(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req, err := c.NewPostRequest(ctx, path, contentFormat, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create post request: %w", err)
	}
	defer c.cc.ReleaseMessage(req)
	return c.Do(req)
}

// NewPutRequest creates put request.
//
// Use ctx to set timeout.
//
// If payload is nil then content format is not used.
func (c *Client[C]) NewPutRequest(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req := c.cc.AcquireMessage(ctx)
	token, err := c.GetToken()
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	err = req.SetupPut(path, token, contentFormat, payload, opts...)
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	return req, nil
}

// Put issues a PUT to the specified path.
//
// Use ctx to set timeout.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
//
// If payload is nil then content format is not used.
func (c *Client[C]) Put(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req, err := c.NewPutRequest(ctx, path, contentFormat, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create put request: %w", err)
	}
	defer c.cc.ReleaseMessage(req)
	return c.Do(req)
}

// NewDeleteRequest creates delete request.
//
// Use ctx to set timeout.
func (c *Client[C]) NewDeleteRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req := c.cc.AcquireMessage(ctx)
	token, err := c.GetToken()
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	err = req.SetupDelete(path, token, opts...)
	if err != nil {
		c.cc.ReleaseMessage(req)
		return nil, err
	}
	return req, nil
}

// Delete deletes the resource identified by the request path.
//
// Use ctx to set timeout.
func (c *Client[C]) Delete(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req, err := c.NewDeleteRequest(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create delete request: %w", err)
	}
	defer c.cc.ReleaseMessage(req)
	return c.Do(req)
}

// Ping issues a PING to the client and waits for PONG response.
//
// Use ctx to set timeout.
func (c *Client[C]) Ping(ctx context.Context) error {
	resp := make(chan bool, 1)
	receivedPong := func() {
		select {
		case resp <- true:
		default:
		}
	}
	cancel, err := c.cc.AsyncPing(receivedPong)
	if err != nil {
		return err
	}
	defer cancel()
	select {
	case <-resp:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
