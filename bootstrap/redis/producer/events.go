// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"strings"
	"time"

	"github.com/mainflux/mainflux/bootstrap"
)

const (
	configPrefix = "config."
	configCreate = configPrefix + "create"
	configUpdate = configPrefix + "update"
	configRemove = configPrefix + "remove"

	thingPrefix            = "thing."
	thingBootstrap         = thingPrefix + "bootstrap"
	thingStateChange       = thingPrefix + "state_change"
	thingUpdateConnections = thingPrefix + "update_connections"
)

type event interface {
	encode() map[string]interface{}
}

var (
	_ event = (*createConfigEvent)(nil)
	_ event = (*updateConfigEvent)(nil)
	_ event = (*removeConfigEvent)(nil)
	_ event = (*bootstrapEvent)(nil)
	_ event = (*changeStateEvent)(nil)
	_ event = (*updateConnectionsEvent)(nil)
)

type createConfigEvent struct {
	mfThing    string
	owner      string
	name       string
	mfChannels []string
	externalID string
	content    string
	timestamp  time.Time
}

func (cce createConfigEvent) encode() map[string]interface{} {
	return map[string]interface{}{
		"thing_id":    cce.mfThing,
		"owner":       cce.owner,
		"name":        cce.name,
		"channels":    strings.Join(cce.mfChannels, ", "),
		"external_id": cce.externalID,
		"content":     cce.content,
		"timestamp":   cce.timestamp.Unix(),
		"operation":   configCreate,
	}
}

type updateConfigEvent struct {
	mfThing   string
	name      string
	content   string
	timestamp time.Time
}

func (uce updateConfigEvent) encode() map[string]interface{} {
	return map[string]interface{}{
		"thing_id":  uce.mfThing,
		"name":      uce.name,
		"content":   uce.content,
		"timestamp": uce.timestamp.Unix(),
		"operation": configUpdate,
	}
}

type removeConfigEvent struct {
	mfThing   string
	timestamp time.Time
}

func (rce removeConfigEvent) encode() map[string]interface{} {
	return map[string]interface{}{
		"thing_id":  rce.mfThing,
		"timestamp": rce.timestamp.Unix(),
		"operation": configRemove,
	}
}

type bootstrapEvent struct {
	externalID string
	success    bool
	timestamp  time.Time
}

func (be bootstrapEvent) encode() map[string]interface{} {
	return map[string]interface{}{
		"external_id": be.externalID,
		"success":     be.success,
		"timestamp":   be.timestamp.Unix(),
		"operation":   thingBootstrap,
	}
}

type changeStateEvent struct {
	mfThing   string
	state     bootstrap.State
	timestamp time.Time
}

func (cse changeStateEvent) encode() map[string]interface{} {
	return map[string]interface{}{
		"thing_id":  cse.mfThing,
		"state":     cse.state.String(),
		"timestamp": cse.timestamp.Unix(),
		"operation": thingStateChange,
	}
}

type updateConnectionsEvent struct {
	mfThing    string
	mfChannels []string
	timestamp  time.Time
}

func (uce updateConnectionsEvent) encode() map[string]interface{} {
	return map[string]interface{}{
		"thing_id":  uce.mfThing,
		"channels":  strings.Join(uce.mfChannels, ", "),
		"timestamp": uce.timestamp.Unix(),
		"operation": thingUpdateConnections,
	}
}
