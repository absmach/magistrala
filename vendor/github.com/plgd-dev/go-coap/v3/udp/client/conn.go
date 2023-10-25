package client

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/net/client"
	limitparallelrequests "github.com/plgd-dev/go-coap/v3/net/client/limitParallelRequests"
	"github.com/plgd-dev/go-coap/v3/net/observation"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/options/config"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	coapErrors "github.com/plgd-dev/go-coap/v3/pkg/errors"
	"github.com/plgd-dev/go-coap/v3/pkg/fn"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/go-coap/v3/udp/coder"
	"go.uber.org/atomic"
	"golang.org/x/sync/semaphore"
)

// https://datatracker.ietf.org/doc/html/rfc7252#section-4.8.2
const ExchangeLifetime = 247 * time.Second

type (
	HandlerFunc                 = func(*responsewriter.ResponseWriter[*Conn], *pool.Message)
	ErrorFunc                   = func(error)
	EventFunc                   = func()
	GetMIDFunc                  = func() int32
	CreateInactivityMonitorFunc = func() InactivityMonitor
)

type InactivityMonitor interface {
	Notify()
	CheckInactivity(now time.Time, cc *Conn)
}

type Session interface {
	Context() context.Context
	Close() error
	MaxMessageSize() uint32
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	// NetConn returns the underlying connection that is wrapped by Session. The Conn returned is shared by all invocations of NetConn, so do not modify it.
	NetConn() net.Conn
	WriteMessage(req *pool.Message) error
	// WriteMulticast sends multicast to the remote multicast address.
	// By default it is sent over all network interfaces and all compatible source IP addresses with hop limit 1.
	// Via opts you can specify the network interface, source IP address, and hop limit.
	WriteMulticastMessage(req *pool.Message, address *net.UDPAddr, opts ...coapNet.MulticastOption) error
	Run(cc *Conn) error
	AddOnClose(f EventFunc)
	SetContextValue(key interface{}, val interface{})
	Done() <-chan struct{}
}

type RequestsMap = coapSync.Map[uint64, *pool.Message]

const (
	errFmtWriteRequest  = "cannot write request: %w"
	errFmtWriteResponse = "cannot write response: %w"
)

type midElement struct {
	handler    HandlerFunc
	start      time.Time
	deadline   time.Time
	retransmit atomic.Int32

	private struct {
		sync.Mutex
		msg *pool.Message
	}
}

func (m *midElement) ReleaseMessage(cc *Conn) {
	m.private.Lock()
	defer m.private.Unlock()
	if m.private.msg != nil {
		cc.ReleaseMessage(m.private.msg)
		m.private.msg = nil
	}
}

func (m *midElement) IsExpired(now time.Time, maxRetransmit int32) bool {
	if !m.deadline.IsZero() && now.After(m.deadline) {
		// remove element if deadline is exceeded
		return true
	}
	retransmit := m.retransmit.Load()
	return retransmit >= maxRetransmit
}

func (m *midElement) Retransmit(now time.Time, acknowledgeTimeout time.Duration) bool {
	if now.After(m.start.Add(acknowledgeTimeout * time.Duration(m.retransmit.Load()+1))) {
		m.retransmit.Inc()
		// retransmit
		return true
	}
	// wait for next retransmit
	return false
}

func (m *midElement) GetMessage(cc *Conn) (*pool.Message, bool, error) {
	m.private.Lock()
	defer m.private.Unlock()
	if m.private.msg == nil {
		return nil, false, nil
	}
	msg := cc.AcquireMessage(m.private.msg.Context())
	if err := m.private.msg.Clone(msg); err != nil {
		cc.ReleaseMessage(msg)
		return nil, false, err
	}
	return msg, true, nil
}

