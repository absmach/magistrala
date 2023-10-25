package options

import (
	"context"
	"fmt"
	"net"
	"time"

	dtlsServer "github.com/plgd-dev/go-coap/v3/dtls/server"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/net/client"
	"github.com/plgd-dev/go-coap/v3/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/options/config"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	tcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	tcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
	udpClient "github.com/plgd-dev/go-coap/v3/udp/client"
	udpServer "github.com/plgd-dev/go-coap/v3/udp/server"
)

type ErrorFunc = config.ErrorFunc

type Handler interface {
	tcpClient.HandlerFunc | udpClient.HandlerFunc
}

// HandlerFuncOpt handler function option.
type HandlerFuncOpt[H Handler] struct {
	h H
}

func panicForInvalidHandlerFunc(t, exp any) {
	panic(fmt.Errorf("invalid HandlerFunc type %T, expected %T", t, exp))
}

func (o HandlerFuncOpt[H]) TCPServerApply(cfg *tcpServer.Config) {
	switch v := any(o.h).(type) {
	case tcpClient.HandlerFunc:
		cfg.Handler = v
	default:
		var exp tcpClient.HandlerFunc
		panicForInvalidHandlerFunc(v, exp)
	}
}

func (o HandlerFuncOpt[H]) TCPClientApply(cfg *tcpClient.Config) {
	switch v := any(o.h).(type) {
	case tcpClient.HandlerFunc:
		cfg.Handler = v
	default:
		var exp tcpClient.HandlerFunc
		panicForInvalidHandlerFunc(v, exp)
	}
}

func (o HandlerFuncOpt[H]) UDPServerApply(cfg *udpServer.Config) {
	switch v := any(o.h).(type) {
	case udpClient.HandlerFunc:
		cfg.Handler = v
	default:
		var exp udpClient.HandlerFunc
		panicForInvalidHandlerFunc(v, exp)
	}
}

func (o HandlerFuncOpt[H]) DTLSServerApply(cfg *dtlsServer.Config) {
	switch v := any(o.h).(type) {
	case udpClient.HandlerFunc:
		cfg.Handler = v
	default:
		var exp udpClient.HandlerFunc
		panicForInvalidHandlerFunc(v, exp)
	}
}

func (o HandlerFuncOpt[H]) UDPClientApply(cfg *udpClient.Config) {
	switch v := any(o.h).(type) {
	case udpClient.HandlerFunc:
		cfg.Handler = v
	default:
		var t udpClient.HandlerFunc
		panicForInvalidHandlerFunc(v, t)
	}
}

// WithHandlerFunc set handle for handling request's.
func WithHandlerFunc[H Handler](h H) HandlerFuncOpt[H] {
	return HandlerFuncOpt[H]{h: h}
}

// HandlerFuncOpt handler function option.
type MuxHandlerOpt struct {
	m mux.Handler
}

func (o MuxHandlerOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.Handler = mux.ToHandler[*tcpClient.Conn](o.m)
}

func (o MuxHandlerOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.Handler = mux.ToHandler[*tcpClient.Conn](o.m)
}

func (o MuxHandlerOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.Handler = mux.ToHandler[*udpClient.Conn](o.m)
}

func (o MuxHandlerOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.Handler = mux.ToHandler[*udpClient.Conn](o.m)
}

func (o MuxHandlerOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.Handler = mux.ToHandler[*udpClient.Conn](o.m)
}

// WithMux set's multiplexer for handle requests.
func WithMux(m mux.Handler) MuxHandlerOpt {
	return MuxHandlerOpt{
		m: m,
	}
}

// ContextOpt handler function option.
type ContextOpt struct {
	ctx context.Context
}

func (o ContextOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.Ctx = o.ctx
}

func (o ContextOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.Ctx = o.ctx
}

func (o ContextOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.Ctx = o.ctx
}

func (o ContextOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.Ctx = o.ctx
}

func (o ContextOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.Ctx = o.ctx
}

// WithContext set's parent context of server.
func WithContext(ctx context.Context) ContextOpt {
	return ContextOpt{ctx: ctx}
}

