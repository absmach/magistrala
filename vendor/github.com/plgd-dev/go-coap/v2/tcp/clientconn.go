package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/keepalive"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"

	kitSync "github.com/plgd-dev/kit/sync"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapNet "github.com/plgd-dev/go-coap/v2/net"
)

var defaultDialOptions = dialOptions{
	ctx:            context.Background(),
	maxMessageSize: 64 * 1024,
	heartBeat:      time.Millisecond * 100,
	handler: func(w *ResponseWriter, r *pool.Message) {
		switch r.Code() {
		case codes.POST, codes.PUT, codes.GET, codes.DELETE:
			w.SetResponse(codes.NotFound, message.TextPlain, nil)
		}
	},
	errors: func(err error) {
		fmt.Println(err)
	},
	goPool: func(f func()) error {
		go func() {
			f()
		}()
		return nil
	},
	dialer:                   &net.Dialer{Timeout: time.Second * 3},
	keepalive:                keepalive.New(),
	net:                      "tcp",
	blockwiseSZX:             blockwise.SZX1024,
	blockwiseEnable:          true,
	blockwiseTransferTimeout: time.Second * 3,
}

type dialOptions struct {
	ctx                             context.Context
	maxMessageSize                  int
	heartBeat                       time.Duration
	handler                         HandlerFunc
	errors                          ErrorFunc
	goPool                          GoPoolFunc
	dialer                          *net.Dialer
	keepalive                       *keepalive.KeepAlive
	net                             string
	blockwiseSZX                    blockwise.SZX
	blockwiseEnable                 bool
	blockwiseTransferTimeout        time.Duration
	disablePeerTCPSignalMessageCSMs bool
	disableTCPSignalMessageCSM      bool
	tlsCfg                          *tls.Config
}

// A DialOption sets options such as credentials, keepalive parameters, etc.
type DialOption interface {
	applyDial(*dialOptions)
}

// ClientConn represents a virtual connection to a conceptual endpoint, to perform COAPs commands.
type ClientConn struct {
	noCopy
	session                 *Session
	observationTokenHandler *HandlerContainer
	observationRequests     *kitSync.Map
}

// Dial creates a client connection to the given target.
func Dial(target string, opts ...DialOption) (*ClientConn, error) {
	cfg := defaultDialOptions
	for _, o := range opts {
		o.applyDial(&cfg)
	}

	var conn net.Conn
	var err error
	if cfg.tlsCfg != nil {
		conn, err = tls.DialWithDialer(cfg.dialer, cfg.net, target, cfg.tlsCfg)
	} else {
		conn, err = cfg.dialer.DialContext(cfg.ctx, cfg.net, target)
	}
	if err != nil {
		return nil, err
	}
	return Client(conn, opts...), nil
}

func bwAcquireMessage(ctx context.Context) blockwise.Message {
	return pool.AcquireMessage(ctx)
}

func bwReleaseMessage(m blockwise.Message) {
	pool.ReleaseMessage(m.(*pool.Message))
}

func bwCreateHandlerFunc(observatioRequests *kitSync.Map) func(token message.Token) (blockwise.Message, bool) {
	return func(token message.Token) (blockwise.Message, bool) {
		msg, ok := observatioRequests.LoadWithFunc(token.String(), func(v interface{}) interface{} {
			r := v.(*pool.Message)
			d := pool.AcquireMessage(r.Context())
			d.ResetOptionsTo(r.Options())
			d.SetCode(r.Code())
			d.SetToken(r.Token())
			return d
		})
		if !ok {
			return nil, ok
		}
		bwMessage := msg.(blockwise.Message)
		return bwMessage, ok
	}
}

