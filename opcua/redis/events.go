// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

type createThingEvent struct {
	id          string
	opcuaNodeID string
}

type removeThingEvent struct {
	id string
}

type connectThingEvent struct {
	chanID  string
	thingID string
}

type createChannelEvent struct {
	id             string
	opcuaServerURI string
}

type removeChannelEvent struct {
	id string
}
