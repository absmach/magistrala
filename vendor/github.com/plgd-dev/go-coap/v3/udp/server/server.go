package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/go-coap/v3/udp/client"
)

type Server struct {
	doneCtx           context.Context
	ctx               context.Context
	multicastRequests *client.RequestsMap
	multicastHandler  *coapSync.Map[uint64, HandlerFunc]
	serverStartedChan chan struct{}
	doneCancel        context.CancelFunc
	cancel            context.CancelFunc
	responseMsgCache  *cache.Cache[string, []byte]

	connsMutex sync.Mutex
	conns      map[string]*client.Conn

	listenMutex sync.Mutex
	listen      *coapNet.UDPConn

	cfg *Config
}

// A Option sets options such as credentials, codec and keepalive parameters, etc.
type Option interface {
	UDPServerApply(cfg *Config)
}

func New(opt ...Option) *Server {
	cfg := DefaultConfig
	for _, o := range opt {
		o.UDPServerApply(&cfg)
	}

	if cfg.Errors == nil {
		cfg.Errors = func(error) {
			// default no-op
		}
	}

	if cfg.GetMID == nil {
		cfg.GetMID = message.GetMID
	}

	if cfg.GetToken == nil {
		cfg.GetToken = message.GetToken
	}

	if cfg.CreateInactivityMonitor == nil {
		cfg.CreateInactivityMonitor = func() client.InactivityMonitor {
			return inactivity.NewNilMonitor[*client.Conn]()
		}
	}
	if cfg.MessagePool == nil {
		cfg.MessagePool = pool.New(0, 0)
	}

	ctx, cancel := context.WithCancel(cfg.Ctx)
	serverStartedChan := make(chan struct{})

	doneCtx, doneCancel := context.WithCancel(context.Background())
	errorsFunc := cfg.Errors
	cfg.Errors = func(err error) {
		if coapNet.IsCancelOrCloseError(err) {
			// this error was produced by cancellation context or closing connection.
			return
		}
		errorsFunc(fmt.Errorf("udp: %w", err))
	}
	return &Server{
		ctx:               ctx,
		cancel:            cancel,
		multicastHandler:  coapSync.NewMap[uint64, HandlerFunc](),
		multicastRequests: coapSync.NewMap[uint64, *pool.Message](),
		serverStartedChan: serverStartedChan,
		doneCtx:           doneCtx,
		doneCancel:        doneCancel,
		responseMsgCache:  cache.NewCache[string, []byte](),
		conns:             make(map[string]*client.Conn),

		cfg: &cfg,
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

func (s *Server) closeConnection(cc *client.Conn) {
	if err := cc.Close(); err != nil {
		s.cfg.Errors(fmt.Errorf("cannot close connection: %w", err))
	}
}

func (s *Server) Serve(l *coapNet.UDPConn) error {
	if s.cfg.BlockwiseSZX > blockwise.SZX1024 {
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

	m := make([]byte, s.cfg.MaxMessageSize)
	var wg sync.WaitGroup

	s.cfg.PeriodicRunner(func(now time.Time) bool {
		s.handleInactivityMonitors(now)
		s.responseMsgCache.CheckExpirations(now)
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
		cc, err := s.getConn(l, raddr, true)
		if err != nil {
			s.cfg.Errors(fmt.Errorf("%v: cannot get client connection: %w", raddr, err))
			continue
		}
		err = cc.Process(buf)
		if err != nil {
			s.closeConnection(cc)
			s.cfg.Errors(fmt.Errorf("%v: cannot process packet: %w", cc.RemoteAddr(), err))
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
			s.cfg.Errors(fmt.Errorf("cannot close listener: %w", errC))
		}
	}
	s.closeSessions()
}

func (s *Server) closeSessions() {
	s.connsMutex.Lock()
	conns := s.conns
	s.conns = make(map[string]*client.Conn)
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

func (s *Server) getConns() []*client.Conn {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()
	conns := make([]*client.Conn, 0, 32)
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	return conns
}

func (s *Server) handleInactivityMonitors(now time.Time) {
	for _, cc := range s.getConns() {
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

func getClose(cc *client.Conn) func() {
	v := cc.Context().Value(closeKey)
	if v == nil {
		return nil
	}
	closeFn, ok := v.(func())
	if !ok {
		panic(fmt.Errorf("invalid type(%T) of context value for key %s", v, closeKey))
	}
	return closeFn
}

func (s *Server) getOrCreateConn(udpConn *coapNet.UDPConn, raddr *net.UDPAddr) (cc *client.Conn, created bool) {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()
	key := raddr.String()
	cc = s.conns[key]

	if cc != nil {
		return cc, false
	}

	createBlockWise := func(cc *client.Conn) *blockwise.BlockWise[*client.Conn] {
		return nil
	}
	if s.cfg.BlockwiseEnable {
		createBlockWise = func(cc *client.Conn) *blockwise.BlockWise[*client.Conn] {
			v := cc
			return blockwise.New(
				v,
				s.cfg.BlockwiseTransferTimeout,
				s.cfg.Errors,
				func(token message.Token) (*pool.Message, bool) {
					msg, ok := v.GetObservationRequest(token)
					if ok {
						return msg, ok
					}
					return s.multicastRequests.LoadWithFunc(token.Hash(), func(m *pool.Message) *pool.Message {
						msg := v.AcquireMessage(m.Context())
						msg.ResetOptionsTo(m.Options())
						msg.SetCode(m.Code())
						msg.SetToken(m.Token())
						msg.SetMessageID(m.MessageID())
						return msg
					})
				})
		}
	}
	session := NewSession(
		s.ctx,
		s.doneCtx,
		udpConn,
		raddr,
		s.cfg.MaxMessageSize,
		s.cfg.MTU,
		false,
	)
	monitor := s.cfg.CreateInactivityMonitor()
	cfg := client.DefaultConfig
	cfg.TransmissionNStart = s.cfg.TransmissionNStart
	cfg.TransmissionAcknowledgeTimeout = s.cfg.TransmissionAcknowledgeTimeout
	cfg.TransmissionMaxRetransmit = s.cfg.TransmissionMaxRetransmit
	cfg.Handler = func(w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message) {
		h, ok := s.multicastHandler.Load(r.Token().Hash())
		if ok {
			h(w, r)
			return
		}
		s.cfg.Handler(w, r)
	}
	cfg.BlockwiseSZX = s.cfg.BlockwiseSZX
	cfg.Errors = s.cfg.Errors
	cfg.GetMID = s.cfg.GetMID
	cfg.GetToken = s.cfg.GetToken
	cfg.MessagePool = s.cfg.MessagePool
	cfg.ProcessReceivedMessage = s.cfg.ProcessReceivedMessage
	cfg.ReceivedMessageQueueSize = s.cfg.ReceivedMessageQueueSize

	cc = client.NewConn(
		session,
		createBlockWise,
		monitor,
		&cfg,
	)
	cc.SetContextValue(closeKey, func() {
		if err := session.Close(); err != nil {
			s.cfg.Errors(fmt.Errorf("cannot close session: %w", err))
		}
		session.shutdown()
	})
	cc.AddOnClose(func() {
		s.connsMutex.Lock()
		defer s.connsMutex.Unlock()
		if cc == s.conns[key] {
			delete(s.conns, key)
		}
	})
	s.conns[key] = cc
	return cc, true
}

func (s *Server) getConn(l *coapNet.UDPConn, raddr *net.UDPAddr, firstTime bool) (*client.Conn, error) {
	cc, created := s.getOrCreateConn(l, raddr)
	if created {
		if s.cfg.OnNewConn != nil {
			s.cfg.OnNewConn(cc)
		}
	} else {
		// check if client is not expired now + 10ms  - if so, close it
		// 10ms - The expected maximum time taken by cc.CheckExpirations and cc.InactivityMonitor().Notify()
		cc.CheckExpirations(time.Now().Add(10 * time.Millisecond))
		if cc.Context().Err() == nil {
			// if client is not closed, extend expiration time
			cc.InactivityMonitor().Notify()
		}
	}

	if cc.Context().Err() != nil {
		// connection is closed so we need to create new one
		if closeFn := getClose(cc); closeFn != nil {
			closeFn()
		}
		if firstTime {
			return s.getConn(l, raddr, false)
		}
		return nil, fmt.Errorf("connection is closed")
	}
	return cc, nil
}

func (s *Server) NewConn(addr *net.UDPAddr) (*client.Conn, error) {
	l := s.getListener()
	if l == nil {
		// server is not started/stopped
		return nil, fmt.Errorf("server is not running")
	}
	return s.getConn(l, addr, true)
}
