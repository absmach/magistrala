package inactivity

import (
	"sync/atomic"
)

type KeepAlive struct {
	pongToken  uint64
	onInactive OnInactiveFunc

	sendPing   func(cc ClientConn, receivePong func()) (func(), error)
	cancelPing func()
	numFails   uint32

	maxRetries uint32
}

func NewKeepAlive(maxRetries uint32, onInactive OnInactiveFunc, sendPing func(cc ClientConn, receivePong func()) (func(), error)) *KeepAlive {
	return &KeepAlive{
		maxRetries: maxRetries,
		sendPing:   sendPing,
		onInactive: onInactive,
	}
}

func (m *KeepAlive) OnInactive(cc ClientConn) {
	v := m.incrementFails()
	if m.cancelPing != nil {
		m.cancelPing()
		m.cancelPing = nil
	}
	if v > m.maxRetries {
		m.onInactive(cc)
		return
	}
	pongToken := atomic.AddUint64(&m.pongToken, 1)
	cancel, err := m.sendPing(cc, func() {
		if atomic.LoadUint64(&m.pongToken) == pongToken {
			m.resetFails()
		}
	})
	if err != nil {
		return
	}
	m.cancelPing = cancel
}

func (m *KeepAlive) incrementFails() uint32 {
	return atomic.AddUint32(&m.numFails, 1)
}

func (m *KeepAlive) resetFails() {
	atomic.StoreUint32(&m.numFails, 0)
}
