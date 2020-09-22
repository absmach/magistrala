package dtls

import (
	"context"
	"time"

	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/keepalive"
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
	maxMessageSize int
}

func (o MaxMessageSizeOpt) apply(opts *serverOptions) {
	opts.maxMessageSize = o.maxMessageSize
}

func (o MaxMessageSizeOpt) applyDial(opts *dialOptions) {
	opts.maxMessageSize = o.maxMessageSize
}

// WithMaxMessageSize limit size of processed message.
func WithMaxMessageSize(maxMessageSize int) MaxMessageSizeOpt {
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
	keepalive *keepalive.KeepAlive
}

func (o KeepAliveOpt) apply(opts *serverOptions) {
	opts.keepalive = o.keepalive
}

func (o KeepAliveOpt) applyDial(opts *dialOptions) {
	opts.keepalive = o.keepalive
}

// WithKeepAlive monitoring's client connection's. nil means disable keepalive.
func WithKeepAlive(keepalive *keepalive.KeepAlive) KeepAliveOpt {
	return KeepAliveOpt{keepalive: keepalive}
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

// HeartBeatOpt heatbeat of read/write operations over connection.
type HeartBeatOpt struct {
	heartbeat time.Duration
}

func (o HeartBeatOpt) apply(opts *serverOptions) {
	opts.heartBeat = o.heartbeat
}

func (o HeartBeatOpt) applyDial(opts *dialOptions) {
	opts.heartBeat = o.heartbeat
}

// WithHeartBeat set deadline's for read/write operations over client connection.
func WithHeartBeat(heartbeat time.Duration) HeartBeatOpt {
	return HeartBeatOpt{heartbeat: heartbeat}
}

// BlockwiseOpt network option.
type BlockwiseOpt struct {
	enable          bool
	szx             blockwise.SZX
	transferTimeout time.Duration
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
	transmissionMaxRetransmit      int
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
	transmissionMaxRetransmit int) TransmissionOpt {
	return TransmissionOpt{
		transmissionNStart:             transmissionNStart,
		transmissionAcknowledgeTimeout: transmissionAcknowledgeTimeout,
		transmissionMaxRetransmit:      transmissionMaxRetransmit,
	}
}

// GetMIDOpt get message ID option.
type GetMIDOpt struct {
	getMID GetMIDFunc
}

func (o GetMIDOpt) apply(opts *serverOptions) {
	opts.getMID = o.getMID
}

func (o GetMIDOpt) applyDial(opts *dialOptions) {
	opts.getMID = o.getMID
}

// WithGetMID allows to set own getMID function to server/client.
func WithGetMID(getMID GetMIDFunc) GetMIDOpt {
	return GetMIDOpt{getMID: getMID}
}
