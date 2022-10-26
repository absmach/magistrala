package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapNet "github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/pkg/connections"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	kitSync "github.com/plgd-dev/kit/v2/sync"
)

// A ServerOption sets options such as credentials, codec and keepalive parameters, etc.
type ServerOption interface {
	apply(*serverOptions)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as COAP handlers.
type HandlerFunc = func(*ResponseWriter, *pool.Message)

type ErrorFunc = func(error)

type GoPoolFunc = func(func()) error

type BlockwiseFactoryFunc = func(getSentRequest func(token message.Token) (blockwise.Message, bool)) *blockwise.BlockWise

// OnNewClientConnFunc is the callback for new connections.
//
// Note: Calling `tlscon.Close()` is forbidden, and `tlscon` should be treated as a
// "read-only" parameter, mainly used to get the peer certificate from the underlining connection
type OnNewClientConnFunc = func(cc *ClientConn, tlscon *tls.Conn)

var defaultServerOptions = func() serverOptions {
	opts := serverOptions{
		ctx:            context.Background(),
		maxMessageSize: 64 * 1024,
		errors: func(err error) {
			fmt.Println(err)
		},
		goPool: func(f func()) error {
			go func() {
				f()
			}()
			return nil
		},
		blockwiseEnable:          true,
		blockwiseSZX:             blockwise.SZX1024,
		blockwiseTransferTimeout: time.Second * 3,
		onNewClientConn:          func(cc *ClientConn, tlscon *tls.Conn) {},
		createInactivityMonitor: func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		},
		periodicRunner: func(f func(now time.Time) bool) {
			go func() {
				for f(time.Now()) {
					time.Sleep(4 * time.Second)
				}
			}()
		},
		connectionCacheSize: 2 * 1024,
		messagePool:         pool.New(1024, 2048),
	}
	opts.handler = func(w *ResponseWriter, r *pool.Message) {
		if err := w.SetResponse(codes.NotFound, message.TextPlain, nil); err != nil {
			opts.errors(fmt.Errorf("server handler: cannot set response: %w", err))
		}
	}
	return opts
}()

type serverOptions struct {
	ctx                             context.Context
	messagePool                     *pool.Pool
	handler                         HandlerFunc
	errors                          ErrorFunc
	goPool                          GoPoolFunc
	createInactivityMonitor         func() inactivity.Monitor
	periodicRunner                  periodic.Func
	onNewClientConn                 OnNewClientConnFunc
	blockwiseTransferTimeout        time.Duration
	maxMessageSize                  uint32
	connectionCacheSize             uint16
	disablePeerTCPSignalMessageCSMs bool
	disableTCPSignalMessageCSM      bool
	blockwiseSZX                    blockwise.SZX
	blockwiseEnable                 bool
}

// Listener defined used by coap
type Listener interface {
	Close() error
	AcceptWithContext(ctx context.Context) (net.Conn, error)
}

type Server struct {
	listen Listener
	ctx    context.Context

	cancel context.CancelFunc

	messagePool *pool.Pool

	errors ErrorFunc
	goPool GoPoolFunc

	blockwiseTransferTimeout time.Duration
	onNewClientConn          OnNewClientConnFunc

	handler                         HandlerFunc
	createInactivityMonitor         func() inactivity.Monitor
	periodicRunner                  periodic.Func
	listenMutex                     sync.Mutex
	maxMessageSize                  uint32
	connectionCacheSize             uint16
	disableTCPSignalMessageCSM      bool
	blockwiseEnable                 bool
	blockwiseSZX                    blockwise.SZX
	disablePeerTCPSignalMessageCSMs bool
}

