package udp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	kitSync "github.com/plgd-dev/kit/sync"

	"github.com/plgd-dev/go-coap/v2/message/codes"
	coapNet "github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/udp/client"
	udpMessage "github.com/plgd-dev/go-coap/v2/udp/message"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

var defaultDialOptions = dialOptions{
	ctx:            context.Background(),
	maxMessageSize: 64 * 1024,
	heartBeat:      time.Millisecond * 100,
	handler: func(w *client.ResponseWriter, r *pool.Message) {
		switch r.Code() {
		case codes.POST, codes.PUT, codes.GET, codes.DELETE:
			w.SetResponse(codes.NotFound, message.TextPlain, nil)
		}
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
}

type dialOptions struct {
	ctx                            context.Context
	maxMessageSize                 int
	heartBeat                      time.Duration
	handler                        HandlerFunc
	errors                         ErrorFunc
	goPool                         GoPoolFunc
	dialer                         *net.Dialer
	net                            string
	blockwiseSZX                   blockwise.SZX
	blockwiseEnable                bool
	blockwiseTransferTimeout       time.Duration
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      int
	getMID                         GetMIDFunc
	closeSocket                    bool
	createInactivityMonitor        func() inactivity.Monitor
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

func bwAcquireMessage(ctx context.Context) blockwise.Message {
	return pool.AcquireMessage(ctx)
}

func bwReleaseMessage(m blockwise.Message) {
	pool.ReleaseMessage(m.(*pool.Message))
}

func bwCreateHandlerFunc(observatioRequests *kitSync.Map) func(token message.Token) (blockwise.Message, bool) {
	return func(token message.Token) (blockwise.Message, bool) {
		msg, ok := observatioRequests.LoadWithFunc(token.String(), func(v interface{}) interface{} {
			r := v.(*pool.Message)
			d := pool.AcquireMessage(r.Context())
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
		cfg.errors = func(error) {}
	}
	if cfg.createInactivityMonitor == nil {
		cfg.createInactivityMonitor = func() inactivity.Monitor {
			return inactivity.NewNilMonitor()
		}
	}

	errors := cfg.errors
	cfg.errors = func(err error) {
		errors(fmt.Errorf("udp: %v: %w", conn.RemoteAddr(), err))
	}

	addr, _ := conn.RemoteAddr().(*net.UDPAddr)
	observatioRequests := kitSync.NewMap()
	var blockWise *blockwise.BlockWise
	if cfg.blockwiseEnable {
		blockWise = blockwise.NewBlockWise(
			bwAcquireMessage,
			bwReleaseMessage,
			cfg.blockwiseTransferTimeout,
			cfg.errors,
			false,
			bwCreateHandlerFunc(observatioRequests),
		)
	}

	observationTokenHandler := client.NewHandlerContainer()
	monitor := cfg.createInactivityMonitor()
	var cc *client.ClientConn
	l := coapNet.NewUDPConn(cfg.net, conn, coapNet.WithHeartBeat(cfg.heartBeat), coapNet.WithErrors(cfg.errors), coapNet.WithOnReadTimeout(func() error {
		monitor.CheckInactivity(cc)
		return nil
	}))
	session := NewSession(cfg.ctx,
		l,
		addr,
		cfg.maxMessageSize,
		cfg.closeSocket,
	)
	cc = client.NewClientConn(session,
		observationTokenHandler, observatioRequests, cfg.transmissionNStart, cfg.transmissionAcknowledgeTimeout, cfg.transmissionMaxRetransmit,
		client.NewObservationHandler(observationTokenHandler, cfg.handler),
		cfg.blockwiseSZX,
		blockWise,
		cfg.goPool,
		cfg.errors,
		cfg.getMID,
		monitor,
	)

	go func() {
		err := cc.Run()
		if err != nil {
			cfg.errors(err)
		}
	}()

	return cc
}
