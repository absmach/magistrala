package pool

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/atomic"
)

type Pool struct {
	// This field needs to be the first in the struct to ensure proper word alignment on 32-bit platforms.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	currentMessagesInPool atomic.Int64
	messagePool           sync.Pool
	maxNumMessages        uint32
	maxMessageBufferSize  uint16
}

func New(maxNumMessages uint32, maxMessageBufferSize uint16) *Pool {
	return &Pool{
		maxNumMessages:       maxNumMessages,
		maxMessageBufferSize: maxMessageBufferSize,
	}
}

// AcquireMessage returns an empty Message instance from Message pool.
//
// The returned Message instance may be passed to ReleaseMessage when it is
// no longer needed. This allows Message recycling, reduces GC pressure
// and usually improves performance.
func (p *Pool) AcquireMessage(ctx context.Context) *Message {
	v := p.messagePool.Get()
	if v == nil {
		return NewMessage(ctx)
	}
	r, ok := v.(*Message)
	if !ok {
		panic(fmt.Errorf("invalid message type(%T) for pool", v))
	}
	p.currentMessagesInPool.Dec()
	r.ctx = ctx
	return r
}

// ReleaseMessage returns req acquired via AcquireMessage to Message pool.
//
// It is forbidden accessing req and/or its' members after returning
// it to Message pool.
func (p *Pool) ReleaseMessage(req *Message) {
	for {
		v := p.currentMessagesInPool.Load()
		if v >= int64(p.maxNumMessages) {
			return
		}
		next := v + 1
		if p.currentMessagesInPool.CompareAndSwap(v, next) {
			break
		}
	}
	req.Reset()
	req.ctx = nil
	p.messagePool.Put(req)
}
