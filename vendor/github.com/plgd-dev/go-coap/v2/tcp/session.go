package tcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapNet "github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	coapTCP "github.com/plgd-dev/go-coap/v2/tcp/message"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

type EventFunc func()

type Session struct {
	// This field needs to be the first in the struct to ensure proper word alignment on 32-bit platforms.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	sequence uint64

	onClose []EventFunc

	ctx atomic.Value

	inactivityMonitor inactivity.Monitor

	errSendCSM error

	cancel context.CancelFunc

	done chan struct{}

	goPool    GoPoolFunc
	errors    ErrorFunc
	blockWise *blockwise.BlockWise

	connection *coapNet.Conn

	handler HandlerFunc

	midHandlerContainer *HandlerContainer

	tokenHandlerContainer *HandlerContainer

	messagePool *pool.Pool

	mutex sync.Mutex

	maxMessageSize                  uint32
	peerBlockWiseTranferEnabled     uint32
	peerMaxMessageSize              uint32
	connectionCacheSize             uint16
	disableTCPSignalMessageCSM      bool
	disablePeerTCPSignalMessageCSMs bool

	blockwiseSZX blockwise.SZX
	closeSocket  bool
}

func NewSession(
	ctx context.Context,
	connection *coapNet.Conn,
	handler HandlerFunc,
	maxMessageSize uint32,
	goPool GoPoolFunc,
	errors ErrorFunc,
	blockwiseSZX blockwise.SZX,
	blockWise *blockwise.BlockWise,
	disablePeerTCPSignalMessageCSMs bool,
	disableTCPSignalMessageCSM bool,
	closeSocket bool,
	inactivityMonitor inactivity.Monitor,
	connectionCacheSize uint16,
	messagePool *pool.Pool,
) *Session {
	ctx, cancel := context.WithCancel(ctx)
	if errors == nil {
		errors = func(error) {
			// default no-op
		}
	}
	if inactivityMonitor == nil {
		inactivityMonitor = inactivity.NewNilMonitor()
	}

	s := &Session{
		cancel:                          cancel,
		connection:                      connection,
		handler:                         handler,
		maxMessageSize:                  maxMessageSize,
		tokenHandlerContainer:           NewHandlerContainer(),
		midHandlerContainer:             NewHandlerContainer(),
		goPool:                          goPool,
		errors:                          errors,
		blockWise:                       blockWise,
		blockwiseSZX:                    blockwiseSZX,
		disablePeerTCPSignalMessageCSMs: disablePeerTCPSignalMessageCSMs,
		disableTCPSignalMessageCSM:      disableTCPSignalMessageCSM,
		closeSocket:                     closeSocket,
		inactivityMonitor:               inactivityMonitor,
		done:                            make(chan struct{}),
		connectionCacheSize:             connectionCacheSize,
		messagePool:                     messagePool,
	}
	s.ctx.Store(&ctx)

	if !disableTCPSignalMessageCSM {
		err := s.sendCSM()
		if err != nil {
			s.errSendCSM = fmt.Errorf("cannot send CSM: %w", err)
		}
	}

	return s
}

// SetContextValue stores the value associated with key to context of connection.
func (s *Session) SetContextValue(key interface{}, val interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	ctx := context.WithValue(s.Context(), key, val)
	s.ctx.Store(&ctx)
}

// Done signalizes that connection is not more processed.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) AddOnClose(f EventFunc) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onClose = append(s.onClose, f)
}

func (s *Session) popOnClose() []EventFunc {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tmp := s.onClose
	s.onClose = nil
	return tmp
}

func (s *Session) shutdown() {
	defer close(s.done)
	for _, f := range s.popOnClose() {
		f()
	}
}

func (s *Session) Close() error {
	s.cancel()
	if s.closeSocket {
		return s.connection.Close()
	}
	return nil
}

func (s *Session) Sequence() uint64 {
	return atomic.AddUint64(&s.sequence, 1)
}

func (s *Session) Context() context.Context {
	return *s.ctx.Load().(*context.Context)
}

func (s *Session) PeerMaxMessageSize() uint32 {
	return atomic.LoadUint32(&s.peerMaxMessageSize)
}

