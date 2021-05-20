package inactivity

import (
	"context"
	"sync/atomic"
	"time"
)

type Monitor = interface {
	CheckInactivity(cc ClientConn)
	Notify()
}

type OnInactiveFunc = func(cc ClientConn)

type ClientConn = interface {
	Context() context.Context
	Close() error
}

type inactivityMonitor struct {
	duration   time.Duration
	onInactive OnInactiveFunc
	// lastActivity stores time.Time
	lastActivity atomic.Value
}

func (m *inactivityMonitor) Notify() {
	m.lastActivity.Store(time.Now())
}

func (m *inactivityMonitor) LastActivity() time.Time {
	if t, ok := m.lastActivity.Load().(time.Time); ok {
		return t
	}
	return time.Time{}
}

func CloseClientConn(cc ClientConn) {
	cc.Close()
}

func NewInactivityMonitor(duration time.Duration, onInactive OnInactiveFunc) Monitor {
	m := &inactivityMonitor{
		duration:   duration,
		onInactive: onInactive,
	}
	m.Notify()
	return m
}

func (m *inactivityMonitor) CheckInactivity(cc ClientConn) {
	if m.onInactive == nil || m.duration == time.Duration(0) {
		return
	}
	if time.Until(m.LastActivity().Add(m.duration)) <= 0 {
		m.onInactive(cc)
	}
}

type nilMonitor struct {
}

func (m *nilMonitor) CheckInactivity(cc ClientConn) {
}

func (m *nilMonitor) Notify() {
}

func NewNilMonitor() Monitor {
	return &nilMonitor{}
}
