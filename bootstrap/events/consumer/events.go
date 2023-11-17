// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import "time"

type removeEvent struct {
	id string
}

type updateChannelEvent struct {
	id        string
	name      string
	metadata  map[string]interface{}
	updatedAt time.Time
	updatedBy string
}

// Connection event is either connect or disconnect event.
type disconnectEvent struct {
	thingID   string
	channelID string
}
