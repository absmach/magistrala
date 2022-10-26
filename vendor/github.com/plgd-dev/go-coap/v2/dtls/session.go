package dtls

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	coapNet "github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/udp/client"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

type EventFunc = func()

type Session struct {
	onClose []EventFunc

	ctx atomic.Value

	cancel     context.CancelFunc
	connection *coapNet.Conn

	done chan struct{}

	mutex sync.Mutex

	maxMessageSize uint32

	closeSocket bool
}

func NewSession(
	ctx context.Context,
	connection *coapNet.Conn,
	maxMessageSize uint32,
	closeSocket bool,
) *Session {
	ctx, cancel := context.WithCancel(ctx)
	s := &Session{
		cancel:         cancel,
		connection:     connection,
		maxMessageSize: maxMessageSize,
		closeSocket:    closeSocket,
		done:           make(chan struct{}),
	}
	s.ctx.Store(&ctx)
	return s
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

func (s *Session) Context() context.Context {
	return *s.ctx.Load().(*context.Context)
}

// SetContextValue stores the value associated with key to context of connection.
func (s *Session) SetContextValue(key interface{}, val interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	ctx := context.WithValue(s.Context(), key, val)
	s.ctx.Store(&ctx)
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

// WriteMulticastMessage sends multicast to the remote multicast address.
// Currently it is not implemented - is is just satisfy golang udp/client/Session interface.
func (s *Session) WriteMulticastMessage(req *pool.Message, address *net.UDPAddr, opts ...coapNet.MulticastOption) error {
	return errors.New("multicast messages not implemented for DTLS")
}

func (s *Session) MaxMessageSize() uint32 {
	return s.maxMessageSize
}

func (s *Session) RemoteAddr() net.Addr {
	return s.connection.RemoteAddr()
}

func (s *Session) LocalAddr() net.Addr {
	return s.connection.LocalAddr()
}

// Run reads and process requests from a connection, until the connection is not closed.
func (s *Session) Run(cc *client.ClientConn) (err error) {
	defer func() {
		err1 := s.Close()
		if err == nil {
			err = err1
		}
		s.shutdown()
	}()
	m := make([]byte, s.maxMessageSize)
	for {
		readBuf := m
		readLen, err := s.connection.ReadWithContext(s.Context(), readBuf)
		if err != nil {
			return fmt.Errorf("cannot read from connection: %w", err)
		}
		readBuf = readBuf[:readLen]
		err = cc.Process(readBuf)
		if err != nil {
			return err
		}
	}
}