// Conn represents a virtual connection to a conceptual endpoint, to perform COAPs commands.
type Conn struct {
	// This field needs to be the first in the struct to ensure proper word alignment on 32-bit platforms.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	sequence atomic.Uint64

	session Session
	*client.Client[*Conn]
	inactivityMonitor InactivityMonitor

	blockWise          *blockwise.BlockWise[*Conn]
	observationHandler *observation.Handler[*Conn]
	transmission       *Transmission
	messagePool        *pool.Pool

	processReceivedMessage config.ProcessReceivedMessageFunc[*Conn]
	errors                 ErrorFunc
	responseMsgCache       *cache.Cache[string, []byte]
	msgIDMutex             *MutexMap

	tokenHandlerContainer *coapSync.Map[uint64, HandlerFunc]
	midHandlerContainer   *coapSync.Map[int32, *midElement]
	msgID                 atomic.Uint32
	blockwiseSZX          blockwise.SZX

	/*
		An outstanding interaction is either a CON for which an ACK has not
		yet been received but is still expected (message layer) or a request
		for which neither a response nor an Acknowledgment message has yet
		been received but is still expected (which may both occur at the same
		time, counting as one outstanding interaction).
	*/
	numOutstandingInteraction *semaphore.Weighted
	receivedMessageReader     *client.ReceivedMessageReader[*Conn]
}

// Transmission is a threadsafe container for transmission related parameters
type Transmission struct {
	nStart             *atomic.Uint32
	acknowledgeTimeout *atomic.Duration
	maxRetransmit      *atomic.Int32
}

// SetTransmissionNStart changing the nStart value will only effect requests queued after the change. The requests waiting here already before the change will get unblocked when enough weight has been released.
func (t *Transmission) SetTransmissionNStart(d uint32) {
	t.nStart.Store(d)
}

func (t *Transmission) SetTransmissionAcknowledgeTimeout(d time.Duration) {
	t.acknowledgeTimeout.Store(d)
}

func (t *Transmission) SetTransmissionMaxRetransmit(d int32) {
	t.maxRetransmit.Store(d)
}

func (cc *Conn) Transmission() *Transmission {
	return cc.transmission
}

// NewConn creates connection over session and observation.
func NewConn(
	session Session,
	createBlockWise func(cc *Conn) *blockwise.BlockWise[*Conn],
	inactivityMonitor InactivityMonitor,
	cfg *Config,
) *Conn {
	if cfg.Errors == nil {
		cfg.Errors = func(error) {
			// default no-op
		}
	}
	if cfg.GetMID == nil {
		cfg.GetMID = message.GetMID
	}
	if cfg.GetToken == nil {
		cfg.GetToken = message.GetToken
	}
	if cfg.ReceivedMessageQueueSize < 0 {
		cfg.ReceivedMessageQueueSize = 0
	}

	cc := Conn{
		session: session,
		transmission: &Transmission{
			atomic.NewUint32(cfg.TransmissionNStart),
			atomic.NewDuration(cfg.TransmissionAcknowledgeTimeout),
			atomic.NewInt32(int32(cfg.TransmissionMaxRetransmit)),
		},
		blockwiseSZX: cfg.BlockwiseSZX,

		tokenHandlerContainer:     coapSync.NewMap[uint64, HandlerFunc](),
		midHandlerContainer:       coapSync.NewMap[int32, *midElement](),
		processReceivedMessage:    cfg.ProcessReceivedMessage,
		errors:                    cfg.Errors,
		msgIDMutex:                NewMutexMap(),
		responseMsgCache:          cache.NewCache[string, []byte](),
		inactivityMonitor:         inactivityMonitor,
		messagePool:               cfg.MessagePool,
		numOutstandingInteraction: semaphore.NewWeighted(math.MaxInt64),
	}
	cc.msgID.Store(uint32(cfg.GetMID() - 0xffff/2))
	cc.blockWise = createBlockWise(&cc)
	limitParallelRequests := limitparallelrequests.New(cfg.LimitClientParallelRequests, cfg.LimitClientEndpointParallelRequests, cc.do, cc.doObserve)
	cc.observationHandler = observation.NewHandler(&cc, cfg.Handler, limitParallelRequests.Do)
	cc.Client = client.New(&cc, cc.observationHandler, cfg.GetToken, limitParallelRequests)
	if cc.processReceivedMessage == nil {
		cc.processReceivedMessage = processReceivedMessage
	}
	cc.receivedMessageReader = client.NewReceivedMessageReader(&cc, cfg.ReceivedMessageQueueSize)
	return &cc
}

