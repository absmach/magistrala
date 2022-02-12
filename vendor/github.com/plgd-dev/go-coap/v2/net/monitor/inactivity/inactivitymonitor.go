package inactivity

import (
	"context"
	"sync/atomic"
	"time"
)

type Monitor = interface {
	CheckInactivity(now time.Time, cc ClientConn)
	Notify()
}

type OnInactiveFunc = func(cc ClientConn)

type ClientConn = interface {
	Context() context.Context
	Close() error
}

type inactivityMonitor struct {
	lastActivity atomic.Value
	duration     time.Duration
	onInactive   OnInactiveFunc
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
	// call cc.Close() directly to check and handle error if necessary
	_ = cc.Close()
}

func NewInactivityMonitor(duration time.Duration, onInactive OnInactiveFunc) Monitor {
	m := &inactivityMonitor{
		duration:   duration,
		onInactive: onInactive,
	}
	m.Notify()
	return m
}

func (m *inactivityMonitor) CheckInactivity(now time.Time, cc ClientConn) {
	if m.onInactive == nil || m.duration == time.Duration(0) {
		return
	}
	if now.After(m.LastActivity().Add(m.duration)) {
		m.onInactive(cc)
	}
}

type nilMonitor struct {
}

func (m *nilMonitor) CheckInactivity(now time.Time, cc ClientConn) {
}

func (m *nilMonitor) Notify() {
}

func NewNilMonitor() Monitor {
	return &nilMonitor{}
}
