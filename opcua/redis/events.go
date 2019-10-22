// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

type createThingEvent struct {
	id       string
	metadata thingMetadata
}

type updateThingEvent struct {
	id       string
	metadata thingMetadata
}

type removeThingEvent struct {
	id string
}

type thingMetadata struct {
	ID string `json:"id"`
}

type createChannelEvent struct {
	id       string
	metadata channelMetadata
}

type updateChannelEvent struct {
	id       string
	metadata channelMetadata
}

type removeChannelEvent struct {
	id string
}

type channelMetadata struct {
	Namespace string `json:"namespace"`
}
