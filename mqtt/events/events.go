// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import "github.com/mainflux/mainflux/pkg/events"

var _ events.Event = (*mqttEvent)(nil)

type mqttEvent struct {
	clientID  string
	eventType string
	instance  string
}

func (me mqttEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"thing_id":   me.clientID,
		"event_type": me.eventType,
		"instance":   me.instance,
	}, nil
}
