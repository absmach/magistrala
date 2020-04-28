// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux coap adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"fmt"
	"sync"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/messaging"
)

const (
	chanID    = "id"
	keyHeader = "key"

	// AckRandomFactor is default ACK coefficient.
	AckRandomFactor = 1.5
	// AckTimeout is the amount of time to wait for a response.
	AckTimeout = 2000 * time.Millisecond
	// MaxRetransmit is the maximum number of times a message will be retransmitted.
	MaxRetransmit = 4
)

// Service specifies coap service API.
type Service interface {
	// Publish Messssage
	Publish(msg messaging.Message) error

	// Subscribes to channel with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(chanID, subtopic, obsID string, obs *Observer) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(obsID string)
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	auth    mainflux.ThingsServiceClient
	ps      messaging.PubSub
	log     logger.Logger
	obs     map[string]*Observer
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(ps messaging.PubSub, log logger.Logger, auth mainflux.ThingsServiceClient, responses <-chan string) Service {
	as := &adapterService{
		auth:    auth,
		ps:      ps,
		log:     log,
		obs:     make(map[string]*Observer),
		obsLock: sync.Mutex{},
	}

	go as.listenResponses(responses)
	return as
}

func (svc *adapterService) get(obsID string) (*Observer, bool) {
	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	val, ok := svc.obs[obsID]
	return val, ok
}

func (svc *adapterService) put(obsID string, o *Observer) {
	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	val, ok := svc.obs[obsID]
	if ok {
		close(val.Cancel)
	}

	svc.obs[obsID] = o
}

func (svc *adapterService) remove(obsID string) {
	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	val, ok := svc.obs[obsID]
	if ok {
		close(val.Cancel)
		delete(svc.obs, obsID)
	}
}

// ListenResponses method handles ACK messages received from client.
func (svc *adapterService) listenResponses(responses <-chan string) {
	for {
		id := <-responses

		val, ok := svc.get(id)
		if ok {
			val.StoreExpired(false)
		}
	}
}

func (svc *adapterService) Publish(msg messaging.Message) error {
	return svc.ps.Publish(msg.Channel, msg)
}

func (svc *adapterService) Subscribe(chanID, subtopic, obsID string, o *Observer) error {
	subject := chanID
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", chanID, subtopic)
	}

	err := svc.ps.Subscribe(subject, func(msg messaging.Message) error {
		o.Messages <- msg
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		<-o.Cancel
		if err := svc.ps.Unsubscribe(subject); err != nil {
			svc.log.Error(fmt.Sprintf("Failed to unsubscribe from %s.%s due to %s", chanID, subtopic, err))
		}
	}()

	// Put method removes Observer if already exists.
	svc.put(obsID, o)
	return nil
}

func (svc *adapterService) Unsubscribe(obsID string) {
	svc.remove(obsID)
}
