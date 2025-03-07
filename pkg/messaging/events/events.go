// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import "github.com/absmach/supermq/pkg/events"

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
	channelID string
	clientID  string
	subtopic  string
}

func (pe publishEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  clientPublish,
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

func (se subscribeEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":     se.operation,
		"subscriber_id": se.subscriberID,
		"client_id":     se.clientID,
		"topic":         se.topic,
	}, nil
}
