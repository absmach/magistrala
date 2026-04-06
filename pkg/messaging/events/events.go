// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import "github.com/absmach/magistrala/pkg/events"

const (
	messagingPrefix   = "messaging"
	clientPublish     = messagingPrefix + ".client_publish"
	clientSubscribe   = messagingPrefix + ".client_subscribe"
	clientUnsubscribe = messagingPrefix + ".client_unsubscribe"
)

var (
	_ events.Event = (*publishEvent)(nil)
	_ events.Event = (*subscribeEvent)(nil)
)

type publishEvent struct {
	domainID  string
	channelID string
	clientID  string
	subtopic  string
}

func (pe publishEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":  clientPublish,
		"domain_id":  pe.domainID,
		"channel_id": pe.channelID,
		"client_id":  pe.clientID,
		"subtopic":   pe.subtopic,
	}, nil
}

type subscribeEvent struct {
	operation    string
	subscriberID string
	clientID     string
	topic        string
}

func (se subscribeEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":     se.operation,
		"subscriber_id": se.subscriberID,
		"client_id":     se.clientID,
		"topic":         se.topic,
	}, nil
}