func NewServer(opt ...ServerOption) *Server {
	opts := defaultServerOptions
	for _, o := range opt {
		o.apply(&opts)
	}

	ctx, cancel := context.WithCancel(opts.ctx)

	if opts.createInactivityMonitor == nil {
		opts.createInactivityMonitor = func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		}
	}
	if opts.messagePool == nil {
		opts.messagePool = pool.New(0, 0)
	}

	if opts.errors == nil {
		opts.errors = func(error) {
			// default no-op
		}
	}
	errorsFunc := opts.errors
	// assign updated func to opts.errors so opts.handler also uses the updated error handler
	opts.errors = func(err error) {
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
			// this error was produced by cancellation context or closing connection.
			return
		}
		errorsFunc(fmt.Errorf("tcp: %w", err))
	}

	return &Server{
		ctx:                             ctx,
		cancel:                          cancel,
		handler:                         opts.handler,
		maxMessageSize:                  opts.maxMessageSize,
		errors:                          opts.errors,
		goPool:                          opts.goPool,
		blockwiseSZX:                    opts.blockwiseSZX,
		blockwiseEnable:                 opts.blockwiseEnable,
		blockwiseTransferTimeout:        opts.blockwiseTransferTimeout,
		disablePeerTCPSignalMessageCSMs: opts.disablePeerTCPSignalMessageCSMs,
		disableTCPSignalMessageCSM:      opts.disableTCPSignalMessageCSM,
		onNewClientConn:                 opts.onNewClientConn,
		createInactivityMonitor:         opts.createInactivityMonitor,
		periodicRunner:                  opts.periodicRunner,
		connectionCacheSize:             opts.connectionCacheSize,
		messagePool:                     opts.messagePool,
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

func (s *Server) checkAcceptError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	switch {
	case errors.Is(err, coapNet.ErrListenerIsClosed):
		s.Stop()
		return false, nil
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		select {
		case <-s.ctx.Done():
		default:
			s.errors(fmt.Errorf("cannot accept connection: %w", err))
			return true, nil
		}
		return false, nil
	default:
		return true, nil
	}
}

func (s *Server) serveConnection(connections *connections.Connections, rw net.Conn) {
	var cc *ClientConn
	monitor := s.createInactivityMonitor()
	cc = s.createClientConn(coapNet.NewConn(rw), monitor)
	if s.onNewClientConn != nil {
		if tlscon, ok := rw.(*tls.Conn); ok {
			s.onNewClientConn(cc, tlscon)
		} else {
			s.onNewClientConn(cc, nil)
		}
	}
	connections.Store(cc)
	defer connections.Delete(cc)

	if err := cc.Run(); err != nil {
		s.errors(fmt.Errorf("%v: %w", cc.RemoteAddr(), err))
	}
}

func (s *Server) Serve(l Listener) error {
	if s.blockwiseSZX > blockwise.SZXBERT {
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
	s.periodicRunner(func(now time.Time) bool {
		connections.CheckExpirations(now)
		return s.ctx.Err() == nil
	})
	defer connections.Close()

	for {
		rw, err := l.AcceptWithContext(s.ctx)
		ok, err := s.checkAcceptError(err)
		if err != nil {
			return err
		}
		if !ok {
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
		s.errors(fmt.Errorf("cannot close listener: %w", err))
	}
}

func (s *Server) createClientConn(connection *coapNet.Conn, monitor inactivity.Monitor) *ClientConn {
	var blockWise *blockwise.BlockWise
	if s.blockwiseEnable {
		blockWise = blockwise.NewBlockWise(
			bwCreateAcquireMessage(s.messagePool),
			bwCreateReleaseMessage(s.messagePool),
			s.blockwiseTransferTimeout,
			s.errors,
			false,
			func(token message.Token) (blockwise.Message, bool) {
				return nil, false
			},
		)
	}
	obsHandler := NewHandlerContainer()
	cc := NewClientConn(
		NewSession(
			s.ctx,
			connection,
			NewObservationHandler(obsHandler, s.handler),
			s.maxMessageSize,
			s.goPool,
			s.errors,
			s.blockwiseSZX,
			blockWise,
			s.disablePeerTCPSignalMessageCSMs,
			s.disableTCPSignalMessageCSM,
			true,
			monitor,
			s.connectionCacheSize,
			s.messagePool,
		),
		obsHandler, kitSync.NewMap(),
	)

	return cc
}
