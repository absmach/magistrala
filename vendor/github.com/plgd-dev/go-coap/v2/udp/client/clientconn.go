package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	atomicTypes "go.uber.org/atomic"

	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"

	"github.com/plgd-dev/go-coap/v2/message/codes"
	udpMessage "github.com/plgd-dev/go-coap/v2/udp/message"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
	kitSync "github.com/plgd-dev/kit/sync"
)

type HandlerFunc = func(*ResponseWriter, *pool.Message)
type ErrorFunc = func(error)
type GoPoolFunc = func(func()) error
type EventFunc = func()
type GetMIDFunc = func() uint16

type Session interface {
	Context() context.Context
	Close() error
	MaxMessageSize() int
	RemoteAddr() net.Addr
	WriteMessage(req *pool.Message) error
	Run(cc *ClientConn) error
	AddOnClose(f EventFunc)
	SetContextValue(key interface{}, val interface{})
}

type Notifier interface {
	Notify()
}

// ClientConn represents a virtual connection to a conceptual endpoint, to perform COAPs commands.
type ClientConn struct {
	// This field needs to be the first in the struct to ensure proper word alignment on 32-bit platforms.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	sequence                uint64
	session                 Session
	handler                 HandlerFunc
	observationTokenHandler *HandlerContainer
	observationRequests     *kitSync.Map
	transmission            *Transmission
	blockwiseSZX            blockwise.SZX
	blockWise               *blockwise.BlockWise
	goPool                  GoPoolFunc
	errors                  ErrorFunc
	getMID                  GetMIDFunc
	responseMsgCache        *cache.Cache
	msgIdMutex              *MutexMap
	activityMonitor         Notifier

	tokenHandlerContainer *HandlerContainer
	midHandlerContainer   *HandlerContainer
}

// Transmission is a threadsafe container for transmission related parameters
type Transmission struct {
	nStart             *atomicTypes.Duration
	acknowledgeTimeout *atomicTypes.Duration
	maxRetransmit      *atomicTypes.Int32
}

func (t *Transmission) SetTransmissionNStart(d time.Duration) {
	t.nStart.Store(d)
}

func (t *Transmission) SetTransmissionAcknowledgeTimeout(d time.Duration) {
	t.acknowledgeTimeout.Store(d)
}

func (t *Transmission) SetTransmissionMaxRetransmit(d int32) {
	t.maxRetransmit.Store(d)
}

func (cc *ClientConn) Transmission() *Transmission {
	return cc.transmission
}

// NewClientConn creates connection over session and observation.
func NewClientConn(
	session Session,
	observationTokenHandler *HandlerContainer,
	observationRequests *kitSync.Map,
	transmissionNStart time.Duration,
	transmissionAcknowledgeTimeout time.Duration,
	transmissionMaxRetransmit int,
	handler HandlerFunc,
	blockwiseSZX blockwise.SZX,
	blockWise *blockwise.BlockWise,
	goPool GoPoolFunc,
	errors ErrorFunc,
	getMID GetMIDFunc,
	activityMonitor Notifier,
) *ClientConn {
	if errors == nil {
		errors = func(error) {}
	}
	if getMID == nil {
		getMID = udpMessage.GetMID
	}
	if activityMonitor == nil {
		activityMonitor = &nilNotifier{}
	}

	return &ClientConn{
		session:                 session,
		observationTokenHandler: observationTokenHandler,
		observationRequests:     observationRequests,
		transmission: &Transmission{
			atomicTypes.NewDuration(transmissionNStart),
			atomicTypes.NewDuration(transmissionAcknowledgeTimeout),
			atomicTypes.NewInt32(int32(transmissionMaxRetransmit)),
		},
		handler:      handler,
		blockwiseSZX: blockwiseSZX,
		blockWise:    blockWise,

		tokenHandlerContainer: NewHandlerContainer(),
		midHandlerContainer:   NewHandlerContainer(),
		goPool:                goPool,
		errors:                errors,
		getMID:                getMID,
		// EXCHANGE_LIFETIME = 247
		responseMsgCache: cache.New(247*time.Second, 60*time.Second),
		msgIdMutex:       NewMutexMap(),
		activityMonitor:  activityMonitor,
	}
}

