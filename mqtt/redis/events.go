// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

type event interface {
	Encode() map[string]interface{}
}

var (
	_ event = (*mqttEvent)(nil)
)

type mqttEvent struct {
	clientID  string
	eventType string
	instance  string
}

func (me mqttEvent) Encode() map[string]interface{} {
	return map[string]interface{}{
		"thing_id":   me.clientID,
		"event_type": me.eventType,
		"instance":   me.instance,
	}
}
