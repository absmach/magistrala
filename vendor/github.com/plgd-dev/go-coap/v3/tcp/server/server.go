package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v3/pkg/connections"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

// A Option sets options such as credentials, codec and keepalive parameters, etc.
type Option interface {
	TCPServerApply(cfg *Config)
}

// Listener defined used by coap
type Listener interface {
	Close() error
	AcceptWithContext(ctx context.Context) (net.Conn, error)
}

type Server struct {
	listenMutex sync.Mutex
	listen      Listener
	ctx         context.Context
	cancel      context.CancelFunc
	cfg         *Config
}

func New(opt ...Option) *Server {
	cfg := DefaultConfig
	for _, o := range opt {
		o.TCPServerApply(&cfg)
	}

	ctx, cancel := context.WithCancel(cfg.Ctx)

	if cfg.CreateInactivityMonitor == nil {
		cfg.CreateInactivityMonitor = func() client.InactivityMonitor {
			return inactivity.NewNilMonitor[*client.Conn]()
		}
	}
	if cfg.MessagePool == nil {
		cfg.MessagePool = pool.New(0, 0)
	}

	if cfg.Errors == nil {
		cfg.Errors = func(error) {
			// default no-op
		}
	}
	if cfg.GetToken == nil {
		cfg.GetToken = message.GetToken
	}
	errorsFunc := cfg.Errors
	// assign updated func to opts.errors so opts.handler also uses the updated error handler
	cfg.Errors = func(err error) {
		if coapNet.IsCancelOrCloseError(err) {
			// this error was produced by cancellation context or closing connection.
			return
		}
		errorsFunc(fmt.Errorf("tcp: %w", err))
	}

	return &Server{
		ctx:    ctx,
		cancel: cancel,
		cfg:    &cfg,
	}
}

func (s *Server) checkAndSetListener(l Listener) error {
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	if s.listen != nil {
		return fmt.Errorf("server already serves listener")
	}
	s.listen = l
	return nil
}

func (s *Server) popListener() Listener {
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	l := s.listen
	s.listen = nil
	return l
}

func (s *Server) checkAcceptError(err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, coapNet.ErrListenerIsClosed):
		s.Stop()
		return false
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		select {
		case <-s.ctx.Done():
		default:
			s.cfg.Errors(fmt.Errorf("cannot accept connection: %w", err))
			return true
		}
		return false
	default:
		return true
	}
}

func (s *Server) serveConnection(connections *connections.Connections, rw net.Conn) {
	var cc *client.Conn
	monitor := s.cfg.CreateInactivityMonitor()
	cc = s.createConn(coapNet.NewConn(rw), monitor)
	if s.cfg.OnNewConn != nil {
		s.cfg.OnNewConn(cc)
	}
	connections.Store(cc)
	defer connections.Delete(cc)

	if err := cc.Run(); err != nil {
		s.cfg.Errors(fmt.Errorf("%v: %w", cc.RemoteAddr(), err))
	}
}

func (s *Server) Serve(l Listener) error {
	if s.cfg.BlockwiseSZX > blockwise.SZXBERT {
		return fmt.Errorf("invalid blockwiseSZX")
	}

	err := s.checkAndSetListener(l)
	if err != nil {
		return err
	}
	defer func() {
		s.Stop()
	}()
	var wg sync.WaitGroup
	defer wg.Wait()

	connections := connections.New()
	s.cfg.PeriodicRunner(func(now time.Time) bool {
		connections.CheckExpirations(now)
		return s.ctx.Err() == nil
	})
	defer connections.Close()

	for {
		rw, err := l.AcceptWithContext(s.ctx)
		if ok := s.checkAcceptError(err); !ok {
			return nil
		}
		if rw == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.serveConnection(connections, rw)
		}()
	}
}

// Stop stops server without wait of ends Serve function.
func (s *Server) Stop() {
	s.cancel()
	l := s.popListener()
	if l == nil {
		return
	}
	if err := l.Close(); err != nil {
		s.cfg.Errors(fmt.Errorf("cannot close listener: %w", err))
	}
}

func (s *Server) createConn(connection *coapNet.Conn, monitor client.InactivityMonitor) *client.Conn {
	createBlockWise := func(cc *client.Conn) *blockwise.BlockWise[*client.Conn] {
		return nil
	}
	if s.cfg.BlockwiseEnable {
		createBlockWise = func(cc *client.Conn) *blockwise.BlockWise[*client.Conn] {
			return blockwise.New(
				cc,
				s.cfg.BlockwiseTransferTimeout,
				s.cfg.Errors,
				func(token message.Token) (*pool.Message, bool) {
					return nil, false
				},
			)
		}
	}
	cfg := client.DefaultConfig
	cfg.Ctx = s.ctx
	cfg.Handler = s.cfg.Handler
	cfg.MaxMessageSize = s.cfg.MaxMessageSize
	cfg.Errors = s.cfg.Errors
	cfg.BlockwiseSZX = s.cfg.BlockwiseSZX
	cfg.DisablePeerTCPSignalMessageCSMs = s.cfg.DisablePeerTCPSignalMessageCSMs
	cfg.DisableTCPSignalMessageCSM = s.cfg.DisableTCPSignalMessageCSM
	cfg.CloseSocket = true
	cfg.ConnectionCacheSize = s.cfg.ConnectionCacheSize
	cfg.MessagePool = s.cfg.MessagePool
	cfg.GetToken = s.cfg.GetToken
	cfg.ProcessReceivedMessage = s.cfg.ProcessReceivedMessage
	cfg.ReceivedMessageQueueSize = s.cfg.ReceivedMessageQueueSize
	cc := client.NewConn(
		connection,
		createBlockWise,
		monitor,
		&cfg,
	)

	return cc
}