func (cc *ClientConn) Session() Session {
	return cc.session
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
	err := cc.tokenHandlerContainer.Insert(token, func(w *ResponseWriter, r *pool.Message) {
		r.Hijack()
		select {
		case respChan <- r:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("cannot add token handler: %w", err)
	}
	defer cc.tokenHandlerContainer.Pop(token)
	err = cc.writeMessage(req)
	if err != nil {
		return nil, fmt.Errorf("cannot write request: %w", err)
	}
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-cc.session.Context().Done():
		return nil, fmt.Errorf("connection was closed: %w", cc.session.Context().Err())
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
	if cc.blockWise == nil {
		return cc.do(req)
	}
	bwresp, err := cc.blockWise.Do(req, cc.blockwiseSZX, cc.session.MaxMessageSize(), func(bwreq blockwise.Message) (blockwise.Message, error) {
		return cc.do(bwreq.(*pool.Message))
	})
	if err != nil {
		return nil, err
	}
	return bwresp.(*pool.Message), nil
}

func (cc *ClientConn) writeMessage(req *pool.Message) error {
	req.SetMessageID(cc.getMID())
	respChan := make(chan struct{})

	// Only confirmable messages ever match an message ID
	if req.Type() == udpMessage.Confirmable {
		err := cc.midHandlerContainer.Insert(req.MessageID(), func(w *ResponseWriter, r *pool.Message) {
			close(respChan)
			if r.IsSeparate() {
				// separate message - just accept
				return
			}
			cc.handleBW(w, r)
		})
		if err != nil {
			return fmt.Errorf("cannot insert mid handler: %w", err)
		}
		defer cc.midHandlerContainer.Pop(req.MessageID())
	}

	err := cc.session.WriteMessage(req)
	if err != nil {
		return fmt.Errorf("cannot write request: %w", err)
	}
	if req.Type() != udpMessage.Confirmable {
		// If the request is not confirmable, we do not need to wait for a response
		// and skip retransmissions
		close(respChan)
	}

	maxRetransmit := cc.transmission.maxRetransmit.Load()
	for i := int32(0); i < maxRetransmit; i++ {
		select {
		case <-respChan:
			return nil
		case <-req.Context().Done():
			return req.Context().Err()
		case <-cc.Context().Done():
			return fmt.Errorf("connection was closed: %w", cc.Context().Err())
		case <-time.After(cc.transmission.acknowledgeTimeout.Load()):
			select {
			case <-req.Context().Done():
				return req.Context().Err()
			case <-cc.session.Context().Done():
				return fmt.Errorf("connection was closed: %w", cc.Context().Err())
			case <-time.After(cc.transmission.nStart.Load()):
				err = cc.session.WriteMessage(req)
				if err != nil {
					return fmt.Errorf("cannot write request: %w", err)
				}
			}
		}
	}
	return fmt.Errorf("timeout: retransmision(%v) was exhausted", cc.transmission.maxRetransmit.Load())
}

// WriteMessage sends an coap message.
func (cc *ClientConn) WriteMessage(req *pool.Message) error {
	if cc.blockWise == nil {
		return cc.writeMessage(req)
	}
	return cc.blockWise.WriteMessage(req, cc.blockwiseSZX, cc.session.MaxMessageSize(), func(bwreq blockwise.Message) error {
		return cc.writeMessage(bwreq.(*pool.Message))
	})
}

func (cc *ClientConn) doWithMID(req *pool.Message) (*pool.Message, error) {
	respChan := make(chan *pool.Message, 1)
	err := cc.midHandlerContainer.Insert(req.MessageID(), func(w *ResponseWriter, r *pool.Message) {
		r.Hijack()
		select {
		case respChan <- r:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("cannot insert mid handler: %w", err)
	}
	defer cc.midHandlerContainer.Pop(req.MessageID())
	err = cc.session.WriteMessage(req)
	if err != nil {
		return nil, fmt.Errorf("cannot write request: %w", err)
	}

	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-cc.session.Context().Done():
		return nil, fmt.Errorf("connection was closed: %w", cc.session.Context().Err())
	case resp := <-respChan:
		return resp, nil
	}
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
	req.SetType(udpMessage.Confirmable)
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
	resp := make(chan bool, 1)
	receivedPong := func() {
		select {
		case resp <- true:
		default:
		}
	}
	cancel, err := cc.AsyncPing(receivedPong)
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

// AsyncPing sends ping and receivedPong will be called when pong arrives. It returns cancellation of ping operation.
func (cc *ClientConn) AsyncPing(receivedPong func()) (func(), error) {
	req := pool.AcquireMessage(cc.Context())
	defer pool.ReleaseMessage(req)
	req.SetType(udpMessage.Confirmable)
	req.SetCode(codes.Empty)
	mid := cc.getMID()
	req.SetMessageID(mid)
	err := cc.midHandlerContainer.Insert(mid, func(w *ResponseWriter, r *pool.Message) {
		if r.Type() == udpMessage.Reset || r.Type() == udpMessage.Acknowledgement {
			receivedPong()
		}
	})
	if err != nil {
		return nil, fmt.Errorf("cannot insert mid handler: %w", err)
	}
	err = cc.session.WriteMessage(req)
	if err != nil {
		cc.midHandlerContainer.Pop(mid)
		return nil, fmt.Errorf("cannot write request: %w", err)
	}
	return func() {
		cc.midHandlerContainer.Pop(mid)
	}, nil
}

// Run reads and process requests from a connection, until the connection is closed.
func (cc *ClientConn) Run() error {
	return cc.session.Run(cc)
}

// AddOnClose calls function on close connection event.
func (cc *ClientConn) AddOnClose(f EventFunc) {
	cc.session.AddOnClose(f)
}

func (cc *ClientConn) RemoteAddr() net.Addr {
	return cc.session.RemoteAddr()
}

func (cc *ClientConn) sendPong(w *ResponseWriter, r *pool.Message) {
	w.SetResponse(codes.Empty, message.TextPlain, nil)
}

type bwResponseWriter struct {
	w *ResponseWriter
}

func (b *bwResponseWriter) Message() blockwise.Message {
	return b.w.response
}

func (b *bwResponseWriter) SetMessage(m blockwise.Message) {
	pool.ReleaseMessage(b.w.response)
	b.w.response = m.(*pool.Message)
}

func (cc *ClientConn) handleBW(w *ResponseWriter, r *pool.Message) {
	if cc.blockWise != nil {
		bwr := bwResponseWriter{
			w: w,
		}
		cc.blockWise.Handle(&bwr, r, cc.blockwiseSZX, cc.session.MaxMessageSize(), func(bw blockwise.ResponseWriter, br blockwise.Message) {
			h, err := cc.tokenHandlerContainer.Pop(r.Token())
			w := bw.(*bwResponseWriter).w
			r := br.(*pool.Message)
			if err == nil {
				h(w, r)
				return
			}
			cc.handler(w, r)
		})
		return
	}
	h, err := cc.tokenHandlerContainer.Pop(r.Token())
	if err == nil {
		h(w, r)
		return
	}
	cc.handler(w, r)
}

func (cc *ClientConn) handle(w *ResponseWriter, r *pool.Message) {
	if r.Code() == codes.Empty && r.Type() == udpMessage.Confirmable && len(r.Token()) == 0 && len(r.Options()) == 0 && r.Body() == nil {
		cc.sendPong(w, r)
		return
	}
	h, err := cc.midHandlerContainer.Pop(r.MessageID())
	if err == nil {
		h(w, r)
		return
	}
	if r.IsSeparate() {
		// msg was processed by token handler - just drop it.
		return
	}
	cc.handleBW(w, r)
}

// Sequence acquires sequence number.
func (cc *ClientConn) Sequence() uint64 {
	return atomic.AddUint64(&cc.sequence, 1)
}

func (cc *ClientConn) addResponseToCache(resp *pool.Message) error {
	marshaledResp, err := resp.Marshal()
	if err != nil {
		return err
	}
	cacheMsg := make([]byte, len(marshaledResp))
	copy(cacheMsg, marshaledResp)
	cc.responseMsgCache.SetDefault(fmt.Sprintf("%d", resp.MessageID()), cacheMsg)
	return nil
}

func (cc *ClientConn) getResponseFromCache(mid uint16, resp *pool.Message) (bool, error) {
	cachedResp, _ := cc.responseMsgCache.Get(fmt.Sprintf("%d", mid))
	if rawMsg, ok := cachedResp.([]byte); ok {
		_, err := resp.Unmarshal(rawMsg)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (cc *ClientConn) Process(datagram []byte) error {
	if cc.session.MaxMessageSize() >= 0 && len(datagram) > cc.session.MaxMessageSize() {
		return fmt.Errorf("max message size(%v) was exceeded %v", cc.session.MaxMessageSize(), len(datagram))
	}
	req := pool.AcquireMessage(cc.Context())
	_, err := req.Unmarshal(datagram)
	if err != nil {
		pool.ReleaseMessage(req)
		return err
	}
	req.SetSequence(cc.Sequence())
	cc.activityMonitor.Notify()
	cc.goPool(func() {
		defer cc.activityMonitor.Notify()
		reqMid := req.MessageID()

		// The same message ID can not be handled concurrently
		// for deduplication to work
		l := cc.msgIdMutex.Lock(reqMid)
		defer l.Unlock()

		origResp := pool.AcquireMessage(cc.Context())
		origResp.SetToken(req.Token())
		// If a request is sent in a Non-confirmable message, then the response
		// is sent using a new Non-confirmable message, although the server may
		// instead send a Confirmable message.
		origResp.SetType(req.Type())
		w := NewResponseWriter(origResp, cc, req.Options())

		if ok, err := cc.getResponseFromCache(req.MessageID(), w.response); ok {
			defer pool.ReleaseMessage(w.response)
			if !req.IsHijacked() {
				pool.ReleaseMessage(req)
			}
			err = cc.session.WriteMessage(w.response)
			if err != nil {
				cc.Close()
				cc.errors(fmt.Errorf("cannot write response: %w", err))
				return
			}
			return
		} else if err != nil {
			cc.Close()
			cc.errors(fmt.Errorf("cannot unmarshal response from cache: %w", err))
			return
		}

		reqType := req.Type()
		origResp.SetModified(false)
		cc.handle(w, req)

		defer pool.ReleaseMessage(w.response)
		if !req.IsHijacked() {
			pool.ReleaseMessage(req)
		}
		if w.response.IsModified() {
			switch {
			case w.response.Type() == udpMessage.Reset:
				w.response.SetMessageID(reqMid)
			case reqType == udpMessage.Confirmable:
				w.response.SetType(udpMessage.Acknowledgement)
				w.response.SetMessageID(reqMid)
			default:
				w.response.SetType(udpMessage.NonConfirmable)
				w.response.SetMessageID(cc.getMID())
			}
			err := cc.session.WriteMessage(w.response)
			if err != nil {
				cc.Close()
				cc.errors(fmt.Errorf("cannot write response: %w", err))
				return
			}
		} else if reqType == udpMessage.Confirmable {
			w.response.Reset()
			w.response.SetCode(codes.Empty)
			w.response.SetType(udpMessage.Acknowledgement)
			w.response.SetMessageID(reqMid)
			err := cc.session.WriteMessage(w.response)
			if err != nil {
				cc.Close()
				cc.errors(fmt.Errorf("cannot write ack reponse: %w", err))
				return
			}
		}

		err = cc.addResponseToCache(w.response)
		if err != nil {
			cc.Close()
			cc.errors(fmt.Errorf("cannot cache response: %w", err))
			return
		}
	})
	return nil
}

func (cc *ClientConn) Client() *Client {
	return NewClient(cc)
}

// SetContextValue stores the value associated with key to context of connection.
func (cc *ClientConn) SetContextValue(key interface{}, val interface{}) {
	cc.session.SetContextValue(key, val)
}
