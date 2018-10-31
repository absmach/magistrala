//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package coap

import (
	"sync"

	"github.com/mainflux/mainflux"
)

// Observer is used to handle CoAP subscription.
type Observer struct {
	// Expired flag is used to mark that ticker sent a
	// CON message, but response is not received yet.
	// The flag changes its value once ACK message is
	// received from the client. If Expired is true
	// when ticker is triggered, Observer should be canceled
	// and removed from the Service map.
	expired bool

	// Message ID for notification messages.
	msgID uint16

	expiredLock, msgIDLock sync.Mutex

	// Messages is used to receive messages from NATS.
	Messages chan mainflux.RawMessage

	// Cancel channel is used to cancel observing resource.
	// Cancel channel should not be used to send or receive any
	// data, it's purpose is to be closed once Observer canceled.
	Cancel chan bool
}

// NewObserver instantiates a new Observer.
func NewObserver() *Observer {
	return &Observer{
		Messages: make(chan mainflux.RawMessage),
		Cancel:   make(chan bool),
	}
}

// LoadExpired reads Expired flag in thread-safe way.
func (o *Observer) LoadExpired() bool {
	o.expiredLock.Lock()
	defer o.expiredLock.Unlock()

	return o.expired
}

// StoreExpired stores Expired flag in thread-safe way.
func (o *Observer) StoreExpired(val bool) {
	o.expiredLock.Lock()
	defer o.expiredLock.Unlock()

	o.expired = val
}

// LoadMessageID reads MessageID and increments
// its value in thread-safe way.
func (o *Observer) LoadMessageID() uint16 {
	o.msgIDLock.Lock()
	defer o.msgIDLock.Unlock()

	o.msgID++
	return o.msgID
}
