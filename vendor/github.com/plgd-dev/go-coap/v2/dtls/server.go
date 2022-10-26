package dtls

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapNet "github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/connections"
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

// OnNewClientConnFunc is the callback for new connections.
//
// Note: Calling `dtlsConn.Close()` is forbidden, and `dtlsConn` should be treated as a
// "read-only" parameter, mainly used to get the peer certificate from the underlining connection
type OnNewClientConnFunc = func(cc *client.ClientConn, dtlsConn *dtls.Conn)

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
		blockwiseTransferTimeout:       time.Second * 5,
		onNewClientConn:                func(cc *client.ClientConn, dtlsConn *dtls.Conn) {},
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
	opts.handler = func(w *client.ResponseWriter, m *pool.Message) {
		if err := w.SetResponse(codes.NotFound, message.TextPlain, nil); err != nil {
			opts.errors(fmt.Errorf("server handler: cannot set response: %w", err))
		}
	}
	return opts
}()

type serverOptions struct {
	ctx                            context.Context
	messagePool                    *pool.Pool
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	createInactivityMonitor        func() inactivity.Monitor
	periodicRunner                 periodic.Func
	getMID                         GetMIDFunc
	blockwiseTransferTimeout       time.Duration
	onNewClientConn                OnNewClientConnFunc
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	maxMessageSize                 uint32
	transmissionMaxRetransmit      uint32
	blockwiseEnable                bool
	blockwiseSZX                   blockwise.SZX
}

// Listener defined used by coap
type Listener interface {
	Close() error
	AcceptWithContext(ctx context.Context) (net.Conn, error)
}

type Server struct {
	listen Listener

	ctx                     context.Context
	cache                   *cache.Cache
	goPool                  GoPoolFunc
	createInactivityMonitor func() inactivity.Monitor
	errors                  ErrorFunc
	cancel                  context.CancelFunc

	blockwiseTransferTimeout time.Duration
	onNewClientConn          OnNewClientConnFunc

	handler                        HandlerFunc
	transmissionAcknowledgeTimeout time.Duration
	messagePool                    *pool.Pool

	getMID                    GetMIDFunc
	periodicRunner            periodic.Func
	transmissionNStart        time.Duration
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

	ctx, cancel := context.WithCancel(opts.ctx)
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

	errorsFunc := opts.errors
	// assign updated func to opts.errors so opts.handler also uses the updated error handler
	opts.errors = func(err error) {
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
			// this error was produced by cancellation context or closing connection.
			return
		}
		errorsFunc(fmt.Errorf("dtls: %w", err))
	}

	return &Server{
		ctx:                            ctx,
		cancel:                         cancel,
		handler:                        opts.handler,
		maxMessageSize:                 opts.maxMessageSize,
		errors:                         opts.errors,
		goPool:                         opts.goPool,
		createInactivityMonitor:        opts.createInactivityMonitor,
		blockwiseSZX:                   opts.blockwiseSZX,
		blockwiseEnable:                opts.blockwiseEnable,
		blockwiseTransferTimeout:       opts.blockwiseTransferTimeout,
		onNewClientConn:                opts.onNewClientConn,
		transmissionNStart:             opts.transmissionNStart,
		transmissionAcknowledgeTimeout: opts.transmissionAcknowledgeTimeout,
		transmissionMaxRetransmit:      opts.transmissionMaxRetransmit,
		getMID:                         opts.getMID,
		periodicRunner:                 opts.periodicRunner,
		cache:                          cache.NewCache(),
		messagePool:                    opts.messagePool,
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

func (s *Server) serveConnection(connections *connections.Connections, cc *client.ClientConn) {
	connections.Store(cc)
	defer connections.Delete(cc)

	if err := cc.Run(); err != nil {
		s.errors(fmt.Errorf("%v: %w", cc.RemoteAddr(), err))
	}
}

func (s *Server) Serve(l Listener) error {
	if s.blockwiseSZX > blockwise.SZX1024 {
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
		var cc *client.ClientConn
		monitor := s.createInactivityMonitor()
		cc = s.createClientConn(coapNet.NewConn(rw), monitor)
		if s.onNewClientConn != nil {
			dtlsConn := rw.(*dtls.Conn)
			s.onNewClientConn(cc, dtlsConn)
		}
		go func() {
			defer wg.Done()
			s.serveConnection(connections, cc)
		}()
	}
}

// Stop stops server without wait of ends Serve function.
func (s *Server) Stop() {
	s.cancel()
	s.listenMutex.Lock()
	l := s.listen
	s.listen = nil
	s.listenMutex.Unlock()
	if l != nil {
		if err := l.Close(); err != nil {
			s.errors(fmt.Errorf("cannot close listener: %w", err))
		}
	}
}

func (s *Server) createClientConn(connection *coapNet.Conn, monitor inactivity.Monitor) *client.ClientConn {
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
	obsHandler := client.NewHandlerContainer()
	session := NewSession(
		s.ctx,
		connection,
		s.maxMessageSize,
		true,
	)
	cc := client.NewClientConn(
		session,
		obsHandler,
		kitSync.NewMap(),
		s.transmissionNStart,
		s.transmissionAcknowledgeTimeout,
		s.transmissionMaxRetransmit,
		client.NewObservationHandler(obsHandler, s.handler),
		s.blockwiseSZX,
		blockWise,
		s.goPool,
		s.errors,
		s.getMID,
		monitor,
		s.cache,
		s.messagePool,
	)

	return cc
}