// MaxMessageSizeOpt handler function option.
type MaxMessageSizeOpt struct {
	maxMessageSize uint32
}

func (o MaxMessageSizeOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.MaxMessageSize = o.maxMessageSize
}

func (o MaxMessageSizeOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.MaxMessageSize = o.maxMessageSize
}

func (o MaxMessageSizeOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.MaxMessageSize = o.maxMessageSize
}

func (o MaxMessageSizeOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.MaxMessageSize = o.maxMessageSize
}

func (o MaxMessageSizeOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.MaxMessageSize = o.maxMessageSize
}

// WithMaxMessageSize limit size of processed message.
func WithMaxMessageSize(maxMessageSize uint32) MaxMessageSizeOpt {
	return MaxMessageSizeOpt{maxMessageSize: maxMessageSize}
}

// ErrorsOpt errors option.
type ErrorsOpt struct {
	errors ErrorFunc
}

func (o ErrorsOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.Errors = o.errors
}

func (o ErrorsOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.Errors = o.errors
}

func (o ErrorsOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.Errors = o.errors
}

func (o ErrorsOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.Errors = o.errors
}

func (o ErrorsOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.Errors = o.errors
}

// WithErrors set function for logging error.
func WithErrors(errors ErrorFunc) ErrorsOpt {
	return ErrorsOpt{errors: errors}
}

// ProcessReceivedMessageOpt gopool option.
type ProcessReceivedMessageOpt[C responsewriter.Client] struct {
	ProcessReceivedMessageFunc config.ProcessReceivedMessageFunc[C]
}

func panicForInvalidProcessReceivedMessageFunc(t, exp any) {
	panic(fmt.Errorf("invalid ProcessReceivedMessageFunc type %T, expected %T", t, exp))
}

func (o ProcessReceivedMessageOpt[C]) TCPServerApply(cfg *tcpServer.Config) {
	switch v := any(o.ProcessReceivedMessageFunc).(type) {
	case config.ProcessReceivedMessageFunc[*tcpClient.Conn]:
		cfg.ProcessReceivedMessage = v
	default:
		var t config.ProcessReceivedMessageFunc[*tcpClient.Conn]
		panicForInvalidProcessReceivedMessageFunc(v, t)
	}
}

func (o ProcessReceivedMessageOpt[C]) TCPClientApply(cfg *tcpClient.Config) {
	switch v := any(o.ProcessReceivedMessageFunc).(type) {
	case config.ProcessReceivedMessageFunc[*tcpClient.Conn]:
		cfg.ProcessReceivedMessage = v
	default:
		var t config.ProcessReceivedMessageFunc[*tcpClient.Conn]
		panicForInvalidProcessReceivedMessageFunc(v, t)
	}
}

func (o ProcessReceivedMessageOpt[C]) UDPServerApply(cfg *udpServer.Config) {
	switch v := any(o.ProcessReceivedMessageFunc).(type) {
	case config.ProcessReceivedMessageFunc[*udpClient.Conn]:
		cfg.ProcessReceivedMessage = v
	default:
		var t config.ProcessReceivedMessageFunc[*udpClient.Conn]
		panicForInvalidProcessReceivedMessageFunc(v, t)
	}
}

func (o ProcessReceivedMessageOpt[C]) DTLSServerApply(cfg *dtlsServer.Config) {
	switch v := any(o.ProcessReceivedMessageFunc).(type) {
	case config.ProcessReceivedMessageFunc[*udpClient.Conn]:
		cfg.ProcessReceivedMessage = v
	default:
		var t config.ProcessReceivedMessageFunc[*udpClient.Conn]
		panicForInvalidProcessReceivedMessageFunc(v, t)
	}
}

func (o ProcessReceivedMessageOpt[C]) UDPClientApply(cfg *udpClient.Config) {
	switch v := any(o.ProcessReceivedMessageFunc).(type) {
	case config.ProcessReceivedMessageFunc[*udpClient.Conn]:
		cfg.ProcessReceivedMessage = v
	default:
		var t config.ProcessReceivedMessageFunc[*udpClient.Conn]
		panicForInvalidProcessReceivedMessageFunc(v, t)
	}
}

