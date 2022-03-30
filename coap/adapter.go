// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/pkg/errors"
	broker "github.com/nats-io/nats.go"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/messaging"
)

const chansPrefix = "channels"

// ErrUnsubscribe indicates an error to unsubscribe
var ErrUnsubscribe = errors.New("unable to unsubscribe")

// Service specifies CoAP service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, key string, msg messaging.Message) error

	// Subscribes to channel with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key, chanID, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,
type adapterService struct {
	auth      mainflux.ThingsServiceClient
	conn      *broker.Conn
	observers map[string]observers
	obsLock   sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(auth mainflux.ThingsServiceClient, nc *broker.Conn) Service {
	as := &adapterService{
		auth:      auth,
		conn:      nc,
		observers: make(map[string]observers),
		obsLock:   sync.Mutex{},
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, msg messaging.Message) error {
	ar := &mainflux.AccessByKeyReq{
		Token:  key,
		ChanID: msg.Channel,
	}
	thid, err := svc.auth.CanAccessByKey(ctx, ar)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	msg.Publisher = thid.GetValue()

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, msg.Channel)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}

	return svc.conn.Publish(subject, data)
}

func (svc *adapterService) Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error {
	ar := &mainflux.AccessByKeyReq{
		Token:  key,
		ChanID: chanID,
	}
	if _, err := svc.auth.CanAccessByKey(ctx, ar); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	obs, err := NewObserver(subject, c, svc.conn)
	if err != nil {
		c.Cancel()
		return err
	}
	return svc.put(subject, c.Token(), obs)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, chanID, subtopic, token string) error {
	ar := &mainflux.AccessByKeyReq{
		Token:  key,
		ChanID: chanID,
	}
	if _, err := svc.auth.CanAccessByKey(ctx, ar); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	return svc.remove(subject, token)
}

func (svc *adapterService) put(topic, token string, o Observer) error {
	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()

	obs, ok := svc.observers[topic]
	// If there are no observers, create map and assign it to the topic.
	if !ok {
		obs = observers{token: o}
		svc.observers[topic] = obs
		return nil
	}
	// If observer exists, cancel subscription and replace it.
	if sub, ok := obs[token]; ok {
		if err := sub.Cancel(); err != nil {
			return errors.Wrap(ErrUnsubscribe, err)
		}
	}
	obs[token] = o
	return nil
}

func (svc *adapterService) remove(topic, token string) error {
	svc.obsLock.Lock()
	defer svc.obsLock.Unlock()
	obs, ok := svc.observers[topic]
	if !ok {
		return nil
	}
	if current, ok := obs[token]; ok {
		if err := current.Cancel(); err != nil {
			return errors.Wrap(ErrUnsubscribe, err)
		}
	}
	delete(obs, token)
	// If there are no observers left for the endpint, remove the map.
	if len(obs) == 0 {
		delete(svc.observers, topic)
	}
	return nil
}
