package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v3/tcp/coder"
	"go.uber.org/atomic"
)

type Session struct {
	// This field needs to be the first in the struct to ensure proper word alignment on 32-bit platforms.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	sequence          atomic.Uint64
	inactivityMonitor InactivityMonitor
	errSendCSM        error
	cancel            context.CancelFunc
	done              chan struct{}
	errors            ErrorFunc
	connection        *coapNet.Conn
	messagePool       *pool.Pool
	ctx               atomic.Value // TODO: change to atomic.Pointer[context.Context] for go1.19
	maxMessageSize    uint32
	private           struct {
		mutex   sync.Mutex
		onClose []EventFunc
	}
	connectionCacheSize        uint16
	disableTCPSignalMessageCSM bool
	closeSocket                bool
}

func NewSession(
	ctx context.Context,
	connection *coapNet.Conn,
	maxMessageSize uint32,
	errors ErrorFunc,
	disableTCPSignalMessageCSM bool,
	closeSocket bool,
	inactivityMonitor InactivityMonitor,
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
		inactivityMonitor = inactivity.NewNilMonitor[*Conn]()
	}

	s := &Session{
		cancel:                     cancel,
		connection:                 connection,
		maxMessageSize:             maxMessageSize,
		errors:                     errors,
		disableTCPSignalMessageCSM: disableTCPSignalMessageCSM,
		closeSocket:                closeSocket,
		inactivityMonitor:          inactivityMonitor,
		done:                       make(chan struct{}),
		connectionCacheSize:        connectionCacheSize,
		messagePool:                messagePool,
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
	ctx := context.WithValue(s.Context(), key, val)
	s.ctx.Store(&ctx)
}

// Done signalizes that connection is not more processed.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) AddOnClose(f EventFunc) {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()
	s.private.onClose = append(s.private.onClose, f)
}

func (s *Session) popOnClose() []EventFunc {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()
	tmp := s.private.onClose
	s.private.onClose = nil
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
	return s.sequence.Inc()
}

func (s *Session) Context() context.Context {
	return *s.ctx.Load().(*context.Context) //nolint:forcetypeassert
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

func (s *Session) processBuffer(buffer *bytes.Buffer, cc *Conn) error {
	for buffer.Len() > 0 {
		var header coder.MessageHeader
		_, err := coder.DefaultCoder.DecodeHeader(buffer.Bytes(), &header)
		if errors.Is(err, message.ErrShortRead) {
			return nil
		}
		if header.MessageLength > s.maxMessageSize {
			return fmt.Errorf("max message size(%v) was exceeded %v", s.maxMessageSize, header.MessageLength)
		}
		if uint32(buffer.Len()) < header.MessageLength {
			return nil
		}
		req := s.messagePool.AcquireMessage(s.Context())
		read, err := req.UnmarshalWithDecoder(coder.DefaultCoder, buffer.Bytes()[:header.MessageLength])
		if err != nil {
			s.messagePool.ReleaseMessage(req)
			return fmt.Errorf("cannot unmarshal with header: %w", err)
		}
		buffer = seekBufferToNextMessage(buffer, read)
		req.SetSequence(s.Sequence())
		s.inactivityMonitor.Notify()
		cc.pushToReceivedMessageQueue(req)
	}
	return nil
}

func (s *Session) WriteMessage(req *pool.Message) error {
	data, err := req.MarshalWithEncoder(coder.DefaultCoder)
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

func shrinkBufferIfNecessary(buffer *bytes.Buffer, maxCap uint16) *bytes.Buffer {
	if buffer.Len() == 0 && buffer.Cap() > int(maxCap) {
		buffer = bytes.NewBuffer(make([]byte, 0, maxCap))
	}
	return buffer
}

// Run reads and process requests from a connection, until the connection is not closed.
func (s *Session) Run(cc *Conn) (err error) {
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
func (s *Session) CheckExpirations(now time.Time, cc *Conn) {
	s.inactivityMonitor.CheckInactivity(now, cc)
}

func (s *Session) AcquireMessage(ctx context.Context) *pool.Message {
	return s.messagePool.AcquireMessage(ctx)
}

func (s *Session) ReleaseMessage(m *pool.Message) {
	s.messagePool.ReleaseMessage(m)
}

// RemoteAddr gets remote address.
func (s *Session) RemoteAddr() net.Addr {
	return s.connection.RemoteAddr()
}

func (s *Session) LocalAddr() net.Addr {
	return s.connection.LocalAddr()
}

// NetConn returns the underlying connection that is wrapped by s. The Conn returned is shared by all invocations of NetConn, so do not modify it.
func (s *Session) NetConn() net.Conn {
	return s.connection.NetConn()
}
