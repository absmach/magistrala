package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/udp/client"
	"github.com/plgd-dev/go-coap/v3/udp/coder"
)

type EventFunc = func()

type Session struct {
	onClose []EventFunc

	ctx atomic.Value // TODO: change to atomic.Pointer[context.Context] for go1.19

	doneCtx    context.Context
	connection *coapNet.UDPConn
	doneCancel context.CancelFunc

	cancel context.CancelFunc
	raddr  *net.UDPAddr

	mutex          sync.Mutex
	maxMessageSize uint32
	mtu            uint16

	closeSocket bool
}

func NewSession(
	ctx context.Context,
	doneCtx context.Context,
	connection *coapNet.UDPConn,
	raddr *net.UDPAddr,
	maxMessageSize uint32,
	mtu uint16,
	closeSocket bool,
) *Session {
	ctx, cancel := context.WithCancel(ctx)

	doneCtx, doneCancel := context.WithCancel(doneCtx)
	s := &Session{
		cancel:         cancel,
		connection:     connection,
		raddr:          raddr,
		maxMessageSize: maxMessageSize,
		mtu:            mtu,
		closeSocket:    closeSocket,
		doneCtx:        doneCtx,
		doneCancel:     doneCancel,
	}
	s.ctx.Store(&ctx)
	return s
}

// SetContextValue stores the value associated with key to context of connection.
func (s *Session) SetContextValue(key interface{}, val interface{}) {
	ctx := context.WithValue(s.Context(), key, val)
	s.ctx.Store(&ctx)
}

// Done signalizes that connection is not more processed.
func (s *Session) Done() <-chan struct{} {
	return s.doneCtx.Done()
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
	defer s.doneCancel()
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
	return *s.ctx.Load().(*context.Context) //nolint:forcetypeassert
}

func (s *Session) WriteMessage(req *pool.Message) error {
	data, err := req.MarshalWithEncoder(coder.DefaultCoder)
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}
	return s.connection.WriteWithContext(req.Context(), s.raddr, data)
}

// WriteMulticastMessage sends multicast to the remote multicast address.
// By default it is sent over all network interfaces and all compatible source IP addresses with hop limit 1.
// Via opts you can specify the network interface, source IP address, and hop limit.
func (s *Session) WriteMulticastMessage(req *pool.Message, address *net.UDPAddr, opts ...coapNet.MulticastOption) error {
	data, err := req.MarshalWithEncoder(coder.DefaultCoder)
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}

	return s.connection.WriteMulticast(req.Context(), address, data, opts...)
}

func (s *Session) Run(cc *client.Conn) (err error) {
	defer func() {
		err1 := s.Close()
		if err == nil {
			err = err1
		}
		s.shutdown()
	}()
	m := make([]byte, s.mtu)
	for {
		buf := m
		n, _, err := s.connection.ReadWithContext(s.Context(), buf)
		if err != nil {
			return err
		}
		buf = buf[:n]
		err = cc.Process(buf)
		if err != nil {
			return err
		}
	}
}

func (s *Session) MaxMessageSize() uint32 {
	return s.maxMessageSize
}

func (s *Session) RemoteAddr() net.Addr {
	return s.raddr
}

func (s *Session) LocalAddr() net.Addr {
	return s.connection.LocalAddr()
}

// NetConn returns the underlying connection that is wrapped by s. The Conn returned is shared by all invocations of NetConn, so do not modify it.
func (s *Session) NetConn() net.Conn {
	return s.connection.NetConn()
}