func (s *Session) PeerBlockWiseTransferEnabled() bool {
	return atomic.LoadUint32(&s.peerBlockWiseTranferEnabled) == 1
}

func (s *Session) handleBlockwise(w *ResponseWriter, r *pool.Message) {
	if s.blockWise != nil && s.PeerBlockWiseTransferEnabled() {
		bwr := bwResponseWriter{
			w: w,
		}
		s.blockWise.Handle(&bwr, r, s.blockwiseSZX, s.maxMessageSize, func(bw blockwise.ResponseWriter, br blockwise.Message) {
			h, err := s.tokenHandlerContainer.Pop(r.Token())
			rw := bw.(*bwResponseWriter).w
			m := br.(*pool.Message)
			if err == nil {
				h(rw, m)
				return
			}
			s.handler(rw, m)
		})
		return
	}
	h, err := s.tokenHandlerContainer.Pop(r.Token())
	if err == nil {
		h(w, r)
		return
	}
	s.handler(w, r)
}

func (s *Session) handleSignals(r *pool.Message, cc *ClientConn) bool {
	switch r.Code() {
	case codes.CSM:
		if s.disablePeerTCPSignalMessageCSMs {
			return true
		}
		if size, err := r.GetOptionUint32(coapTCP.MaxMessageSize); err == nil {
			atomic.StoreUint32(&s.peerMaxMessageSize, size)
		}
		if r.HasOption(coapTCP.BlockWiseTransfer) {
			atomic.StoreUint32(&s.peerBlockWiseTranferEnabled, 1)
		}
		return true
	case codes.Ping:
		// if r.HasOption(coapTCP.Custody) {
		//TODO
		// }
		if err := s.sendPong(r.Token()); err != nil && !coapNet.IsConnectionBrokenError(err) {
			s.errors(fmt.Errorf("cannot handle ping signal: %w", err))
		}
		return true
	case codes.Release:
		// if r.HasOption(coapTCP.AlternativeAddress) {
		//TODO
		// }
		return true
	case codes.Abort:
		// if r.HasOption(coapTCP.BadCSMOption) {
		//TODO
		// }
		return true
	case codes.Pong:
		h, err := s.tokenHandlerContainer.Pop(r.Token())
		if err == nil {
			s.processReq(r, cc, h)
		}
		return true
	}
	return false
}

type bwResponseWriter struct {
	w *ResponseWriter
}

func (b *bwResponseWriter) Message() blockwise.Message {
	return b.w.response
}

func (b *bwResponseWriter) SetMessage(m blockwise.Message) {
	b.w.cc.session.messagePool.ReleaseMessage(b.w.response)
	b.w.response = m.(*pool.Message)
}

func (b *bwResponseWriter) RemoteAddr() net.Addr {
	return b.w.cc.RemoteAddr()
}

func (s *Session) Handle(w *ResponseWriter, r *pool.Message) {
	s.handleBlockwise(w, r)
}

func (s *Session) TokenHandler() *HandlerContainer {
	return s.tokenHandlerContainer
}

func (s *Session) processReq(req *pool.Message, cc *ClientConn, handler func(w *ResponseWriter, r *pool.Message)) {
	origResp := s.messagePool.AcquireMessage(s.Context())
	origResp.SetToken(req.Token())
	w := NewResponseWriter(origResp, cc, req.Options())
	handler(w, req)
	defer s.messagePool.ReleaseMessage(w.response)
	if !req.IsHijacked() {
		s.messagePool.ReleaseMessage(req)
	}
	if w.response.IsModified() {
		err := s.WriteMessage(w.response)
		if err != nil {
			if errC := s.Close(); errC != nil {
				s.errors(fmt.Errorf("cannot close connection: %w", errC))
			}
			s.errors(fmt.Errorf("cannot write response to %v: %w", s.connection.RemoteAddr(), err))
		}
	}
}