// Client creates client over tcp/tcp-tls connection.
func Client(conn net.Conn, opts ...DialOption) *ClientConn {
	cfg := defaultDialOptions
	for _, o := range opts {
		o.applyDial(&cfg)
	}
	if cfg.errors == nil {
		cfg.errors = func(error) {}
	}

	observatioRequests := kitSync.NewMap()
	var blockWise *blockwise.BlockWise
	if cfg.blockwiseEnable {
		blockWise = blockwise.NewBlockWise(
			bwAcquireMessage,
			bwReleaseMessage,
			cfg.blockwiseTransferTimeout,
			cfg.errors,
			false,
			bwCreateHandlerFunc(observatioRequests),
		)
	}

	observationTokenHandler := NewHandlerContainer()

	l := coapNet.NewConn(conn, coapNet.WithHeartBeat(cfg.heartBeat))
	session := NewSession(cfg.ctx,
		l,
		NewObservationHandler(observationTokenHandler, cfg.handler),
		cfg.maxMessageSize,
		cfg.goPool,
		cfg.errors,
		cfg.blockwiseSZX,
		blockWise,
		cfg.disablePeerTCPSignalMessageCSMs,
		cfg.disableTCPSignalMessageCSM,
	)
	cc := NewClientConn(session, observationTokenHandler, observatioRequests)

	go func() {
		err := cc.Run()
		if err != nil {
			cfg.errors(err)
		}
	}()
	if cfg.keepalive != nil {
		go func() {
			err := cfg.keepalive.Run(cc)
			if err != nil {
				cfg.errors(err)
			}
		}()
	}

	return cc
}

// NewClientConn creates connection over session and observation.
func NewClientConn(session *Session, observationTokenHandler *HandlerContainer, observationRequests *kitSync.Map) *ClientConn {
	return &ClientConn{
		session:                 session,
		observationTokenHandler: observationTokenHandler,
		observationRequests:     observationRequests,
	}
}

// Close closes connection without wait of ends Run function.
func (cc *ClientConn) Close() error {
	return cc.session.Close()
}

func (cc *ClientConn) do(req *pool.Message) (*pool.Message, error) {
	token := req.Token()
	if token == nil {
		return nil, fmt.Errorf("invalid token")
	}
	respChan := make(chan *pool.Message, 1)
	err := cc.session.TokenHandler().Insert(token, func(w *ResponseWriter, r *pool.Message) {
		r.Hijack()
		respChan <- r
	})
	if err != nil {
		return nil, fmt.Errorf("cannot add token handler: %w", err)
	}
	defer cc.session.TokenHandler().Pop(token)
	err = cc.session.WriteMessage(req)
	if err != nil {
		return nil, fmt.Errorf("cannot write request: %w", err)
	}

	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-cc.session.Context().Done():
		return nil, fmt.Errorf("connection was closed: %w", cc.Context().Err())
	case resp := <-respChan:
		return resp, nil
	}
}

// Do sends an coap message and returns an coap response.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
//
// Caller is responsible to release request and response.
func (cc *ClientConn) Do(req *pool.Message) (*pool.Message, error) {
	if !cc.session.PeerBlockWiseTransferEnabled() || cc.session.blockWise == nil {
		return cc.do(req)
	}
	bwresp, err := cc.session.blockWise.Do(req, cc.session.blockwiseSZX, cc.session.maxMessageSize, func(bwreq blockwise.Message) (blockwise.Message, error) {
		return cc.do(bwreq.(*pool.Message))
	})
	if err != nil {
		return nil, err
	}
	return bwresp.(*pool.Message), nil
}

func (cc *ClientConn) writeMessage(req *pool.Message) error {
	return cc.session.WriteMessage(req)
}

// WriteMessage sends an coap message.
func (cc *ClientConn) WriteMessage(req *pool.Message) error {
	if !cc.session.PeerBlockWiseTransferEnabled() || cc.session.blockWise == nil {
		return cc.writeMessage(req)
	}
	return cc.session.blockWise.WriteMessage(req, cc.session.blockwiseSZX, cc.session.maxMessageSize, func(bwreq blockwise.Message) error {
		return cc.writeMessage(bwreq.(*pool.Message))
	})
}

