// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/twins"
)

const (
	twinPrefix     = "twins."
	twinAdd        = twinPrefix + "add"
	twinUpdate     = twinPrefix + "update"
	twinRemove     = twinPrefix + "remove"
	twinView       = twinPrefix + "view"
	twinList       = twinPrefix + "list"
	twinListStates = twinPrefix + "list_states"
	twinSaveStates = twinPrefix + "save_states"
)

var (
	_ events.Event = (*addTwinEvent)(nil)
	_ events.Event = (*updateTwinEvent)(nil)
	_ events.Event = (*removeTwinEvent)(nil)
	_ events.Event = (*viewTwinEvent)(nil)
	_ events.Event = (*listTwinsEvent)(nil)
	_ events.Event = (*listStatesEvent)(nil)
	_ events.Event = (*saveStatesEvent)(nil)
)

type addTwinEvent struct {
	Twin       twins.Twin
	Definition twins.Definition
}

func (ate addTwinEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": twinAdd,
		"id":        ate.Twin.ID,
		"created":   ate.Twin.Created,
	}

	if ate.Twin.Owner != "" {
		val["owner"] = ate.Twin.Owner
	}
	if ate.Twin.Name != "" {
		val["name"] = ate.Twin.Name
	}
	if ate.Twin.Revision != 0 {
		val["revision"] = ate.Twin.Revision
	}
	if ate.Twin.Metadata != nil {
		metadata, err := json.Marshal(ate.Twin.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if len(ate.Twin.Definitions) > 0 {
		definitions, err := json.Marshal(ate.Twin.Definitions)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["twin_definitions"] = definitions
	}
	if ate.Definition.ID != 0 {
		val["definition_id"] = ate.Definition.ID
	}
	if ate.Definition.Created != (time.Time{}) {
		val["definition_created"] = ate.Definition.Created
	}
	if len(ate.Definition.Attributes) > 0 {
		attributes, err := json.Marshal(ate.Definition.Attributes)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["definition_attributes"] = attributes
	}
	if ate.Definition.Delta != 0 {
		val["definition_delta"] = ate.Definition.Delta
	}

	return val, nil
}

type updateTwinEvent struct {
	twin       twins.Twin
	definition twins.Definition
}

func (ute updateTwinEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": twinUpdate,
		"id":        ute.twin.ID,
	}

	if ute.twin.Owner != "" {
		val["owner"] = ute.twin.Owner
	}
	if ute.twin.Name != "" {
		val["name"] = ute.twin.Name
	}
	if ute.twin.Revision != 0 {
		val["revision"] = ute.twin.Revision
	}
	if ute.twin.Metadata != nil {
		metadata, err := json.Marshal(ute.twin.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if len(ute.twin.Definitions) > 0 {
		definitions, err := json.Marshal(ute.twin.Definitions)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["twin_definitions"] = definitions
	}
	if ute.twin.Created != (time.Time{}) {
		val["created"] = ute.twin.Created
	}
	if ute.definition.ID != 0 {
		val["definition_id"] = ute.definition.ID
	}
	if ute.definition.Created != (time.Time{}) {
		val["definition_created"] = ute.definition.Created
	}
	if len(ute.definition.Attributes) > 0 {
		attributes, err := json.Marshal(ute.definition.Attributes)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["definition_attributes"] = attributes
	}
	if ute.definition.Delta != 0 {
		val["definition_delta"] = ute.definition.Delta
	}

	return val, nil
}

type viewTwinEvent struct {
	id string
}

func (vte viewTwinEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": twinView,
		"id":        vte.id,
	}, nil
}

type removeTwinEvent struct {
	id string
}

func (rte removeTwinEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": twinRemove,
		"id":        rte.id,
	}, nil
}

type listTwinsEvent struct {
	offset   uint64
	limit    uint64
	name     string
	metadata twins.Metadata
}

func (lte listTwinsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": twinList,
	}

	if lte.name != "" {
		val["name"] = lte.name
	}
	if lte.metadata != nil {
		metadata, err := json.Marshal(lte.metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lte.offset != 0 {
		val["offset"] = lte.offset
	}
	if lte.limit != 0 {
		val["limit"] = lte.limit
	}

	return val, nil
}

type listStatesEvent struct {
	offset uint64
	limit  uint64
	id     string
}

func (lsge listStatesEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": twinListStates,
	}

	if lsge.offset != 0 {
		val["offset"] = lsge.offset
	}
	if lsge.limit != 0 {
		val["limit"] = lsge.limit
	}
	if lsge.id != "" {
		val["id"] = lsge.id
	}

	return val, nil
}

type saveStatesEvent struct {
	msg *messaging.Message
}

func (ice saveStatesEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": twinSaveStates,
	}

	if ice.msg != nil {
		msg, err := json.Marshal(ice.msg)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["message"] = msg
	}

	return val, nil
}
