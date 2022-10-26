package udp

import (
	"context"
	"fmt"
	"net"
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

var defaultDialOptions = func() dialOptions {
	opts := dialOptions{
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
		periodicRunner: func(f func(now time.Time) bool) {
			go func() {
				for f(time.Now()) {
					time.Sleep(4 * time.Second)
				}
			}()
		},
		dialer:                         &net.Dialer{Timeout: time.Second * 3},
		net:                            "udp",
		blockwiseSZX:                   blockwise.SZX1024,
		blockwiseEnable:                true,
		blockwiseTransferTimeout:       time.Second * 3,
		transmissionNStart:             time.Second,
		transmissionAcknowledgeTimeout: time.Second * 2,
		transmissionMaxRetransmit:      4,
		getMID:                         udpMessage.GetMID,
		createInactivityMonitor: func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		},
		messagePool: pool.New(1024, 1600),
	}
	opts.handler = func(w *client.ResponseWriter, r *pool.Message) {
		switch r.Code() {
		case codes.POST, codes.PUT, codes.GET, codes.DELETE:
			if err := w.SetResponse(codes.NotFound, message.TextPlain, nil); err != nil {
				opts.errors(fmt.Errorf("udp client: cannot set response: %w", err))
			}
		}
	}
	return opts
}()

type dialOptions struct {
	net                            string
	ctx                            context.Context
	getMID                         GetMIDFunc
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	dialer                         *net.Dialer
	periodicRunner                 periodic.Func
	messagePool                    *pool.Pool
	blockwiseTransferTimeout       time.Duration
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	createInactivityMonitor        func() inactivity.Monitor
	maxMessageSize                 uint32
	transmissionMaxRetransmit      uint32
	closeSocket                    bool
	blockwiseSZX                   blockwise.SZX
	blockwiseEnable                bool
}

// A DialOption sets options such as credentials, keepalive parameters, etc.
type DialOption interface {
	applyDial(*dialOptions)
}

// Dial creates a client connection to the given target.
func Dial(target string, opts ...DialOption) (*client.ClientConn, error) {
	cfg := defaultDialOptions
	for _, o := range opts {
		o.applyDial(&cfg)
	}

	c, err := cfg.dialer.DialContext(cfg.ctx, cfg.net, target)
	if err != nil {
		return nil, err
	}
	conn, ok := c.(*net.UDPConn)
	if !ok {
		return nil, fmt.Errorf("unsupported connection type: %T", c)
	}
	opts = append(opts, WithCloseSocket())
	return Client(conn, opts...), nil
}

func bwCreateAcquireMessage(messagePool *pool.Pool) func(ctx context.Context) blockwise.Message {
	return func(ctx context.Context) blockwise.Message {
		return messagePool.AcquireMessage(ctx)
	}
}

func bwCreateReleaseMessage(messagePool *pool.Pool) func(m blockwise.Message) {
	return func(m blockwise.Message) {
		messagePool.ReleaseMessage(m.(*pool.Message))
	}
}

func bwCreateHandlerFunc(messagePool *pool.Pool, observatioRequests *kitSync.Map) func(token message.Token) (blockwise.Message, bool) {
	return func(token message.Token) (blockwise.Message, bool) {
		msg, ok := observatioRequests.LoadWithFunc(token.Hash(), func(v interface{}) interface{} {
			r := v.(*pool.Message)
			d := messagePool.AcquireMessage(r.Context())
			d.ResetOptionsTo(r.Options())
			d.SetCode(r.Code())
			d.SetToken(r.Token())
			d.SetMessageID(r.MessageID())
			return d
		})
		if !ok {
			return nil, ok
		}
		bwMessage := msg.(blockwise.Message)
		return bwMessage, ok
	}
}

// Client creates client over udp connection.
func Client(conn *net.UDPConn, opts ...DialOption) *client.ClientConn {
	cfg := defaultDialOptions
	for _, o := range opts {
		o.applyDial(&cfg)
	}
	if cfg.errors == nil {
		cfg.errors = func(error) {
			// default no-op
		}
	}
	if cfg.createInactivityMonitor == nil {
		cfg.createInactivityMonitor = func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		}
	}
	if cfg.messagePool == nil {
		cfg.messagePool = pool.New(0, 0)
	}

	errorsFunc := cfg.errors
	cfg.errors = func(err error) {
		if coapNet.IsCancelOrCloseError(err) {
			// this error was produced by cancellation context or closing connection.
			return
		}
		errorsFunc(fmt.Errorf("udp: %v: %w", conn.RemoteAddr(), err))
	}

	addr, _ := conn.RemoteAddr().(*net.UDPAddr)
	observatioRequests := kitSync.NewMap()
	var blockWise *blockwise.BlockWise
	if cfg.blockwiseEnable {
		blockWise = blockwise.NewBlockWise(
			bwCreateAcquireMessage(cfg.messagePool),
			bwCreateReleaseMessage(cfg.messagePool),
			cfg.blockwiseTransferTimeout,
			cfg.errors,
			false,
			bwCreateHandlerFunc(cfg.messagePool, observatioRequests),
		)
	}

	observationTokenHandler := client.NewHandlerContainer()
	monitor := cfg.createInactivityMonitor()
	cache := cache.NewCache()
	l := coapNet.NewUDPConn(cfg.net, conn, coapNet.WithErrors(cfg.errors))
	session := NewSession(cfg.ctx,
		l,
		addr,
		cfg.maxMessageSize,
		cfg.closeSocket,
		context.Background(),
	)
	cc := client.NewClientConn(session,
		observationTokenHandler, observatioRequests, cfg.transmissionNStart, cfg.transmissionAcknowledgeTimeout, cfg.transmissionMaxRetransmit,
		client.NewObservationHandler(observationTokenHandler, cfg.handler),
		cfg.blockwiseSZX,
		blockWise,
		cfg.goPool,
		cfg.errors,
		cfg.getMID,
		monitor,
		cache,
		cfg.messagePool,
	)
	cfg.periodicRunner(func(now time.Time) bool {
		cc.CheckExpirations(now)
		return cc.Context().Err() == nil
	})

	go func() {
		err := cc.Run()
		if err != nil {
			cfg.errors(err)
		}
	}()

	return cc
}