func newCommonRequest(ctx context.Context, code codes.Code, path string, opts ...message.Option) (*pool.Message, error) {
	token, err := message.GetToken()
	if err != nil {
		return nil, fmt.Errorf("cannot get token: %w", err)
	}
	req := pool.AcquireMessage(ctx)
	req.SetCode(code)
	req.SetToken(token)
	req.ResetOptionsTo(opts)
	req.SetPath(path)
	return req, nil
}

// NewGetRequest creates get request.
//
// Use ctx to set timeout.
func NewGetRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	return newCommonRequest(ctx, codes.GET, path, opts...)
}

// Get issues a GET to the specified path.
//
// Use ctx to set timeout.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
func (cc *ClientConn) Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req, err := NewGetRequest(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create get request: %w", err)
	}
	defer pool.ReleaseMessage(req)
	return cc.Do(req)
}

// NewPostRequest creates post request.
//
// Use ctx to set timeout.
//
// An error is returned if by failure to speak COAP (such as a network connectivity problem).
// Any status code doesn't cause an error.
//
// If payload is nil then content format is not used.
func NewPostRequest(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req, err := newCommonRequest(ctx, codes.POST, path, opts...)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.SetContentFormat(contentFormat)
		req.SetBody(payload)
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
func (cc *ClientConn) Post(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req, err := NewPostRequest(ctx, path, contentFormat, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create post request: %w", err)
	}
	defer pool.ReleaseMessage(req)
	return cc.Do(req)
}

// NewPutRequest creates put request.
//
// Use ctx to set timeout.
//
// If payload is nil then content format is not used.
func NewPutRequest(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req, err := newCommonRequest(ctx, codes.PUT, path, opts...)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.SetContentFormat(contentFormat)
		req.SetBody(payload)
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
func (cc *ClientConn) Put(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error) {
	req, err := NewPutRequest(ctx, path, contentFormat, payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create put request: %w", err)
	}
	defer pool.ReleaseMessage(req)
	return cc.Do(req)
}

// NewDeleteRequest creates delete request.
//
// Use ctx to set timeout.
func NewDeleteRequest(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	return newCommonRequest(ctx, codes.DELETE, path, opts...)
}

// Delete deletes the resource identified by the request path.
//
// Use ctx to set timeout.
func (cc *ClientConn) Delete(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	req, err := NewDeleteRequest(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create delete request: %w", err)
	}
	defer pool.ReleaseMessage(req)
	return cc.Do(req)
}

// Context returns the client's context.
//
// If connections was closed context is cancelled.
func (cc *ClientConn) Context() context.Context {
	return cc.session.Context()
}

// Ping issues a PING to the client and waits for PONG reponse.
//
// Use ctx to set timeout.
func (cc *ClientConn) Ping(ctx context.Context) error {
	token, err := message.GetToken()
	if err != nil {
		return fmt.Errorf("cannot get token: %w", err)
	}
	req := pool.AcquireMessage(ctx)
	req.SetToken(token)
	req.SetCode(codes.Ping)
	defer pool.ReleaseMessage(req)
	resp, err := cc.Do(req)
	if err != nil {
		return err
	}
	defer pool.ReleaseMessage(resp)
	if resp.Code() == codes.Pong {
		return nil
	}
	return fmt.Errorf("unexpected code(%v)", resp.Code())
}

// Run reads and process requests from a connection, until the connection is not closed.
func (cc *ClientConn) Run() (err error) {
	return cc.session.Run(cc)
}

// AddOnClose calls function on close connection event.
func (cc *ClientConn) AddOnClose(f EventFunc) {
	cc.session.AddOnClose(f)
}

// RemoteAddr gets remote address.
func (cc *ClientConn) RemoteAddr() net.Addr {
	return cc.session.connection.RemoteAddr()
}

// Client get instance which implements mux.Client.
func (cc *ClientConn) Client() *ClientTCP {
	return NewClientTCP(cc)
}

// Sequence acquires sequence number.
func (cc *ClientConn) Sequence() uint64 {
	return cc.session.Sequence()
}
