package udp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapNet "github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/udp/client"
	udpMessage "github.com/plgd-dev/go-coap/v2/udp/message"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
	kitSync "github.com/plgd-dev/kit/sync"
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

type BlockwiseFactoryFunc = func(getSendedRequest func(token message.Token) (blockwise.Message, bool)) *blockwise.BlockWise

type OnNewClientConnFunc = func(cc *client.ClientConn)

type GetMIDFunc = func() uint16

var defaultServerOptions = serverOptions{
	ctx:            context.Background(),
	maxMessageSize: 64 * 1024,
	handler: func(w *client.ResponseWriter, r *pool.Message) {
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
}

type serverOptions struct {
	ctx                            context.Context
	maxMessageSize                 int
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	createInactivityMonitor        func() inactivity.Monitor
	net                            string
	blockwiseSZX                   blockwise.SZX
	blockwiseEnable                bool
	blockwiseTransferTimeout       time.Duration
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      int
	getMID                         GetMIDFunc
}

type Server struct {
	maxMessageSize                 int
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	createInactivityMonitor        func() inactivity.Monitor
	blockwiseSZX                   blockwise.SZX
	blockwiseEnable                bool
	blockwiseTransferTimeout       time.Duration
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      int
	getMID                         GetMIDFunc

	conns             map[string]*client.ClientConn
	connsMutex        sync.Mutex
	ctx               context.Context
	cancel            context.CancelFunc
	serverStartedChan chan struct{}

	multicastRequests *kitSync.Map
	multicastHandler  *client.HandlerContainer

	listen      *coapNet.UDPConn
	listenMutex sync.Mutex
}

func NewServer(opt ...ServerOption) *Server {
	opts := defaultServerOptions
	for _, o := range opt {
		o.apply(&opts)
	}

	if opts.errors == nil {
		opts.errors = func(error) {}
	}

	if opts.getMID == nil {
		opts.getMID = udpMessage.GetMID
	}

	if opts.createInactivityMonitor == nil {
		opts.createInactivityMonitor = func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		}
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	serverStartedChan := make(chan struct{})

	return &Server{
		ctx:            ctx,
		cancel:         cancel,
		handler:        opts.handler,
		maxMessageSize: opts.maxMessageSize,
		errors: func(err error) {
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
		s.listenMutex.Lock()
		defer s.listenMutex.Unlock()
		s.listen = nil
		s.serverStartedChan = make(chan struct{}, 1)
	}()

	m := make([]byte, s.maxMessageSize)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.handleInactivityMonitors()
	}()

	for {
		buf := m
		n, raddr, err := l.ReadWithContext(s.ctx, buf)
		if err != nil {
			wg.Wait()

			select {
			case <-s.ctx.Done():
				return nil
			default:
				return err
			}
		}
		buf = buf[:n]
		cc, created := s.getOrCreateClientConn(l, raddr)
		if created {
			if s.onNewClientConn != nil {
				s.onNewClientConn(cc)
			}
		}
		err = cc.Process(buf)
		if err != nil {
			cc.Close()
			s.errors(fmt.Errorf("%v: %w", cc.RemoteAddr(), err))
		}
	}
}

// Stop stops server without wait of ends Serve function.
func (s *Server) Stop() {
	s.cancel()
	s.closeSessions()
}

func (s *Server) closeSessions() {
	s.connsMutex.Lock()
	conns := s.conns
	s.conns = make(map[string]*client.ClientConn)
	s.connsMutex.Unlock()
	for _, cc := range conns {
		cc.Close()
		close := getClose(cc)
		if close != nil {
			close()
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

const inactivityMonitorKey = "gocoapInactivityMonitor"
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

func (s *Server) handleInactivityMonitors() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, cc := range s.getClientConns() {
				select {
				case <-cc.Context().Done():
					close := getClose(cc)
					if close != nil {
						close()
					}
					continue
				default:
					monitor := getInactivityMonitor(cc)
					monitor.CheckInactivity(cc)
				}
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func getInactivityMonitor(cc *client.ClientConn) inactivity.Monitor {
	v := cc.Context().Value(inactivityMonitorKey)
	if v == nil {
		return nil
	}
	return v.(inactivity.Monitor)
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
				bwAcquireMessage,
				bwReleaseMessage,
				s.blockwiseTransferTimeout,
				s.errors,
				false,
				bwCreateHandlerFunc(s.multicastRequests),
			)
		}
		obsHandler := client.NewHandlerContainer()
		session := NewSession(
			s.ctx,
			UDPConn,
			raddr,
			s.maxMessageSize,
			false,
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
		)
		cc.SetContextValue(inactivityMonitorKey, monitor)
		cc.SetContextValue(closeKey, func() {
			session.close()
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
