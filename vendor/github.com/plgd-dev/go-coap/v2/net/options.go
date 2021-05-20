package net

import "time"

// A UDPOption sets options such as heartBeat, errors parameters, etc.
type UDPOption interface {
	applyUDP(*udpConnOptions)
}

type HeartBeatOpt struct {
	heartBeat time.Duration
}

func (h HeartBeatOpt) applyUDP(o *udpConnOptions) {
	o.heartBeat = h.heartBeat
}

func (h HeartBeatOpt) applyConn(o *connOptions) {
	o.heartBeat = h.heartBeat
}

func (h HeartBeatOpt) applyTCPListener(o *tcpListenerOptions) {
	o.heartBeat = h.heartBeat
}

func (h HeartBeatOpt) applyTLSListener(o *tlsListenerOptions) {
	o.heartBeat = h.heartBeat
}

func (h HeartBeatOpt) applyDTLSListener(o *dtlsListenerOptions) {
	o.heartBeat = h.heartBeat
}

func WithHeartBeat(v time.Duration) HeartBeatOpt {
	return HeartBeatOpt{
		heartBeat: v,
	}
}

type ErrorsOpt struct {
	errors func(err error)
}

func (h ErrorsOpt) applyUDP(o *udpConnOptions) {
	o.errors = h.errors
}

func WithErrors(v func(err error)) ErrorsOpt {
	return ErrorsOpt{
		errors: v,
	}
}

type OnTimeoutOpt struct {
	onTimeout func() error
}

func WithOnTimeout(onTimeout func() error) OnTimeoutOpt {
	return OnTimeoutOpt{
		onTimeout: onTimeout,
	}
}

func (h OnTimeoutOpt) applyConn(o *connOptions) {
	o.onReadTimeout = h.onTimeout
	o.onWriteTimeout = h.onTimeout
}

func (h OnTimeoutOpt) applyUDP(o *udpConnOptions) {
	o.onReadTimeout = h.onTimeout
	o.onWriteTimeout = h.onTimeout
}

func (h OnTimeoutOpt) applyTCPListener(o *tcpListenerOptions) {
	o.onTimeout = h.onTimeout
}

func (h OnTimeoutOpt) applyTLSListener(o *tlsListenerOptions) {
	o.onTimeout = h.onTimeout
}

func (h OnTimeoutOpt) applyDTLSListener(o *dtlsListenerOptions) {
	o.onTimeout = h.onTimeout
}

type OnReadTimeoutOpt struct {
	onReadTimeout func() error
}

func WithOnReadTimeout(onReadTimeout func() error) OnReadTimeoutOpt {
	return OnReadTimeoutOpt{
		onReadTimeout: onReadTimeout,
	}
}

func (h OnReadTimeoutOpt) applyConn(o *connOptions) {
	o.onReadTimeout = h.onReadTimeout
}

func (h OnReadTimeoutOpt) applyUDP(o *udpConnOptions) {
	o.onReadTimeout = h.onReadTimeout
}

type OnWriteTimeoutOpt struct {
	onWriteTimeout func() error
}

func WithOnWriteTimeout(onWriteTimeout func() error) OnWriteTimeoutOpt {
	return OnWriteTimeoutOpt{
		onWriteTimeout: onWriteTimeout,
	}
}

func (h OnWriteTimeoutOpt) applyConn(o *connOptions) {
	o.onWriteTimeout = h.onWriteTimeout
}

func (h OnWriteTimeoutOpt) applyUDP(o *udpConnOptions) {
	o.onWriteTimeout = h.onWriteTimeout
}
