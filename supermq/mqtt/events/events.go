// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import "github.com/absmach/supermq/pkg/events"

const (
	mqttPrefix       = "mqtt"
	clientSubscribe  = mqttPrefix + ".client_subscribe"
	clientConnect    = mqttPrefix + ".client_connect"
	clientDisconnect = mqttPrefix + ".client_disconnect"
)

var (
	_ events.Event = (*connectEvent)(nil)
	_ events.Event = (*subscribeEvent)(nil)
)

type connectEvent struct {
	operation    string
	clientID     string
	subscriberID string
	instance     string
}

func (ce connectEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":     ce.operation,
		"client_id":     ce.clientID,
		"subscriber_id": ce.subscriberID,
		"instance":      ce.instance,
	}, nil
}

type subscribeEvent struct {
	operation    string
	clientID     string
	subscriberID string
	channelID    string
	subtopic     string
}

func (se subscribeEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":     se.operation,
		"client_id":     se.clientID,
		"subscriber_id": se.subscriberID,
		"channel_id":    se.channelID,
		"subtopic":      se.subtopic,
	}, nil
}
