package udp

import (
	"context"
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
	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/go-coap/v2/udp/client"
	udpMessage "github.com/plgd-dev/go-coap/v2/udp/message"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
	kitSync "github.com/plgd-dev/kit/v2/sync"
)

// A ServerOption sets options such as credentials, codec and keepalive parameters, etc.
type ServerOption interface {
	apply(*serverOptions)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as COAP handlers.
type HandlerFunc = func(*client.ResponseWriter, *pool.Message)

type ErrorFunc = func(error)

type GoPoolFunc = func(func()) error

type BlockwiseFactoryFunc = func(getSentRequest func(token message.Token) (blockwise.Message, bool)) *blockwise.BlockWise

type OnNewClientConnFunc = func(cc *client.ClientConn)

type GetMIDFunc = func() uint16

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
		createInactivityMonitor: func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		},
		blockwiseEnable:                true,
		blockwiseSZX:                   blockwise.SZX1024,
		blockwiseTransferTimeout:       time.Second * 3,
		onNewClientConn:                func(cc *client.ClientConn) {},
		transmissionNStart:             time.Second,
		transmissionAcknowledgeTimeout: time.Second * 2,
		transmissionMaxRetransmit:      4,
		getMID:                         udpMessage.GetMID,
		periodicRunner: func(f func(now time.Time) bool) {
			go func() {
				for f(time.Now()) {
					time.Sleep(4 * time.Second)
				}
			}()
		},
		messagePool: pool.New(1024, 1600),
	}
	opts.handler = func(w *client.ResponseWriter, r *pool.Message) {
		if err := w.SetResponse(codes.NotFound, message.TextPlain, nil); err != nil {
			opts.errors(fmt.Errorf("udp server: cannot set response: %w", err))
		}
	}
	return opts
}()

type serverOptions struct {
	ctx                            context.Context
	messagePool                    *pool.Pool
	blockwiseTransferTimeout       time.Duration
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	createInactivityMonitor        func() inactivity.Monitor
	periodicRunner                 periodic.Func
	getMID                         GetMIDFunc
	handler                        HandlerFunc
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	maxMessageSize                 uint32
	transmissionMaxRetransmit      uint32
	blockwiseEnable                bool
	blockwiseSZX                   blockwise.SZX
}

type Server struct {
	doneCtx context.Context
	ctx     context.Context

	listen *coapNet.UDPConn

	goPool                         GoPoolFunc
	createInactivityMonitor        func() inactivity.Monitor
	periodicRunner                 periodic.Func
	cache                          *cache.Cache
	blockwiseTransferTimeout       time.Duration
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration

	conns  map[string]*client.ClientConn
	getMID GetMIDFunc

	messagePool       *pool.Pool
	doneCancel        context.CancelFunc
	handler           HandlerFunc
	cancel            context.CancelFunc
	serverStartedChan chan struct{}

	multicastRequests *kitSync.Map
	multicastHandler  *client.HandlerContainer

	errors                    ErrorFunc
	connsMutex                sync.Mutex
	listenMutex               sync.Mutex
	transmissionMaxRetransmit uint32
	maxMessageSize            uint32
	blockwiseEnable           bool
	blockwiseSZX              blockwise.SZX
}

func NewServer(opt ...ServerOption) *Server {
	opts := defaultServerOptions
	for _, o := range opt {
		o.apply(&opts)
	}

	if opts.errors == nil {
		opts.errors = func(error) {
			// default no-op
		}
	}

	if opts.getMID == nil {
		opts.getMID = udpMessage.GetMID
	}

	if opts.createInactivityMonitor == nil {
		opts.createInactivityMonitor = func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		}
	}
	if opts.messagePool == nil {
		opts.messagePool = pool.New(0, 0)
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	serverStartedChan := make(chan struct{})

	doneCtx, doneCancel := context.WithCancel(context.Background())

	return &Server{
		ctx:            ctx,
		cancel:         cancel,
		handler:        opts.handler,
		maxMessageSize: opts.maxMessageSize,
		errors: func(err error) {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
				// this error was produced by cancellation context or closing connection.
				return
			}
			opts.errors(fmt.Errorf("udp: %w", err))
		},
		goPool:                         opts.goPool,
		createInactivityMonitor:        opts.createInactivityMonitor,
		blockwiseSZX:                   opts.blockwiseSZX,
		blockwiseEnable:                opts.blockwiseEnable,
		blockwiseTransferTimeout:       opts.blockwiseTransferTimeout,
		multicastHandler:               client.NewHandlerContainer(),
		multicastRequests:              kitSync.NewMap(),
		serverStartedChan:              serverStartedChan,
		onNewClientConn:                opts.onNewClientConn,
		transmissionNStart:             opts.transmissionNStart,
		transmissionAcknowledgeTimeout: opts.transmissionAcknowledgeTimeout,
		transmissionMaxRetransmit:      opts.transmissionMaxRetransmit,
		getMID:                         opts.getMID,
		periodicRunner:                 opts.periodicRunner,
		doneCtx:                        doneCtx,
		doneCancel:                     doneCancel,
		cache:                          cache.NewCache(),
		messagePool:                    opts.messagePool,

		conns: make(map[string]*client.ClientConn),
	}
}

func (s *Server) checkAndSetListener(l *coapNet.UDPConn) error {
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	if s.listen != nil {
		return fmt.Errorf("server already serve: %v", s.listen.LocalAddr().String())
	}
	s.listen = l
	close(s.serverStartedChan)
	return nil
}

func (s *Server) closeConnection(cc *client.ClientConn) {
	if err := cc.Close(); err != nil {
		s.errors(fmt.Errorf("cannot close connection: %w", err))
	}
}

