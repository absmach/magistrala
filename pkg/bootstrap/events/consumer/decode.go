// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import "github.com/absmach/magistrala/pkg/events"

type connectionEvent struct {
	channelIDs []string
	clientIDs  []string
}

func decodeConnection(data map[string]any) connectionEvent {
	return connectionEvent{
		channelIDs: events.ReadStringSlice(data, "channel_ids"),
		clientIDs:  events.ReadStringSlice(data, "client_ids"),
	}
}

func readString(data map[string]any, key string) string {
	val, _ := data[key].(string)
	return val
}