func WithProcessReceivedMessageFunc[C responsewriter.Client](processReceivedMessageFunc config.ProcessReceivedMessageFunc[C]) ProcessReceivedMessageOpt[C] {
	return ProcessReceivedMessageOpt[C]{ProcessReceivedMessageFunc: processReceivedMessageFunc}
}

type (
	UDPOnInactive = func(cc *udpClient.Conn)
	TCPOnInactive = func(cc *tcpClient.Conn)
)

type OnInactiveFunc interface {
	UDPOnInactive | TCPOnInactive
}

func panicForInvalidOnInactiveFunc(t, exp any) {
	panic(fmt.Errorf("invalid OnInactiveFunc type %T, expected %T", t, exp))
}

// KeepAliveOpt keepalive option.
type KeepAliveOpt[C OnInactiveFunc] struct {
	timeout    time.Duration
	onInactive C
	maxRetries uint32
}

func (o KeepAliveOpt[C]) toTCPCreateInactivityMonitor(onInactive TCPOnInactive) func() tcpClient.InactivityMonitor {
	return func() tcpClient.InactivityMonitor {
		keepalive := inactivity.NewKeepAlive(o.maxRetries, onInactive, func(cc *tcpClient.Conn, receivePong func()) (func(), error) {
			return cc.AsyncPing(receivePong)
		})
		return inactivity.New(o.timeout/time.Duration(o.maxRetries+1), keepalive.OnInactive)
	}
}

func (o KeepAliveOpt[C]) toUDPCreateInactivityMonitor(onInactive UDPOnInactive) func() udpClient.InactivityMonitor {
	return func() udpClient.InactivityMonitor {
		keepalive := inactivity.NewKeepAlive(o.maxRetries, onInactive, func(cc *udpClient.Conn, receivePong func()) (func(), error) {
			return cc.AsyncPing(receivePong)
		})
		return inactivity.New(o.timeout/time.Duration(o.maxRetries+1), keepalive.OnInactive)
	}
}