func seekBufferToNextMessage(buffer *bytes.Buffer, msgSize int) *bytes.Buffer {
	if msgSize == buffer.Len() {
		// buffer is empty so reset it
		buffer.Reset()
		return buffer
	}
	// rewind to next message
	trimmed := 0
	for trimmed != msgSize {
		b := make([]byte, 4096)
		max := 4096
		if msgSize-trimmed < max {
			max = msgSize - trimmed
		}
		v, _ := buffer.Read(b[:max])
		trimmed += v
	}
	return buffer
}

func (s *Session) processBuffer(buffer *bytes.Buffer, cc *ClientConn) error {
	for buffer.Len() > 0 {
		var hdr coapTCP.MessageHeader
		err := hdr.Unmarshal(buffer.Bytes())
		if errors.Is(err, message.ErrShortRead) {
			return nil
		}
		if hdr.TotalLen > s.maxMessageSize {
			return fmt.Errorf("max message size(%v) was exceeded %v", s.maxMessageSize, hdr.TotalLen)
		}
		if uint32(buffer.Len()) < hdr.TotalLen {
			return nil
		}
		req := s.messagePool.AcquireMessage(s.Context())
		read, err := req.Unmarshal(buffer.Bytes()[:hdr.TotalLen])
		if err != nil {
			s.messagePool.ReleaseMessage(req)
			return fmt.Errorf("cannot unmarshal with header: %w", err)
		}
		buffer = seekBufferToNextMessage(buffer, read)
		req.SetSequence(s.Sequence())
		s.inactivityMonitor.Notify()
		if s.handleSignals(req, cc) {
			continue
		}
		err = s.goPool(func() {
			s.processReq(req, cc, s.Handle)
		})
		if err != nil {
			return fmt.Errorf("cannot spawn go routine: %w", err)
		}
	}
	return nil
}

func (s *Session) WriteMessage(req *pool.Message) error {
	data, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}
	err = s.connection.WriteWithContext(req.Context(), data)
	if err != nil {
		return fmt.Errorf("cannot write to connection: %w", err)
	}
	return err
}

func (s *Session) sendCSM() error {
	token, err := message.GetToken()
	if err != nil {
		return fmt.Errorf("cannot get token: %w", err)
	}
	req := s.messagePool.AcquireMessage(s.Context())
	defer s.messagePool.ReleaseMessage(req)
	req.SetCode(codes.CSM)
	req.SetToken(token)
	return s.WriteMessage(req)
}

func (s *Session) sendPong(token message.Token) error {
	req := s.messagePool.AcquireMessage(s.Context())
	defer s.messagePool.ReleaseMessage(req)
	req.SetCode(codes.Pong)
	req.SetToken(token)
	return s.WriteMessage(req)
}

func shrinkBufferIfNecessary(buffer *bytes.Buffer, maxCap uint16) *bytes.Buffer {
	if buffer.Len() == 0 && buffer.Cap() > int(maxCap) {
		buffer = bytes.NewBuffer(make([]byte, 0, maxCap))
	}
	return buffer
}

// Run reads and process requests from a connection, until the connection is not closed.
func (s *Session) Run(cc *ClientConn) (err error) {
	defer func() {
		err1 := s.Close()
		if err == nil {
			err = err1
		}
		s.shutdown()
	}()
	if s.errSendCSM != nil {
		return s.errSendCSM
	}
	buffer := bytes.NewBuffer(make([]byte, 0, s.connectionCacheSize))
	readBuf := make([]byte, s.connectionCacheSize)
	for {
		err = s.processBuffer(buffer, cc)
		if err != nil {
			return err
		}
		buffer = shrinkBufferIfNecessary(buffer, s.connectionCacheSize)
		readLen, err := s.connection.ReadWithContext(s.Context(), readBuf)
		if err != nil {
			if coapNet.IsConnectionBrokenError(err) { // other side closed the connection, ignore the error and return
				return nil
			}
			return fmt.Errorf("cannot read from connection: %w", err)
		}
		if readLen > 0 {
			buffer.Write(readBuf[:readLen])
		}
	}
}

// CheckExpirations checks and remove expired items from caches.
func (s *Session) CheckExpirations(now time.Time, cc *ClientConn) {
	s.inactivityMonitor.CheckInactivity(now, cc)
	if s.blockWise != nil {
		s.blockWise.CheckExpirations(now)
	}
}
