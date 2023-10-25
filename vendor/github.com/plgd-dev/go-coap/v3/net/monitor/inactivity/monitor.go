package inactivity

import (
	"context"
	"sync/atomic"
	"time"
)

type OnInactiveFunc[C Conn] func(cc C)

type Conn = interface {
	Context() context.Context
	Close() error
}

type Monitor[C Conn] struct {
	lastActivity atomic.Value
	duration     time.Duration
	onInactive   OnInactiveFunc[C]
}

func (m *Monitor[C]) Notify() {
	m.lastActivity.Store(time.Now())
}

func (m *Monitor[C]) LastActivity() time.Time {
	if t, ok := m.lastActivity.Load().(time.Time); ok {
		return t
	}
	return time.Time{}
}

func CloseConn(cc Conn) {
	// call cc.Close() directly to check and handle error if necessary
	_ = cc.Close()
}

func New[C Conn](duration time.Duration, onInactive OnInactiveFunc[C]) *Monitor[C] {
	m := &Monitor[C]{
		duration:   duration,
		onInactive: onInactive,
	}
	m.Notify()
	return m
}

func (m *Monitor[C]) CheckInactivity(now time.Time, cc C) {
	if m.onInactive == nil || m.duration == time.Duration(0) {
		return
	}
	if now.After(m.LastActivity().Add(m.duration)) {
		m.onInactive(cc)
	}
}

type NilMonitor[C Conn] struct{}

func (m *NilMonitor[C]) CheckInactivity(time.Time, C) {
	// do nothing
}

func (m *NilMonitor[C]) Notify() {
	// do nothing
}

func NewNilMonitor[C Conn]() *NilMonitor[C] {
	return &NilMonitor[C]{}
}
