// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/supermq"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

const (
	clientCreate         = "client.create"
	clientRemove         = "client.remove"
	mqttSubscribe        = "mqtt.client_subscribe"
	mqttDisconnect       = "mqtt.client_disconnect"
	messagingPublish     = "messaging.client_publish"
	messagingSubscribe   = "messaging.client_subscribe"
	messagingUnsubscribe = "messaging.client_unsubscribe"
)

type service struct {
	idProvider supermq.IDProvider
	repository Repository
}

func NewService(idp supermq.IDProvider, repository Repository) Service {
	return &service{
		idProvider: idp,
		repository: repository,
	}
}

func (svc *service) Save(ctx context.Context, journal Journal) error {
	id, err := svc.idProvider.ID()
	if err != nil {
		return err
	}
	journal.ID = id

	if err := svc.repository.Save(ctx, journal); err != nil {
		return err
	}
	if err := svc.handleTelemetry(ctx, journal); err != nil {
		return err
	}

	return nil
}

func (svc *service) RetrieveAll(ctx context.Context, session smqauthn.Session, page Page) (JournalsPage, error) {
	journalPage, err := svc.repository.RetrieveAll(ctx, page)
	if err != nil {
		return JournalsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return journalPage, nil
}

func (svc *service) RetrieveClientTelemetry(ctx context.Context, session smqauthn.Session, clientID string) (ClientTelemetry, error) {
	ct, err := svc.repository.RetrieveClientTelemetry(ctx, clientID, session.DomainID)
	if err != nil {
		return ClientTelemetry{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	subs, err := svc.repository.CountSubscriptions(ctx, clientID)
	if err != nil {
		return ClientTelemetry{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	ct.Subscriptions = subs

	return ct, nil
}

func (svc *service) handleTelemetry(ctx context.Context, journal Journal) error {
	switch journal.Operation {
	case clientCreate:
		return svc.addClientTelemetry(ctx, journal)

	case clientRemove:
		return svc.removeClientTelemetry(ctx, journal)

	case mqttSubscribe:
		return svc.addMqttSubscription(ctx, journal)

	case messagingSubscribe:
		return svc.addSubscription(ctx, journal)

	case messagingUnsubscribe:
		return svc.removeSubscription(ctx, journal)

	case messagingPublish:
		return svc.updateMessageCount(ctx, journal)

	case mqttDisconnect:
		return svc.removeMqttSubscription(ctx, journal)

	default:
		return nil
	}
}

func (svc *service) addClientTelemetry(ctx context.Context, journal Journal) error {
	ce, err := toClientEvent(journal)
	if err != nil {
		return err
	}
	ct := ClientTelemetry{
		ClientID:  ce.id,
		DomainID:  ce.domain,
		FirstSeen: ce.createdAt,
		LastSeen:  ce.createdAt,
	}
	return svc.repository.SaveClientTelemetry(ctx, ct)
}

func (svc *service) removeClientTelemetry(ctx context.Context, journal Journal) error {
	ce, err := toClientEvent(journal)
	if err != nil {
		return err
	}
	return svc.repository.DeleteClientTelemetry(ctx, ce.id, ce.domain)
}

func (svc *service) addSubscription(ctx context.Context, journal Journal) error {
	ae, err := toSubscribeEvent(journal)
	if err != nil {
		return err
	}
	var subtopic string
	topics := strings.Split(ae.topic, ".")
	if len(topics) > 2 {
		subtopic = topics[2]
	}

	id, err := svc.idProvider.ID()
	if err != nil {
		return err
	}

	sub := ClientSubscription{
		ID:           id,
		SubscriberID: ae.subscriberID,
		ChannelID:    topics[1],
		Subtopic:     subtopic,
		ClientID:     ae.clientID,
	}

	return svc.repository.AddSubscription(ctx, sub)
}

func (svc *service) addMqttSubscription(ctx context.Context, journal Journal) error {
	ae, err := toMqttSubscribeEvent(journal)
	if err != nil {
		return err
	}

	id, err := svc.idProvider.ID()
	if err != nil {
		return err
	}

	sub := ClientSubscription{
		ID:           id,
		SubscriberID: ae.subscriberID,
		ChannelID:    ae.channelID,
		Subtopic:     ae.subtopic,
		ClientID:     ae.clientID,
	}

	return svc.repository.AddSubscription(ctx, sub)
}

func (svc *service) removeSubscription(ctx context.Context, journal Journal) error {
	ae, err := toUnsubscribeEvent(journal)
	if err != nil {
		return err
	}

	return svc.repository.RemoveSubscription(ctx, ae.subscriberID)
}

func (svc *service) removeMqttSubscription(ctx context.Context, journal Journal) error {
	ae, err := toMqttDisconnectEvent(journal)
	if err != nil {
		return err
	}

	return svc.repository.RemoveSubscription(ctx, ae.subscriberID)
}

func (svc *service) updateMessageCount(ctx context.Context, journal Journal) error {
	ae, err := toPublishEvent(journal)
	if err != nil {
		return err
	}
	if err := svc.repository.IncrementInboundMessages(ctx, ae.clientID); err != nil {
		return err
	}
	if err := svc.repository.IncrementOutboundMessages(ctx, ae.channelID, ae.subtopic); err != nil {
		return err
	}
	return nil
}

type clientEvent struct {
	id        string
	domain    string
	createdAt time.Time
}

func toClientEvent(journal Journal) (clientEvent, error) {
	var createdAt time.Time
	var err error
	id, err := getStringAttribute(journal, "id")
	if err != nil {
		return clientEvent{}, err
	}
	domain, err := getStringAttribute(journal, "domain")
	if err != nil {
		return clientEvent{}, err
	}

	createdAtStr := journal.Attributes["created_at"].(string)
	if createdAtStr != "" {
		createdAt, err = time.Parse(time.RFC3339, createdAtStr)
		if err != nil {
			return clientEvent{}, fmt.Errorf("invalid created_at format")
		}
	}
	return clientEvent{
		id:        id,
		domain:    domain,
		createdAt: createdAt,
	}, nil
}

type adapterEvent struct {
	clientID     string
	channelID    string
	subscriberID string
	topic        string
	subtopic     string
}

func toPublishEvent(journal Journal) (adapterEvent, error) {
	clientID, err := getStringAttribute(journal, "client_id")
	if err != nil {
		return adapterEvent{}, err
	}
	channelID, err := getStringAttribute(journal, "channel_id")
	if err != nil {
		return adapterEvent{}, err
	}
	subtopic, err := getStringAttribute(journal, "subtopic")
	if err != nil {
		return adapterEvent{}, err
	}

	return adapterEvent{
		clientID:  clientID,
		channelID: channelID,
		subtopic:  subtopic,
	}, nil
}

func toSubscribeEvent(journal Journal) (adapterEvent, error) {
	subscriberID, err := getStringAttribute(journal, "subscriber_id")
	if err != nil {
		return adapterEvent{}, err
	}
	topic, err := getStringAttribute(journal, "topic")
	if err != nil {
		return adapterEvent{}, err
	}
	var clientID string
	clientID, err = getStringAttribute(journal, "client_id")
	if err != nil {
		clientID = ""
	}

	return adapterEvent{
		clientID:     clientID,
		subscriberID: subscriberID,
		topic:        topic,
	}, nil
}

func toUnsubscribeEvent(journal Journal) (adapterEvent, error) {
	subscriberID, err := getStringAttribute(journal, "subscriber_id")
	if err != nil {
		return adapterEvent{}, err
	}
	topic, err := getStringAttribute(journal, "topic")
	if err != nil {
		return adapterEvent{}, err
	}

	return adapterEvent{
		subscriberID: subscriberID,
		topic:        topic,
	}, nil
}

func toMqttSubscribeEvent(journal Journal) (adapterEvent, error) {
	clientID, err := getStringAttribute(journal, "client_id")
	if err != nil {
		return adapterEvent{}, err
	}
	subscriberID, err := getStringAttribute(journal, "subscriber_id")
	if err != nil {
		return adapterEvent{}, err
	}
	channelID, err := getStringAttribute(journal, "channel_id")
	if err != nil {
		return adapterEvent{}, err
	}
	subtopic, err := getStringAttribute(journal, "subtopic")
	if err != nil {
		return adapterEvent{}, err
	}

	return adapterEvent{
		clientID:     clientID,
		subscriberID: subscriberID,
		channelID:    channelID,
		subtopic:     subtopic,
	}, nil
}

func toMqttDisconnectEvent(journal Journal) (adapterEvent, error) {
	subscriberID, err := getStringAttribute(journal, "subscriber_id")
	if err != nil {
		return adapterEvent{}, err
	}
	clientID, err := getStringAttribute(journal, "client_id")
	if err != nil {
		return adapterEvent{}, err
	}

	return adapterEvent{
		subscriberID: subscriberID,
		channelID:    clientID,
	}, nil
}

func getStringAttribute(journal Journal, key string) (string, error) {
	value, ok := journal.Attributes[key].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid %s attribute", key)
	}
	return value, nil
}
