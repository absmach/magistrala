package udp

import (
	"context"
	"net"
	"time"

	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/go-coap/v2/udp/client"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

// HandlerFuncOpt handler function option.
type HandlerFuncOpt struct {
	h HandlerFunc
}

func (o HandlerFuncOpt) apply(opts *serverOptions) {
	opts.handler = o.h
}

func (o HandlerFuncOpt) applyDial(opts *dialOptions) {
	opts.handler = o.h
}

// WithHandlerFunc set handle for handling request's.
func WithHandlerFunc(h HandlerFunc) HandlerFuncOpt {
	return HandlerFuncOpt{h: h}
}

// ContextOpt handler function option.
type ContextOpt struct {
	ctx context.Context
}

func (o ContextOpt) apply(opts *serverOptions) {
	opts.ctx = o.ctx
}

func (o ContextOpt) applyDial(opts *dialOptions) {
	opts.ctx = o.ctx
}

// WithContext set's parent context of server.
func WithContext(ctx context.Context) ContextOpt {
	return ContextOpt{ctx: ctx}
}

// MaxMessageSizeOpt handler function option.
type MaxMessageSizeOpt struct {
	maxMessageSize uint32
}

func (o MaxMessageSizeOpt) apply(opts *serverOptions) {
	opts.maxMessageSize = o.maxMessageSize
}

func (o MaxMessageSizeOpt) applyDial(opts *dialOptions) {
	opts.maxMessageSize = o.maxMessageSize
}

// WithMaxMessageSize limit size of processed message.
func WithMaxMessageSize(maxMessageSize uint32) MaxMessageSizeOpt {
	return MaxMessageSizeOpt{maxMessageSize: maxMessageSize}
}

// ErrorsOpt errors option.
type ErrorsOpt struct {
	errors ErrorFunc
}

func (o ErrorsOpt) apply(opts *serverOptions) {
	opts.errors = o.errors
}

func (o ErrorsOpt) applyDial(opts *dialOptions) {
	opts.errors = o.errors
}

// WithErrors set function for logging error.
func WithErrors(errors ErrorFunc) ErrorsOpt {
	return ErrorsOpt{errors: errors}
}

// GoPoolOpt gopool option.
type GoPoolOpt struct {
	goPool GoPoolFunc
}

func (o GoPoolOpt) apply(opts *serverOptions) {
	opts.goPool = o.goPool
}

func (o GoPoolOpt) applyDial(opts *dialOptions) {
	opts.goPool = o.goPool
}

// WithGoPool sets function for managing spawning go routines
// for handling incoming request's.
// Eg: https://github.com/panjf2000/ants.
func WithGoPool(goPool GoPoolFunc) GoPoolOpt {
	return GoPoolOpt{goPool: goPool}
}

// KeepAliveOpt keepalive option.
type KeepAliveOpt struct {
	timeout    time.Duration
	onInactive inactivity.OnInactiveFunc
	maxRetries uint32
}

func (o KeepAliveOpt) apply(opts *serverOptions) {
	opts.createInactivityMonitor = func() inactivity.Monitor {
		keepalive := inactivity.NewKeepAlive(o.maxRetries, o.onInactive, func(cc inactivity.ClientConn, receivePong func()) (func(), error) {
			return cc.(*client.ClientConn).AsyncPing(receivePong)
		})
		return inactivity.NewInactivityMonitor(o.timeout/time.Duration(o.maxRetries+1), keepalive.OnInactive)
	}
}

func (o KeepAliveOpt) applyDial(opts *dialOptions) {
	opts.createInactivityMonitor = func() inactivity.Monitor {
		keepalive := inactivity.NewKeepAlive(o.maxRetries, o.onInactive, func(cc inactivity.ClientConn, receivePong func()) (func(), error) {
			return cc.(*client.ClientConn).AsyncPing(receivePong)
		})
		return inactivity.NewInactivityMonitor(o.timeout/time.Duration(o.maxRetries+1), keepalive.OnInactive)
	}
}

// WithKeepAlive monitoring's client connection's.
func WithKeepAlive(maxRetries uint32, timeout time.Duration, onInactive inactivity.OnInactiveFunc) KeepAliveOpt {
	return KeepAliveOpt{
		maxRetries: maxRetries,
		timeout:    timeout,
		onInactive: onInactive,
	}
}

// InactivityMonitorOpt notifies when a connection was inactive for a given duration.
type InactivityMonitorOpt struct {
	duration   time.Duration
	onInactive inactivity.OnInactiveFunc
}

func (o InactivityMonitorOpt) apply(opts *serverOptions) {
	opts.createInactivityMonitor = func() inactivity.Monitor {
		return inactivity.NewInactivityMonitor(o.duration, o.onInactive)
	}
}

func (o InactivityMonitorOpt) applyDial(opts *dialOptions) {
	opts.createInactivityMonitor = func() inactivity.Monitor {
		return inactivity.NewInactivityMonitor(o.duration, o.onInactive)
	}
}

// WithInactivityMonitor set deadline's for read operations over client connection.
func WithInactivityMonitor(duration time.Duration, onInactive inactivity.OnInactiveFunc) InactivityMonitorOpt {
	return InactivityMonitorOpt{
		duration:   duration,
		onInactive: onInactive,
	}
}

// NetOpt network option.
type NetOpt struct {
	net string
}

func (o NetOpt) applyDial(opts *dialOptions) {
	opts.net = o.net
}

// WithNetwork define's udp version (udp4, udp6, udp) for client.
func WithNetwork(net string) NetOpt {
	return NetOpt{net: net}
}

// PeriodicRunnerOpt function which is executed in every ticks
type PeriodicRunnerOpt struct {
	periodicRunner periodic.Func
}

func (o PeriodicRunnerOpt) applyDial(opts *dialOptions) {
	opts.periodicRunner = o.periodicRunner
}

func (o PeriodicRunnerOpt) apply(opts *serverOptions) {
	opts.periodicRunner = o.periodicRunner
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

func (o BlockwiseOpt) apply(opts *serverOptions) {
	opts.blockwiseEnable = o.enable
	opts.blockwiseSZX = o.szx
	opts.blockwiseTransferTimeout = o.transferTimeout
}

func (o BlockwiseOpt) applyDial(opts *dialOptions) {
	opts.blockwiseEnable = o.enable
	opts.blockwiseSZX = o.szx
	opts.blockwiseTransferTimeout = o.transferTimeout
}

// WithBlockwise configure's blockwise transfer.
func WithBlockwise(enable bool, szx blockwise.SZX, transferTimeout time.Duration) BlockwiseOpt {
	return BlockwiseOpt{
		enable:          enable,
		szx:             szx,
		transferTimeout: transferTimeout,
	}
}

// OnNewClientConnOpt network option.
type OnNewClientConnOpt struct {
	onNewClientConn OnNewClientConnFunc
}

func (o OnNewClientConnOpt) apply(opts *serverOptions) {
	opts.onNewClientConn = o.onNewClientConn
}

// WithOnNewClientConn server's notify about new client connection.
func WithOnNewClientConn(onNewClientConn OnNewClientConnFunc) OnNewClientConnOpt {
	return OnNewClientConnOpt{
		onNewClientConn: onNewClientConn,
	}
}

// TransmissionOpt transmission options.
type TransmissionOpt struct {
	transmissionNStart             time.Duration
	transmissionAcknowledgeTimeout time.Duration
	transmissionMaxRetransmit      uint32
}

func (o TransmissionOpt) apply(opts *serverOptions) {
	opts.transmissionNStart = o.transmissionNStart
	opts.transmissionAcknowledgeTimeout = o.transmissionAcknowledgeTimeout
	opts.transmissionMaxRetransmit = o.transmissionMaxRetransmit
}

func (o TransmissionOpt) applyDial(opts *dialOptions) {
	opts.transmissionNStart = o.transmissionNStart
	opts.transmissionAcknowledgeTimeout = o.transmissionAcknowledgeTimeout
	opts.transmissionMaxRetransmit = o.transmissionMaxRetransmit
}

// WithTransmission set options for (re)transmission for Confirmable message-s.
func WithTransmission(transmissionNStart time.Duration,
	transmissionAcknowledgeTimeout time.Duration,
	transmissionMaxRetransmit uint32) TransmissionOpt {
	return TransmissionOpt{
		transmissionNStart:             transmissionNStart,
		transmissionAcknowledgeTimeout: transmissionAcknowledgeTimeout,
		transmissionMaxRetransmit:      transmissionMaxRetransmit,
	}
}

// CloseSocketOpt close socket option.
type CloseSocketOpt struct {
}

func (o CloseSocketOpt) applyDial(opts *dialOptions) {
	opts.closeSocket = true
}

// WithCloseSocket closes socket at the close connection.
func WithCloseSocket() CloseSocketOpt {
	return CloseSocketOpt{}
}

// DialerOpt dialer option.
type DialerOpt struct {
	dialer *net.Dialer
}

func (o DialerOpt) applyDial(opts *dialOptions) {
	if o.dialer != nil {
		opts.dialer = o.dialer
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

func (o MessagePoolOpt) apply(opts *serverOptions) {
	opts.messagePool = o.messagePool
}

func (o MessagePoolOpt) applyDial(opts *dialOptions) {
	opts.messagePool = o.messagePool
}

// WithMessagePool configure's message pool for acquire/releasing coap messages
func WithMessagePool(messagePool *pool.Pool) MessagePoolOpt {
	return MessagePoolOpt{
		messagePool: messagePool,
	}
}
