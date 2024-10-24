// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import "github.com/absmach/magistrala/pkg/events"

var _ events.Event = (*mqttEvent)(nil)

type mqttEvent struct {
	clientID  string
	operation string
	instance  string
}

func (me mqttEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id": me.clientID,
		"operation": me.operation,
		"instance":  me.instance,
	}, nil
}