func processReceivedMessage(req *pool.Message, cc *Conn, handler config.HandlerFunc[*Conn]) {
	cc.ProcessReceivedMessageWithHandler(req, handler)
}

func (cc *Conn) ProcessReceivedMessage(req *pool.Message) {
	cc.processReceivedMessage(req, cc, cc.handleReq)
}

func (cc *Conn) Session() Session {
	return cc.session
}

func (cc *Conn) GetMessageID() int32 {
	// To prevent collisions during reconnections, it is important to always increment the global counter.
	// For example, if a connection (cc) is established and later closed due to inactivity, a new cc may
	// be created shortly after. However, if the new cc is initialized with the same message ID as the
	// previous one, the receiver may mistakenly treat the incoming message as a duplicate and discard it.
	// Hence, by incrementing the global counter, we can ensure unique message IDs and avoid such issues.
	message.GetMID()
	return int32(uint16(cc.msgID.Inc()))
}

// Close closes connection without waiting for the end of the Run function.
func (cc *Conn) Close() error {
	err := cc.session.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (cc *Conn) doInternal(req *pool.Message) (*pool.Message, error) {
	token := req.Token()
	if token == nil {
		return nil, fmt.Errorf("invalid token")
	}

	respChan := make(chan *pool.Message, 1)
	if _, loaded := cc.tokenHandlerContainer.LoadOrStore(token.Hash(), func(w *responsewriter.ResponseWriter[*Conn], r *pool.Message) {
		r.Hijack()
		select {
		case respChan <- r:
		default:
		}
	}); loaded {
		return nil, fmt.Errorf("cannot add token(%v) handler: %w", token, coapErrors.ErrKeyAlreadyExists)
	}
	defer func() {
		_, _ = cc.tokenHandlerContainer.LoadAndDelete(token.Hash())
	}()
	err := cc.writeMessage(req)
	if err != nil {
		return nil, fmt.Errorf(errFmtWriteRequest, err)
	}
	cc.receivedMessageReader.TryToReplaceLoop()
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-cc.Context().Done():
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
func (cc *Conn) do(req *pool.Message) (*pool.Message, error) {
	if cc.blockWise == nil {
		return cc.doInternal(req)
	}
	resp, err := cc.blockWise.Do(req, cc.blockwiseSZX, cc.session.MaxMessageSize(), func(bwReq *pool.Message) (*pool.Message, error) {
		if bwReq.Options().HasOption(message.Block1) || bwReq.Options().HasOption(message.Block2) {
			bwReq.SetMessageID(cc.GetMessageID())
		}
		return cc.doInternal(bwReq)
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DoObserve subscribes for every change with request.
func (cc *Conn) doObserve(req *pool.Message, observeFunc func(req *pool.Message)) (client.Observation, error) {
	return cc.observationHandler.NewObservation(req, observeFunc)
}

func (cc *Conn) releaseOutstandingInteraction() {
	cc.numOutstandingInteraction.Release(1)
}

func (cc *Conn) acquireOutstandingInteraction(ctx context.Context) error {
	nStart := cc.Transmission().nStart.Load()
	if nStart == 0 {
		return fmt.Errorf("invalid NStart value %v", nStart)
	}
	n := math.MaxInt64 - int64(cc.Transmission().nStart.Load()) + 1
	err := cc.numOutstandingInteraction.Acquire(ctx, n)
	if err != nil {
		return err
	}
	cc.numOutstandingInteraction.Release(n - 1)
	return nil
}

func (cc *Conn) waitForAcknowledge(req *pool.Message, waitForResponseChan chan struct{}) error {
	cc.receivedMessageReader.TryToReplaceLoop()
	select {
	case <-waitForResponseChan:
		return nil
	case <-req.Context().Done():
		return req.Context().Err()
	case <-cc.Context().Done():
		return fmt.Errorf("connection was closed: %w", cc.Context().Err())
	}
}

func (cc *Conn) prepareWriteMessage(req *pool.Message, handler HandlerFunc) (func(), error) {
	var closeFns fn.FuncList

	// Only confirmable messages ever match an message ID
	switch req.Type() {
	case message.Confirmable:
		msg := cc.AcquireMessage(req.Context())
		if err := req.Clone(msg); err != nil {
			cc.ReleaseMessage(msg)
			return nil, fmt.Errorf("cannot clone message: %w", err)
		}
		if req.Code() >= codes.GET && req.Code() <= codes.DELETE {
			if err := cc.acquireOutstandingInteraction(req.Context()); err != nil {
				return nil, err
			}
			closeFns = append(closeFns, func() {
				cc.releaseOutstandingInteraction()
			})
		}
		deadline, _ := req.Context().Deadline()
		if _, loaded := cc.midHandlerContainer.LoadOrStore(req.MessageID(), &midElement{
			handler:  handler,
			start:    time.Now(),
			deadline: deadline,
			private: struct {
				sync.Mutex
				msg *pool.Message
			}{msg: msg},
		}); loaded {
			closeFns.Execute()
			return nil, fmt.Errorf("cannot insert mid(%v) handler: %w", req.MessageID(), coapErrors.ErrKeyAlreadyExists)
		}
		closeFns = append(closeFns, func() {
			_, _ = cc.midHandlerContainer.LoadAndDelete(req.MessageID())
		})
	case message.NonConfirmable:
		/* TODO need to acquireOutstandingInteraction
		if req.Code() >= codes.GET && req.Code() <= codes.DELETE {
		}
		*/
	}
	return closeFns.ToFunction(), nil
}

func (cc *Conn) writeMessageAsync(req *pool.Message) error {
	req.UpsertType(message.Confirmable)
	req.UpsertMessageID(cc.GetMessageID())
	closeFn, err := cc.prepareWriteMessage(req, func(w *responsewriter.ResponseWriter[*Conn], r *pool.Message) {
		// do nothing
	})
	if err != nil {
		return err
	}
	defer closeFn()
	if err := cc.session.WriteMessage(req); err != nil {
		return fmt.Errorf(errFmtWriteRequest, err)
	}
	return nil
}

func (cc *Conn) writeMessage(req *pool.Message) error {
	req.UpsertType(message.Confirmable)
	req.UpsertMessageID(cc.GetMessageID())
	if req.Type() != message.Confirmable {
		return cc.writeMessageAsync(req)
	}
	respChan := make(chan struct{})
	closeFn, err := cc.prepareWriteMessage(req, func(w *responsewriter.ResponseWriter[*Conn], r *pool.Message) {
		close(respChan)
	})
	if err != nil {
		return err
	}
	defer closeFn()
	if err := cc.session.WriteMessage(req); err != nil {
		return fmt.Errorf(errFmtWriteRequest, err)
	}
	if err := cc.waitForAcknowledge(req, respChan); err != nil {
		return fmt.Errorf(errFmtWriteRequest, err)
	}
	return nil
}

// WriteMessage sends an coap message.
func (cc *Conn) WriteMessage(req *pool.Message) error {
	if cc.blockWise == nil {
		return cc.writeMessage(req)
	}
	return cc.blockWise.WriteMessage(req, cc.blockwiseSZX, cc.session.MaxMessageSize(), func(bwReq *pool.Message) error {
		if bwReq.Options().HasOption(message.Block1) || bwReq.Options().HasOption(message.Block2) {
			bwReq.SetMessageID(cc.GetMessageID())
		}
		return cc.writeMessage(bwReq)
	})
}

// Context returns the client's context.
//
// If connections was closed context is cancelled.
func (cc *Conn) Context() context.Context {
	return cc.session.Context()
}

// AsyncPing sends ping and receivedPong will be called when pong arrives. It returns cancellation of ping operation.
func (cc *Conn) AsyncPing(receivedPong func()) (func(), error) {
	req := cc.AcquireMessage(cc.Context())
	req.SetType(message.Confirmable)
	req.SetCode(codes.Empty)
	mid := cc.GetMessageID()
	req.SetMessageID(mid)
	if _, loaded := cc.midHandlerContainer.LoadOrStore(mid, &midElement{
		handler: func(w *responsewriter.ResponseWriter[*Conn], r *pool.Message) {
			if r.Type() == message.Reset || r.Type() == message.Acknowledgement {
				receivedPong()
			}
		},
		start:    time.Now(),
		deadline: time.Time{}, // no deadline
		private: struct {
			sync.Mutex
			msg *pool.Message
		}{msg: req},
	}); loaded {
		return nil, fmt.Errorf("cannot insert mid(%v) handler: %w", mid, coapErrors.ErrKeyAlreadyExists)
	}
	removeMidHandler := func() {
		if elem, ok := cc.midHandlerContainer.LoadAndDelete(mid); ok {
			elem.ReleaseMessage(cc)
		}
	}
	if err := cc.session.WriteMessage(req); err != nil {
		removeMidHandler()
		return nil, fmt.Errorf(errFmtWriteRequest, err)
	}
	return removeMidHandler, nil
}

// Run reads and process requests from a connection, until the connection is closed.
func (cc *Conn) Run() error {
	return cc.session.Run(cc)
}

// AddOnClose calls function on close connection event.
func (cc *Conn) AddOnClose(f EventFunc) {
	cc.session.AddOnClose(f)
}

func (cc *Conn) RemoteAddr() net.Addr {
	return cc.session.RemoteAddr()
}

func (cc *Conn) LocalAddr() net.Addr {
	return cc.session.LocalAddr()
}

func (cc *Conn) sendPong(w *responsewriter.ResponseWriter[*Conn], r *pool.Message) {
	if err := w.SetResponse(codes.Empty, message.TextPlain, nil); err != nil {
		cc.errors(fmt.Errorf("cannot send pong response: %w", err))
	}
	if r.Type() == message.Confirmable {
		w.Message().SetType(message.Acknowledgement)
		w.Message().SetMessageID(r.MessageID())
	} else {
		if w.Message().Type() != message.Reset {
			w.Message().SetType(message.NonConfirmable)
		}
		w.Message().SetMessageID(cc.GetMessageID())
	}
}

func (cc *Conn) handle(w *responsewriter.ResponseWriter[*Conn], m *pool.Message) {
	if m.IsSeparateMessage() {
		// msg was processed by token handler - just drop it.
		return
	}
	if cc.blockWise != nil {
		cc.blockWise.Handle(w, m, cc.blockwiseSZX, cc.session.MaxMessageSize(), func(rw *responsewriter.ResponseWriter[*Conn], rm *pool.Message) {
			if h, ok := cc.tokenHandlerContainer.LoadAndDelete(rm.Token().Hash()); ok {
				h(rw, rm)
				return
			}
			cc.observationHandler.Handle(rw, rm)
		})
		return
	}
	if h, ok := cc.tokenHandlerContainer.LoadAndDelete(m.Token().Hash()); ok {
		h(w, m)
		return
	}
	cc.observationHandler.Handle(w, m)
}

// Sequence acquires sequence number.
func (cc *Conn) Sequence() uint64 {
	return cc.sequence.Add(1)
}

func (cc *Conn) responseMsgCacheID(msgID int32) string {
	return fmt.Sprintf("resp-%v-%d", cc.RemoteAddr(), msgID)
}

func (cc *Conn) addResponseToCache(resp *pool.Message) error {
	marshaledResp, err := resp.MarshalWithEncoder(coder.DefaultCoder)
	if err != nil {
		return err
	}
	cacheMsg := make([]byte, len(marshaledResp))
	copy(cacheMsg, marshaledResp)
	cc.responseMsgCache.LoadOrStore(cc.responseMsgCacheID(resp.MessageID()), cache.NewElement(cacheMsg, time.Now().Add(ExchangeLifetime), nil))
	return nil
}

func (cc *Conn) getResponseFromCache(mid int32, resp *pool.Message) (bool, error) {
	cachedResp := cc.responseMsgCache.Load(cc.responseMsgCacheID(mid))
	if cachedResp == nil {
		return false, nil
	}
	if rawMsg := cachedResp.Data(); len(rawMsg) > 0 {
		_, err := resp.UnmarshalWithDecoder(coder.DefaultCoder, rawMsg)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// checkMyMessageID compare client msgID against peer messageID and if it is near < 0xffff/4 then incrase msgID.
// When msgIDs met it can cause issue because cache can send message to which doesn't bellows to request.
func (cc *Conn) checkMyMessageID(req *pool.Message) {
	if req.Type() == message.Confirmable {
		for {
			oldID := cc.msgID.Load()
			if uint16(req.MessageID())-uint16(cc.msgID.Load()) >= 0xffff/4 {
				return
			}
			newID := oldID + 0xffff/2
			if cc.msgID.CompareAndSwap(oldID, newID) {
				break
			}
		}
	}
}

func (cc *Conn) checkResponseCache(req *pool.Message, w *responsewriter.ResponseWriter[*Conn]) (bool, error) {
	if req.Type() == message.Confirmable || req.Type() == message.NonConfirmable {
		if ok, err := cc.getResponseFromCache(req.MessageID(), w.Message()); ok {
			w.Message().SetMessageID(req.MessageID())
			w.Message().SetType(message.NonConfirmable)
			if req.Type() == message.Confirmable {
				// req could be changed from NonConfirmation to confirmation message.
				w.Message().SetType(message.Acknowledgement)
			}
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("cannot unmarshal response from cache: %w", err)
		}
	}
	return false, nil
}

func isPongOrResetResponse(w *responsewriter.ResponseWriter[*Conn]) bool {
	return w.Message().IsModified() && (w.Message().Type() == message.Reset || w.Message().Code() == codes.Empty)
}

func sendJustAcknowledgeMessage(reqType message.Type, w *responsewriter.ResponseWriter[*Conn]) bool {
	return reqType == message.Confirmable && !w.Message().IsModified()
}

func (cc *Conn) processResponse(reqType message.Type, reqMessageID int32, w *responsewriter.ResponseWriter[*Conn]) error {
	switch {
	case isPongOrResetResponse(w):
		if reqType == message.Confirmable {
			w.Message().SetType(message.Acknowledgement)
			w.Message().SetMessageID(reqMessageID)
		} else {
			if w.Message().Type() != message.Reset {
				w.Message().SetType(message.NonConfirmable)
			}
			w.Message().SetMessageID(cc.GetMessageID())
		}
		return nil
	case sendJustAcknowledgeMessage(reqType, w):
		// send message to separate(confirm received) message, if response is not modified
		w.Message().SetCode(codes.Empty)
		w.Message().SetType(message.Acknowledgement)
		w.Message().SetMessageID(reqMessageID)
		w.Message().SetToken(nil)
		err := cc.addResponseToCache(w.Message())
		if err != nil {
			return fmt.Errorf("cannot cache response: %w", err)
		}
		return nil
	case !w.Message().IsModified():
		// don't send response
		return nil
	}

	// send piggybacked response
	w.Message().SetType(message.Confirmable)
	w.Message().SetMessageID(cc.GetMessageID())
	if reqType == message.Confirmable {
		w.Message().SetType(message.Acknowledgement)
		w.Message().SetMessageID(reqMessageID)
	}
	if reqType == message.Confirmable || reqType == message.NonConfirmable {
		err := cc.addResponseToCache(w.Message())
		if err != nil {
			return fmt.Errorf("cannot cache response: %w", err)
		}
	}
	return nil
}

func (cc *Conn) handleReq(w *responsewriter.ResponseWriter[*Conn], req *pool.Message) {
	defer cc.inactivityMonitor.Notify()
	reqMid := req.MessageID()

	// The same message ID can not be handled concurrently
	// for deduplication to work
	l := cc.msgIDMutex.Lock(reqMid)
	defer l.Unlock()

	if ok, err := cc.checkResponseCache(req, w); err != nil {
		cc.closeConnection()
		cc.errors(fmt.Errorf(errFmtWriteResponse, err))
		return
	} else if ok {
		return
	}

	w.Message().SetModified(false)
	reqType := req.Type()
	reqMessageID := req.MessageID()
	cc.handle(w, req)

	err := cc.processResponse(reqType, reqMessageID, w)
	if err != nil {
		cc.closeConnection()
		cc.errors(fmt.Errorf(errFmtWriteResponse, err))
	}
}

func (cc *Conn) closeConnection() {
	if errC := cc.Close(); errC != nil {
		cc.errors(fmt.Errorf("cannot close connection: %w", errC))
	}
}

func (cc *Conn) ProcessReceivedMessageWithHandler(req *pool.Message, handler config.HandlerFunc[*Conn]) {
	defer func() {
		if !req.IsHijacked() {
			cc.ReleaseMessage(req)
		}
	}()
	resp := cc.AcquireMessage(cc.Context())
	resp.SetToken(req.Token())
	w := responsewriter.New(resp, cc, req.Options()...)
	defer func() {
		cc.ReleaseMessage(w.Message())
	}()
	handler(w, req)
	select {
	case <-cc.Context().Done():
		return
	default:
	}
	if !w.Message().IsModified() {
		// nothing to send
		return
	}
	errW := cc.writeMessageAsync(w.Message())
	if errW != nil {
		cc.closeConnection()
		cc.errors(fmt.Errorf(errFmtWriteResponse, errW))
	}
}

func (cc *Conn) handlePong(w *responsewriter.ResponseWriter[*Conn], r *pool.Message) {
	cc.sendPong(w, r)
}

func (cc *Conn) handleSpecialMessages(r *pool.Message) bool {
	// ping request
	if r.Code() == codes.Empty && r.Type() == message.Confirmable && len(r.Token()) == 0 && len(r.Options()) == 0 && r.Body() == nil {
		cc.ProcessReceivedMessageWithHandler(r, cc.handlePong)
		return true
	}
	// if waits for concrete message handler
	if elem, ok := cc.midHandlerContainer.LoadAndDelete(r.MessageID()); ok {
		elem.ReleaseMessage(cc)
		resp := cc.AcquireMessage(cc.Context())
		resp.SetToken(r.Token())
		w := responsewriter.New(resp, cc, r.Options()...)
		defer func() {
			cc.ReleaseMessage(w.Message())
		}()
		elem.handler(w, r)
		// we just confirmed that message was processed for cc.writeMessage
		// the body of the message is need to be processed by the loopOverReceivedMessageQueue goroutine
		return false
	}
	// separate message
	if r.IsSeparateMessage() {
		// msg was processed by token handler - just drop it.
		return true
	}
	return false
}

func (cc *Conn) Process(datagram []byte) error {
	if uint32(len(datagram)) > cc.session.MaxMessageSize() {
		return fmt.Errorf("max message size(%v) was exceeded %v", cc.session.MaxMessageSize(), len(datagram))
	}
	req := cc.AcquireMessage(cc.Context())
	_, err := req.UnmarshalWithDecoder(coder.DefaultCoder, datagram)
	if err != nil {
		cc.ReleaseMessage(req)
		return err
	}
	req.SetSequence(cc.Sequence())
	cc.checkMyMessageID(req)
	cc.inactivityMonitor.Notify()
	if cc.handleSpecialMessages(req) {
		return nil
	}
	select {
	case cc.receivedMessageReader.C() <- req:
	case <-cc.Context().Done():
	}
	return nil
}

// SetContextValue stores the value associated with key to context of connection.
func (cc *Conn) SetContextValue(key interface{}, val interface{}) {
	cc.session.SetContextValue(key, val)
}

// Done signalizes that connection is not more processed.
func (cc *Conn) Done() <-chan struct{} {
	return cc.session.Done()
}

func (cc *Conn) checkMidHandlerContainer(now time.Time, maxRetransmit int32, acknowledgeTimeout time.Duration, key int32, value *midElement) {
	if value.IsExpired(now, maxRetransmit) {
		cc.midHandlerContainer.Delete(key)
		value.ReleaseMessage(cc)
		cc.errors(fmt.Errorf(errFmtWriteRequest, context.DeadlineExceeded))
		return
	}
	if !value.Retransmit(now, acknowledgeTimeout) {
		return
	}
	msg, ok, err := value.GetMessage(cc)
	if err != nil {
		cc.midHandlerContainer.Delete(key)
		value.ReleaseMessage(cc)
		cc.errors(fmt.Errorf(errFmtWriteRequest, err))
		return
	}
	if ok {
		defer cc.ReleaseMessage(msg)
		err := cc.session.WriteMessage(msg)
		if err != nil {
			cc.errors(fmt.Errorf(errFmtWriteRequest, err))
		}
	}
}

// CheckExpirations checks and remove expired items from caches.
func (cc *Conn) CheckExpirations(now time.Time) {
	cc.inactivityMonitor.CheckInactivity(now, cc)
	cc.responseMsgCache.CheckExpirations(now)
	if cc.blockWise != nil {
		cc.blockWise.CheckExpirations(now)
	}
	maxRetransmit := cc.transmission.maxRetransmit.Load()
	acknowledgeTimeout := cc.transmission.acknowledgeTimeout.Load()
	x := struct {
		now                time.Time
		maxRetransmit      int32
		acknowledgeTimeout time.Duration
		cc                 *Conn
	}{
		now:                now,
		maxRetransmit:      maxRetransmit,
		acknowledgeTimeout: acknowledgeTimeout,
		cc:                 cc,
	}
	cc.midHandlerContainer.Range(func(key int32, value *midElement) bool {
		x.cc.checkMidHandlerContainer(x.now, x.maxRetransmit, x.acknowledgeTimeout, key, value)
		return true
	})
}

func (cc *Conn) AcquireMessage(ctx context.Context) *pool.Message {
	return cc.messagePool.AcquireMessage(ctx)
}

func (cc *Conn) ReleaseMessage(m *pool.Message) {
	cc.messagePool.ReleaseMessage(m)
}

// WriteMulticastMessage sends multicast to the remote multicast address.
// By default it is sent over all network interfaces and all compatible source IP addresses with hop limit 1.
// Via opts you can specify the network interface, source IP address, and hop limit.
func (cc *Conn) WriteMulticastMessage(req *pool.Message, address *net.UDPAddr, options ...coapNet.MulticastOption) error {
	if req.Type() == message.Confirmable {
		return fmt.Errorf("multicast messages cannot be confirmable")
	}
	req.UpsertMessageID(cc.GetMessageID())

	err := cc.session.WriteMulticastMessage(req, address, options...)
	if err != nil {
		return fmt.Errorf(errFmtWriteRequest, err)
	}
	return nil
}

func (cc *Conn) InactivityMonitor() InactivityMonitor {
	return cc.inactivityMonitor
}

// NetConn returns the underlying connection that is wrapped by cc. The Conn returned is shared by all invocations of NetConn, so do not modify it.
func (cc *Conn) NetConn() net.Conn {
	return cc.session.NetConn()
}
