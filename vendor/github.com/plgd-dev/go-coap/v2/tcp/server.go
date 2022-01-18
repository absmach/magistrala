package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	kitSync "github.com/plgd-dev/kit/sync"

	"github.com/plgd-dev/go-coap/v2/message/codes"

	coapNet "github.com/plgd-dev/go-coap/v2/net"
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

type BlockwiseFactoryFunc = func(getSendedRequest func(token message.Token) (blockwise.Message, bool)) *blockwise.BlockWise

// OnNewClientConnFunc is the callback for new connections.
//
// Note: Calling `tlscon.Close()` is forbidden, and `tlscon` should be treated as a
// "read-only" parameter, mainly used to get the peer certificate from the underlining connection
type OnNewClientConnFunc = func(cc *ClientConn, tlscon *tls.Conn)

var defaultServerOptions = serverOptions{
	ctx:            context.Background(),
	maxMessageSize: 64 * 1024,
	handler: func(w *ResponseWriter, r *pool.Message) {
		w.SetResponse(codes.NotFound, message.TextPlain, nil)
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
	blockwiseEnable:          true,
	blockwiseSZX:             blockwise.SZX1024,
	blockwiseTransferTimeout: time.Second * 3,
	onNewClientConn:          func(cc *ClientConn, tlscon *tls.Conn) {},
	heartBeat:                time.Millisecond * 100,
	createInactivityMonitor: func() inactivity.Monitor {
		return inactivity.NewNilMonitor()
	},
}

type serverOptions struct {
	ctx                             context.Context
	maxMessageSize                  int
	handler                         HandlerFunc
	errors                          ErrorFunc
	goPool                          GoPoolFunc
	createInactivityMonitor         func() inactivity.Monitor
	blockwiseSZX                    blockwise.SZX
	blockwiseEnable                 bool
	blockwiseTransferTimeout        time.Duration
	onNewClientConn                 OnNewClientConnFunc
	heartBeat                       time.Duration
	disablePeerTCPSignalMessageCSMs bool
	disableTCPSignalMessageCSM      bool
}

// Listener defined used by coap
type Listener interface {
	Close() error
	AcceptWithContext(ctx context.Context) (net.Conn, error)
}

type Server struct {
	maxMessageSize                  int
	handler                         HandlerFunc
	errors                          ErrorFunc
	goPool                          GoPoolFunc
	createInactivityMonitor         func() inactivity.Monitor
	blockwiseSZX                    blockwise.SZX
	blockwiseEnable                 bool
	blockwiseTransferTimeout        time.Duration
	onNewClientConn                 OnNewClientConnFunc
	heartBeat                       time.Duration
	disablePeerTCPSignalMessageCSMs bool
	disableTCPSignalMessageCSM      bool

	ctx    context.Context
	cancel context.CancelFunc

	listen      Listener
	listenMutex sync.Mutex
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

	return &Server{
		ctx:            ctx,
		cancel:         cancel,
		handler:        opts.handler,
		maxMessageSize: opts.maxMessageSize,
		errors: func(err error) {
			opts.errors(fmt.Errorf("tcp: %w", err))
		},
		goPool:                          opts.goPool,
		blockwiseSZX:                    opts.blockwiseSZX,
		blockwiseEnable:                 opts.blockwiseEnable,
		blockwiseTransferTimeout:        opts.blockwiseTransferTimeout,
		heartBeat:                       opts.heartBeat,
		disablePeerTCPSignalMessageCSMs: opts.disablePeerTCPSignalMessageCSMs,
		disableTCPSignalMessageCSM:      opts.disableTCPSignalMessageCSM,
		onNewClientConn:                 opts.onNewClientConn,
		createInactivityMonitor:         opts.createInactivityMonitor,
	}
}

func (s *Server) checkAndSetListener(l Listener) error {
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	if s.listen != nil {
		return fmt.Errorf("server already serve listener")
	}
	s.listen = l
	return nil
}

func (s *Server) checkAcceptError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	switch err {
	case coapNet.ErrListenerIsClosed:
		s.Stop()
		return false, nil
	case context.DeadlineExceeded, context.Canceled:
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

func (s *Server) Serve(l Listener) error {
	if s.blockwiseSZX > blockwise.SZXBERT {
		return fmt.Errorf("invalid blockwiseSZX")
	}

	err := s.checkAndSetListener(l)
	if err != nil {
		return err
	}

	defer func() {
		s.listenMutex.Lock()
		defer s.listenMutex.Unlock()
		s.listen = nil
	}()
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		rw, err := l.AcceptWithContext(s.ctx)
		ok, err := s.checkAcceptError(err)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if rw != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var cc *ClientConn
				monitor := s.createInactivityMonitor()
				opts := []coapNet.ConnOption{
					coapNet.WithHeartBeat(s.heartBeat),
					coapNet.WithOnReadTimeout(func() error {
						monitor.CheckInactivity(cc)
						return nil
					}),
				}
				cc = s.createClientConn(coapNet.NewConn(rw, opts...), monitor)
				if s.onNewClientConn != nil {
					if tlscon, ok := rw.(*tls.Conn); ok {
						s.onNewClientConn(cc, tlscon)
					} else {
						s.onNewClientConn(cc, nil)
					}
				}
				err := cc.Run()
				if err != nil {
					s.errors(fmt.Errorf("%v: %w", cc.RemoteAddr(), err))
				}
			}()
		}
	}
}

// Stop stops server without wait of ends Serve function.
func (s *Server) Stop() {
	s.cancel()
}

func (s *Server) createClientConn(connection *coapNet.Conn, monitor inactivity.Monitor) *ClientConn {
	var blockWise *blockwise.BlockWise
	if s.blockwiseEnable {
		blockWise = blockwise.NewBlockWise(
			bwAcquireMessage,
			bwReleaseMessage,
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
			monitor),
		obsHandler, kitSync.NewMap(),
	)

	return cc
}
