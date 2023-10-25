package client

import (
	"sync"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"go.uber.org/atomic"
)

type ReceivedMessageReaderClient interface {
	Done() <-chan struct{}
	ProcessReceivedMessage(req *pool.Message)
}

type ReceivedMessageReader[C ReceivedMessageReaderClient] struct {
	queue chan *pool.Message
	cc    C

	private struct {
		mutex           sync.Mutex
		loopDone        chan struct{}
		readingMessages *atomic.Bool
	}
}

// NewReceivedMessageReader creates a new ReceivedMessageReader[C] instance.
func NewReceivedMessageReader[C ReceivedMessageReaderClient](cc C, queueSize int) *ReceivedMessageReader[C] {
	r := ReceivedMessageReader[C]{
		queue: make(chan *pool.Message, queueSize),
		cc:    cc,
		private: struct {
			mutex           sync.Mutex
			loopDone        chan struct{}
			readingMessages *atomic.Bool
		}{
			loopDone:        make(chan struct{}),
			readingMessages: atomic.NewBool(true),
		},
	}

	go r.loop(r.private.loopDone, r.private.readingMessages)
	return &r
}

// C returns the channel to push received messages to.
func (r *ReceivedMessageReader[C]) C() chan<- *pool.Message {
	return r.queue
}

// The loop function continuously listens to messages. IT can be replaced with a new one by calling the TryToReplaceLoop function,
// ensuring that only one loop is reading from the queue at a time.
// The loopDone channel is used to signal when the loop should be closed.
// The readingMessages variable is used to indicate if the loop is currently reading from the queue.
// When the loop is not reading from the queue, it sets readingMessages to false, and when it starts reading again, it sets it to true.
// If the client is closed, the loop also closes.
func (r *ReceivedMessageReader[C]) loop(loopDone chan struct{}, readingMessages *atomic.Bool) {
	for {
		select {
		// if the loop is replaced, the old loop will be closed
		case <-loopDone:
			return
		// process received message until the queue is empty
		case req := <-r.queue:
			// This signalizes that the loop is not reading messages.
			readingMessages.Store(false)
			r.cc.ProcessReceivedMessage(req)
			// This signalizes that the loop is reading messages. We call mutex because we want to ensure that TryToReplaceLoop has ended and
			// loopDone is closed if it was replaced.
			r.private.mutex.Lock()
			readingMessages.Store(true)
			r.private.mutex.Unlock()
		// if the client is closed, the loop will be closed
		case <-r.cc.Done():
			return
		}
	}
}

// TryToReplaceLoop function attempts to replace the loop with a new one,
// but only if the loop is not currently reading messages. If the loop is reading messages,
// the function returns immediately. If the loop is not reading messages, the current loop is closed,
// and new loopDone and readingMessages channels and variables are created.
func (r *ReceivedMessageReader[C]) TryToReplaceLoop() {
	r.private.mutex.Lock()
	if r.private.readingMessages.Load() {
		r.private.mutex.Unlock()
		return
	}
	defer r.private.mutex.Unlock()
	close(r.private.loopDone)
	loopDone := make(chan struct{})
	readingMessages := atomic.NewBool(true)
	r.private.loopDone = loopDone
	r.private.readingMessages = readingMessages
	go r.loop(loopDone, readingMessages)
}
