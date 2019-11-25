// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

type createThingEvent struct {
	id                  string
	opcuaNodeIdentifier string
}

type removeThingEvent struct {
	id string
}

type createChannelEvent struct {
	id                 string
	opcuaNodeNamespace string
}

type removeChannelEvent struct {
	id string
}