func (o KeepAliveOpt[C]) TCPServerApply(cfg *tcpServer.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case TCPOnInactive:
		cfg.CreateInactivityMonitor = o.toTCPCreateInactivityMonitor(onInactive)
	default:
		var t TCPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o KeepAliveOpt[C]) TCPClientApply(cfg *tcpClient.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case TCPOnInactive:
		cfg.CreateInactivityMonitor = o.toTCPCreateInactivityMonitor(onInactive)
	default:
		var t TCPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o KeepAliveOpt[C]) UDPServerApply(cfg *udpServer.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case UDPOnInactive:
		cfg.CreateInactivityMonitor = o.toUDPCreateInactivityMonitor(onInactive)
	default:
		var t UDPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o KeepAliveOpt[C]) DTLSServerApply(cfg *dtlsServer.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case UDPOnInactive:
		cfg.CreateInactivityMonitor = o.toUDPCreateInactivityMonitor(onInactive)
	default:
		var t UDPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o KeepAliveOpt[C]) UDPClientApply(cfg *udpClient.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case UDPOnInactive:
		cfg.CreateInactivityMonitor = o.toUDPCreateInactivityMonitor(onInactive)
	default:
		var t UDPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

// WithKeepAlive monitoring's client connection's.
func WithKeepAlive[C OnInactiveFunc](maxRetries uint32, timeout time.Duration, onInactive C) KeepAliveOpt[C] {
	return KeepAliveOpt[C]{
		maxRetries: maxRetries,
		timeout:    timeout,
		onInactive: onInactive,
	}
}

// InactivityMonitorOpt notifies when a connection was inactive for a given duration.
type InactivityMonitorOpt[C OnInactiveFunc] struct {
	duration   time.Duration
	onInactive C
}

func (o InactivityMonitorOpt[C]) toTCPCreateInactivityMonitor(onInactive TCPOnInactive) func() tcpClient.InactivityMonitor {
	return func() tcpClient.InactivityMonitor {
		return inactivity.New(o.duration, onInactive)
	}
}

func (o InactivityMonitorOpt[C]) toUDPCreateInactivityMonitor(onInactive UDPOnInactive) func() udpClient.InactivityMonitor {
	return func() udpClient.InactivityMonitor {
		return inactivity.New(o.duration, onInactive)
	}
}

func (o InactivityMonitorOpt[C]) TCPServerApply(cfg *tcpServer.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case TCPOnInactive:
		cfg.CreateInactivityMonitor = o.toTCPCreateInactivityMonitor(onInactive)
	default:
		var t TCPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o InactivityMonitorOpt[C]) TCPClientApply(cfg *tcpClient.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case TCPOnInactive:
		cfg.CreateInactivityMonitor = o.toTCPCreateInactivityMonitor(onInactive)
	default:
		var t TCPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o InactivityMonitorOpt[C]) UDPServerApply(cfg *udpServer.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case UDPOnInactive:
		cfg.CreateInactivityMonitor = o.toUDPCreateInactivityMonitor(onInactive)
	default:
		var t UDPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o InactivityMonitorOpt[C]) DTLSServerApply(cfg *dtlsServer.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case UDPOnInactive:
		cfg.CreateInactivityMonitor = o.toUDPCreateInactivityMonitor(onInactive)
	default:
		var t UDPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

func (o InactivityMonitorOpt[C]) UDPClientApply(cfg *udpClient.Config) {
	switch onInactive := any(o.onInactive).(type) {
	case UDPOnInactive:
		cfg.CreateInactivityMonitor = o.toUDPCreateInactivityMonitor(onInactive)
	default:
		var t UDPOnInactive
		panicForInvalidOnInactiveFunc(onInactive, t)
	}
}

// WithInactivityMonitor set deadline's for read operations over client connection.
func WithInactivityMonitor[C OnInactiveFunc](duration time.Duration, onInactive C) InactivityMonitorOpt[C] {
	return InactivityMonitorOpt[C]{
		duration:   duration,
		onInactive: onInactive,
	}
}

// NetOpt network option.
type NetOpt struct {
	net string
}

func (o NetOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.Net = o.net
}

func (o NetOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.Net = o.net
}

// WithNetwork define's tcp version (udp4, udp6, tcp) for client.
func WithNetwork(net string) NetOpt {
	return NetOpt{net: net}
}

// PeriodicRunnerOpt function which is executed in every ticks
type PeriodicRunnerOpt struct {
	periodicRunner periodic.Func
}

func (o PeriodicRunnerOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.PeriodicRunner = o.periodicRunner
}

func (o PeriodicRunnerOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.PeriodicRunner = o.periodicRunner
}

func (o PeriodicRunnerOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.PeriodicRunner = o.periodicRunner
}

func (o PeriodicRunnerOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.PeriodicRunner = o.periodicRunner
}

func (o PeriodicRunnerOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.PeriodicRunner = o.periodicRunner
}

// WithPeriodicRunner set function which is executed in every ticks.
func WithPeriodicRunner(periodicRunner periodic.Func) PeriodicRunnerOpt {
	return PeriodicRunnerOpt{periodicRunner: periodicRunner}
}

// BlockwiseOpt network option.
type BlockwiseOpt struct {
	transferTimeout time.Duration
	enable          bool
	szx             blockwise.SZX
}

func (o BlockwiseOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.BlockwiseEnable = o.enable
	cfg.BlockwiseSZX = o.szx
	cfg.BlockwiseTransferTimeout = o.transferTimeout
}

func (o BlockwiseOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.BlockwiseEnable = o.enable
	cfg.BlockwiseSZX = o.szx
	cfg.BlockwiseTransferTimeout = o.transferTimeout
}

func (o BlockwiseOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.BlockwiseEnable = o.enable
	cfg.BlockwiseSZX = o.szx
	cfg.BlockwiseTransferTimeout = o.transferTimeout
}

func (o BlockwiseOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.BlockwiseEnable = o.enable
	cfg.BlockwiseSZX = o.szx
	cfg.BlockwiseTransferTimeout = o.transferTimeout
}

func (o BlockwiseOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.BlockwiseEnable = o.enable
	cfg.BlockwiseSZX = o.szx
	cfg.BlockwiseTransferTimeout = o.transferTimeout
}

// WithBlockwise configure's blockwise transfer.
func WithBlockwise(enable bool, szx blockwise.SZX, transferTimeout time.Duration) BlockwiseOpt {
	return BlockwiseOpt{
		enable:          enable,
		szx:             szx,
		transferTimeout: transferTimeout,
	}
}

type OnNewConnFunc interface {
	tcpServer.OnNewConnFunc | udpServer.OnNewConnFunc
}

// OnNewConnOpt network option.
type OnNewConnOpt[F OnNewConnFunc] struct {
	f F
}

func panicForInvalidOnNewConnFunc(t, exp any) {
	panic(fmt.Errorf("invalid OnNewConnFunc type %T, expected %T", t, exp))
}

func (o OnNewConnOpt[F]) UDPServerApply(cfg *udpServer.Config) {
	switch v := any(o.f).(type) {
	case udpServer.OnNewConnFunc:
		cfg.OnNewConn = v
	default:
		var exp udpServer.OnNewConnFunc
		panicForInvalidOnNewConnFunc(v, exp)
	}
}

func (o OnNewConnOpt[F]) DTLSServerApply(cfg *dtlsServer.Config) {
	switch v := any(o.f).(type) {
	case udpServer.OnNewConnFunc:
		cfg.OnNewConn = v
	default:
		var exp udpServer.OnNewConnFunc
		panicForInvalidOnNewConnFunc(v, exp)
	}
}

func (o OnNewConnOpt[F]) TCPServerApply(cfg *tcpServer.Config) {
	switch v := any(o.f).(type) {
	case tcpServer.OnNewConnFunc:
		cfg.OnNewConn = v
	default:
		var exp tcpServer.OnNewConnFunc
		panicForInvalidOnNewConnFunc(v, exp)
	}
}

// WithOnNewConn server's notify about new client connection.
func WithOnNewConn[F OnNewConnFunc](onNewConn F) OnNewConnOpt[F] {
	return OnNewConnOpt[F]{
		f: onNewConn,
	}
}

// CloseSocketOpt close socket option.
type CloseSocketOpt struct{}

func (o CloseSocketOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.CloseSocket = true
}

func (o CloseSocketOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.CloseSocket = true
}

// WithCloseSocket closes socket at the close connection.
func WithCloseSocket() CloseSocketOpt {
	return CloseSocketOpt{}
}

// DialerOpt dialer option.
type DialerOpt struct {
	dialer *net.Dialer
}

func (o DialerOpt) UDPClientApply(cfg *udpClient.Config) {
	if o.dialer != nil {
		cfg.Dialer = o.dialer
	}
}

func (o DialerOpt) TCPClientApply(cfg *tcpClient.Config) {
	if o.dialer != nil {
		cfg.Dialer = o.dialer
	}
}

// WithDialer set dialer for dial.
func WithDialer(dialer *net.Dialer) DialerOpt {
	return DialerOpt{
		dialer: dialer,
	}
}

// ConnectionCacheOpt network option.
type MessagePoolOpt struct {
	messagePool *pool.Pool
}

func (o MessagePoolOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.MessagePool = o.messagePool
}

func (o MessagePoolOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.MessagePool = o.messagePool
}

func (o MessagePoolOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.MessagePool = o.messagePool
}

func (o MessagePoolOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.MessagePool = o.messagePool
}

func (o MessagePoolOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.MessagePool = o.messagePool
}

// WithMessagePool configure's message pool for acquire/releasing coap messages
func WithMessagePool(messagePool *pool.Pool) MessagePoolOpt {
	return MessagePoolOpt{
		messagePool: messagePool,
	}
}

// GetTokenOpt token option.
type GetTokenOpt struct {
	getToken client.GetTokenFunc
}

func (o GetTokenOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.GetToken = o.getToken
}

func (o GetTokenOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.GetToken = o.getToken
}

func (o GetTokenOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.GetToken = o.getToken
}

func (o GetTokenOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.GetToken = o.getToken
}

func (o GetTokenOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.GetToken = o.getToken
}

// WithGetToken set function for generating tokens.
func WithGetToken(getToken client.GetTokenFunc) GetTokenOpt {
	return GetTokenOpt{getToken: getToken}
}

// LimitClientParallelRequestOpt limit's number of parallel requests from client.
type LimitClientParallelRequestOpt struct {
	limitClientParallelRequests int64
}

func (o LimitClientParallelRequestOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.LimitClientParallelRequests = o.limitClientParallelRequests
}

func (o LimitClientParallelRequestOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.LimitClientParallelRequests = o.limitClientParallelRequests
}

func (o LimitClientParallelRequestOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.LimitClientParallelRequests = o.limitClientParallelRequests
}

func (o LimitClientParallelRequestOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.LimitClientParallelRequests = o.limitClientParallelRequests
}

func (o LimitClientParallelRequestOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.LimitClientParallelRequests = o.limitClientParallelRequests
}

// WithLimitClientParallelRequestOpt limits number of parallel requests from client. (default: 1)
func WithLimitClientParallelRequest(limitClientParallelRequests int64) LimitClientParallelRequestOpt {
	return LimitClientParallelRequestOpt{limitClientParallelRequests: limitClientParallelRequests}
}

// LimitClientEndpointParallelRequestOpt limit's number of parallel requests to endpoint by client.
type LimitClientEndpointParallelRequestOpt struct {
	limitClientEndpointParallelRequests int64
}

func (o LimitClientEndpointParallelRequestOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.LimitClientEndpointParallelRequests = o.limitClientEndpointParallelRequests
}

func (o LimitClientEndpointParallelRequestOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.LimitClientEndpointParallelRequests = o.limitClientEndpointParallelRequests
}

func (o LimitClientEndpointParallelRequestOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.LimitClientEndpointParallelRequests = o.limitClientEndpointParallelRequests
}

func (o LimitClientEndpointParallelRequestOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.LimitClientEndpointParallelRequests = o.limitClientEndpointParallelRequests
}

func (o LimitClientEndpointParallelRequestOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.LimitClientEndpointParallelRequests = o.limitClientEndpointParallelRequests
}

// WithLimitClientEndpointParallelRequest limits number of parallel requests to endpoint from client. (default: 1)
func WithLimitClientEndpointParallelRequest(limitClientEndpointParallelRequests int64) LimitClientEndpointParallelRequestOpt {
	return LimitClientEndpointParallelRequestOpt{limitClientEndpointParallelRequests: limitClientEndpointParallelRequests}
}

// ReceivedMessageQueueSizeOpt limit's message queue size for received messages.
type ReceivedMessageQueueSizeOpt struct {
	receivedMessageQueueSize int
}

func (o ReceivedMessageQueueSizeOpt) TCPServerApply(cfg *tcpServer.Config) {
	cfg.ReceivedMessageQueueSize = o.receivedMessageQueueSize
}

func (o ReceivedMessageQueueSizeOpt) TCPClientApply(cfg *tcpClient.Config) {
	cfg.ReceivedMessageQueueSize = o.receivedMessageQueueSize
}

func (o ReceivedMessageQueueSizeOpt) UDPServerApply(cfg *udpServer.Config) {
	cfg.ReceivedMessageQueueSize = o.receivedMessageQueueSize
}

func (o ReceivedMessageQueueSizeOpt) DTLSServerApply(cfg *dtlsServer.Config) {
	cfg.ReceivedMessageQueueSize = o.receivedMessageQueueSize
}

func (o ReceivedMessageQueueSizeOpt) UDPClientApply(cfg *udpClient.Config) {
	cfg.ReceivedMessageQueueSize = o.receivedMessageQueueSize
}

// WithReceivedMessageQueueSize limit's message queue size for received messages. (default: 16)
func WithReceivedMessageQueueSize(receivedMessageQueueSize int) ReceivedMessageQueueSizeOpt {
	return ReceivedMessageQueueSizeOpt{receivedMessageQueueSize: receivedMessageQueueSize}
}