func (s *Server) Serve(l *coapNet.UDPConn) error {
	if s.blockwiseSZX > blockwise.SZX1024 {
		return fmt.Errorf("invalid blockwiseSZX")
	}

	err := s.checkAndSetListener(l)
	if err != nil {
		return err
	}

	defer func() {
		s.closeSessions()
		s.doneCancel()
		s.listenMutex.Lock()
		defer s.listenMutex.Unlock()
		s.listen = nil
		s.serverStartedChan = make(chan struct{}, 1)
	}()

	m := make([]byte, s.maxMessageSize)
	var wg sync.WaitGroup

	s.periodicRunner(func(now time.Time) bool {
		s.handleInactivityMonitors(now)
		s.cache.CheckExpirations(now)
		return s.ctx.Err() == nil
	})

	for {
		buf := m
		n, raddr, err := l.ReadWithContext(s.ctx, buf)
		if err != nil {
			wg.Wait()

			select {
			case <-s.ctx.Done():
				return nil
			default:
				if coapNet.IsCancelOrCloseError(err) {
					return nil
				}
				return err
			}
		}
		buf = buf[:n]
		cc := s.getClientConn(l, raddr)
		err = cc.Process(buf)
		if err != nil {
			s.closeConnection(cc)
			s.errors(fmt.Errorf("%v: %w", cc.RemoteAddr(), err))
		}
	}
}

func (s *Server) getListener() *coapNet.UDPConn {
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	return s.listen
}

// Stop stops server without wait of ends Serve function.
func (s *Server) Stop() {
	s.cancel()
	l := s.getListener()
	if l != nil {
		if errC := l.Close(); errC != nil {
			s.errors(fmt.Errorf("cannot close listener: %w", errC))
		}
	}
	s.closeSessions()
}

func (s *Server) closeSessions() {
	s.connsMutex.Lock()
	conns := s.conns
	s.conns = make(map[string]*client.ClientConn)
	s.connsMutex.Unlock()
	for _, cc := range conns {
		s.closeConnection(cc)
		if closeFn := getClose(cc); closeFn != nil {
			closeFn()
		}
	}
}

func (s *Server) conn() *coapNet.UDPConn {
	s.listenMutex.Lock()
	serverStartedChan := s.serverStartedChan
	s.listenMutex.Unlock()
	select {
	case <-serverStartedChan:
	case <-s.ctx.Done():
	}
	s.listenMutex.Lock()
	defer s.listenMutex.Unlock()
	return s.listen
}

const closeKey = "gocoapCloseConnection"

func (s *Server) getClientConns() []*client.ClientConn {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()
	conns := make([]*client.ClientConn, 0, 32)
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	return conns
}

func (s *Server) handleInactivityMonitors(now time.Time) {
	for _, cc := range s.getClientConns() {
		select {
		case <-cc.Context().Done():
			if closeFn := getClose(cc); closeFn != nil {
				closeFn()
			}
			continue
		default:
			cc.CheckExpirations(now)
		}
	}
}

func getClose(cc *client.ClientConn) func() {
	v := cc.Context().Value(closeKey)
	if v == nil {
		return nil
	}
	return v.(func())
}

func (s *Server) getOrCreateClientConn(UDPConn *coapNet.UDPConn, raddr *net.UDPAddr) (cc *client.ClientConn, created bool) {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()
	key := raddr.String()
	cc = s.conns[key]
	if cc == nil {
		created = true
		var blockWise *blockwise.BlockWise
		if s.blockwiseEnable {
			blockWise = blockwise.NewBlockWise(
				bwCreateAcquireMessage(s.messagePool),
				bwCreateReleaseMessage(s.messagePool),
				s.blockwiseTransferTimeout,
				s.errors,
				false,
				bwCreateHandlerFunc(s.messagePool, s.multicastRequests),
			)
		}
		obsHandler := client.NewHandlerContainer()
		session := NewSession(
			s.ctx,
			UDPConn,
			raddr,
			s.maxMessageSize,
			false,
			s.doneCtx,
		)
		monitor := s.createInactivityMonitor()
		cc = client.NewClientConn(
			session,
			obsHandler,
			s.multicastRequests,
			s.transmissionNStart,
			s.transmissionAcknowledgeTimeout,
			s.transmissionMaxRetransmit,
			client.NewObservationHandler(obsHandler, func(w *client.ResponseWriter, r *pool.Message) {
				h, err := s.multicastHandler.Get(r.Token())
				if err == nil {
					h(w, r)
					return
				}
				s.handler(w, r)
			}),
			s.blockwiseSZX,
			blockWise,
			s.goPool,
			s.errors,
			s.getMID,
			monitor,
			s.cache,
			s.messagePool,
		)
		cc.SetContextValue(closeKey, func() {
			if err := session.Close(); err != nil {
				s.errors(fmt.Errorf("cannot close session: %w", err))
			}
			session.shutdown()
		})
		cc.AddOnClose(func() {
			s.connsMutex.Lock()
			defer s.connsMutex.Unlock()
			delete(s.conns, key)
		})
		s.conns[key] = cc
	}
	return cc, created
}

func (s *Server) getClientConn(l *coapNet.UDPConn, raddr *net.UDPAddr) *client.ClientConn {
	cc, created := s.getOrCreateClientConn(l, raddr)
	if created {
		if s.onNewClientConn != nil {
			s.onNewClientConn(cc)
		}
	}
	return cc
}

func (s *Server) NewClientConn(addr *net.UDPAddr) (*client.ClientConn, error) {
	l := s.getListener()
	if l == nil {
		return nil, fmt.Errorf("server is not running")
	}
	return s.getClientConn(l, addr), nil
}
