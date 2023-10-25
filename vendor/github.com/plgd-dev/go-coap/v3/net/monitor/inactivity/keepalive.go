package inactivity

import (
	"unsafe"

	"go.uber.org/atomic"
)

type cancelPingFunc func()

type KeepAlive[C Conn] struct {
	pongToken  atomic.Uint64
	onInactive OnInactiveFunc[C]

	sendPing   func(cc C, receivePong func()) (func(), error)
	cancelPing atomic.UnsafePointer
	numFails   atomic.Uint32

	maxRetries uint32
}

func NewKeepAlive[C Conn](maxRetries uint32, onInactive OnInactiveFunc[C], sendPing func(cc C, receivePong func()) (func(), error)) *KeepAlive[C] {
	return &KeepAlive[C]{
		maxRetries: maxRetries,
		sendPing:   sendPing,
		onInactive: onInactive,
	}
}

func (m *KeepAlive[C]) checkCancelPing() {
	cancelPingPtr := m.cancelPing.Swap(nil)
	if cancelPingPtr != nil {
		cancelPing := *(*cancelPingFunc)(cancelPingPtr)
		cancelPing()
	}
}

func (m *KeepAlive[C]) OnInactive(cc C) {
	v := m.incrementFails()
	m.checkCancelPing()
	if v > m.maxRetries {
		m.onInactive(cc)
		return
	}
	pongToken := m.pongToken.Add(1)
	cancel, err := m.sendPing(cc, func() {
		if m.pongToken.Load() == pongToken {
			m.resetFails()
		}
	})
	if err != nil {
		return
	}
	m.cancelPing.Store(unsafe.Pointer(&cancel))
}

func (m *KeepAlive[C]) incrementFails() uint32 {
	return m.numFails.Add(1)
}

func (m *KeepAlive[C]) resetFails() {
	m.numFails.Store(0)
}
