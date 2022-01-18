package tcp

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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
	sequence   uint64
	connection *coapNet.Conn

	maxMessageSize                  int
	peerMaxMessageSize              uint32
	peerBlockWiseTranferEnabled     uint32
	disablePeerTCPSignalMessageCSMs bool
	disableTCPSignalMessageCSM      bool
	goPool                          GoPoolFunc
	errors                          ErrorFunc
	closeSocket                     bool
	inactivityMonitor               Notifier

	tokenHandlerContainer *HandlerContainer
	midHandlerContainer   *HandlerContainer
	handler               HandlerFunc

	blockwiseSZX blockwise.SZX
	blockWise    *blockwise.BlockWise

	mutex   sync.Mutex
	onClose []EventFunc

	cancel context.CancelFunc
	ctx    atomic.Value

	errSendCSM error
}

func NewSession(
	ctx context.Context,
	connection *coapNet.Conn,
	handler HandlerFunc,
	maxMessageSize int,
	goPool GoPoolFunc,
	errors ErrorFunc,
	blockwiseSZX blockwise.SZX,
	blockWise *blockwise.BlockWise,
	disablePeerTCPSignalMessageCSMs bool,
	disableTCPSignalMessageCSM bool,
	closeSocket bool,
	inactivityMonitor Notifier,
) *Session {
	ctx, cancel := context.WithCancel(ctx)
	if errors == nil {
		errors = func(error) {}
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

func (s *Session) Done() <-chan struct{} {
	return s.Context().Done()
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

func (s *Session) close() error {
	for _, f := range s.popOnClose() {
		f()
	}
	if s.closeSocket {
		return s.connection.Close()
	}
	return nil
}

func (s *Session) Close() error {
	s.cancel()
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
			w := bw.(*bwResponseWriter).w
			r := br.(*pool.Message)
			if err == nil {
				h(w, r)
				return
			}
			s.handler(w, r)
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
		if r.HasOption(coapTCP.Custody) {
			//TODO
		}
		s.sendPong(r.Token())
		return true
	case codes.Release:
		if r.HasOption(coapTCP.AlternativeAddress) {
			//TODO
		}
		return true
	case codes.Abort:
		if r.HasOption(coapTCP.BadCSMOption) {
			//TODO
		}
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
	pool.ReleaseMessage(b.w.response)
	b.w.response = m.(*pool.Message)
}

func (s *Session) Handle(w *ResponseWriter, r *pool.Message) {
	s.handleBlockwise(w, r)
}

func (s *Session) TokenHandler() *HandlerContainer {
	return s.tokenHandlerContainer
}

func (s *Session) processReq(req *pool.Message, cc *ClientConn, handler func(w *ResponseWriter, r *pool.Message)) {
	origResp := pool.AcquireMessage(s.Context())
	origResp.SetToken(req.Token())
	w := NewResponseWriter(origResp, cc, req.Options())
	handler(w, req)
	defer pool.ReleaseMessage(w.response)
	if !req.IsHijacked() {
		pool.ReleaseMessage(req)
	}
	if w.response.IsModified() {
		err := s.WriteMessage(w.response)
		if err != nil {
			s.Close()
			s.errors(fmt.Errorf("cannot write response to %v: %w", s.connection.RemoteAddr(), err))
		}
	}
}

func (s *Session) processBuffer(buffer *bytes.Buffer, cc *ClientConn) error {
	for buffer.Len() > 0 {
		var hdr coapTCP.MessageHeader
		err := hdr.Unmarshal(buffer.Bytes())
		if err == message.ErrShortRead {
			return nil
		}
		if s.maxMessageSize >= 0 && hdr.TotalLen > s.maxMessageSize {
			return fmt.Errorf("max message size(%v) was exceeded %v", s.maxMessageSize, hdr.TotalLen)
		}
		if buffer.Len() < hdr.TotalLen {
			return nil
		}
		req := pool.AcquireMessage(s.Context())
		readed, err := req.Unmarshal(buffer.Bytes()[:hdr.TotalLen])
		if err != nil {
			pool.ReleaseMessage(req)
			return fmt.Errorf("cannot unmarshal with header: %w", err)
		}
		if readed == buffer.Len() {
			// buffer is empty so reset it
			buffer.Reset()
		} else {
			// rewind to next message
			trimmed := 0
			for trimmed != readed {
				b := make([]byte, 4096)
				max := 4096
				if readed-trimmed < max {
					max = readed - trimmed
				}
				v, _ := buffer.Read(b[:max])
				trimmed += v
			}
		}
		req.SetSequence(s.Sequence())
		s.inactivityMonitor.Notify()
		if s.handleSignals(req, cc) {
			continue
		}
		s.goPool(func() {
			s.processReq(req, cc, s.Handle)
		})
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
	req := pool.AcquireMessage(s.Context())
	defer pool.ReleaseMessage(req)
	req.SetCode(codes.CSM)
	req.SetToken(token)
	return s.WriteMessage(req)
}

func (s *Session) sendPong(token message.Token) error {
	req := pool.AcquireMessage(s.Context())
	defer pool.ReleaseMessage(req)
	req.SetCode(codes.Pong)
	req.SetToken(token)
	return s.WriteMessage(req)
}

// Run reads and process requests from a connection, until the connection is not closed.
func (s *Session) Run(cc *ClientConn) (err error) {
	defer func() {
		err1 := s.Close()
		if err == nil {
			err = err1
		}
		err1 = s.close()
		if err == nil {
			err = err1
		}
	}()
	if s.errSendCSM != nil {
		return s.errSendCSM
	}
	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	readBuf := make([]byte, 1024)
	for {
		err = s.processBuffer(buffer, cc)
		if err != nil {
			return err
		}
		readLen, err := s.connection.ReadWithContext(s.Context(), readBuf)
		if err != nil {
			return fmt.Errorf("cannot read from connection: %w", err)
		}
		if readLen > 0 {
			buffer.Write(readBuf[:readLen])
		}
	}
}
