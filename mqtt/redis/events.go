// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import "github.com/mainflux/mainflux/internal/clients/redis"

var _ redis.Event = (*mqttEvent)(nil)

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
