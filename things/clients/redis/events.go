// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
)

const (
	clientPrefix      = "thing."
	clientCreate      = clientPrefix + "create"
	clientUpdate      = clientPrefix + "update"
	clientRemove      = clientPrefix + "remove"
	clientView        = clientPrefix + "view"
	clientList        = clientPrefix + "list"
	clientListByGroup = clientPrefix + "list_by_group"
	clientIdentify    = clientPrefix + "identify"
)

type event interface {
	Encode() (map[string]interface{}, error)
}

var (
	_ event = (*createClientEvent)(nil)
	_ event = (*updateClientEvent)(nil)
	_ event = (*removeClientEvent)(nil)
	_ event = (*viewClientEvent)(nil)
	_ event = (*listClientEvent)(nil)
	_ event = (*listClientByGroupEvent)(nil)
	_ event = (*identifyClientEvent)(nil)
)

type createClientEvent struct {
	mfclients.Client
}

func (cce createClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  clientCreate,
		"id":         cce.ID,
		"status":     cce.Status.String(),
		"created_at": cce.CreatedAt,
	}

	if cce.Name != "" {
		val["name"] = cce.Name
	}
	if len(cce.Tags) > 0 {
		tags := fmt.Sprintf("[%s]", strings.Join(cce.Tags, ","))
		val["tags"] = tags
	}
	if cce.Owner != "" {
		val["owner"] = cce.Owner
	}
	if cce.Metadata != nil {
		metadata, err := json.Marshal(cce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if cce.Credentials.Identity != "" {
		val["identity"] = cce.Credentials.Identity
	}

	return val, nil
}

type updateClientEvent struct {
	mfclients.Client
	operation string
}

func (uce updateClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  clientUpdate + "." + uce.operation,
		"updated_at": uce.UpdatedAt,
		"updated_by": uce.UpdatedBy,
	}

	if uce.ID != "" {
		val["id"] = uce.ID
	}
	if uce.Name != "" {
		val["name"] = uce.Name
	}
	if len(uce.Tags) > 0 {
		tags := fmt.Sprintf("[%s]", strings.Join(uce.Tags, ","))
		val["tags"] = tags
	}
	if uce.Credentials.Identity != "" {
		val["identity"] = uce.Credentials.Identity
	}
	if uce.Metadata != nil {
		metadata, err := json.Marshal(uce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if !uce.CreatedAt.IsZero() {
		val["created_at"] = uce.CreatedAt
	}
	if uce.Status.String() != "" {
		val["status"] = uce.Status.String()
	}

	return val, nil
}

type removeClientEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
}

func (rce removeClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  clientRemove,
		"id":         rce.id,
		"status":     rce.status,
		"updated_at": rce.updatedAt,
		"updated_by": rce.updatedBy,
	}, nil
}

type viewClientEvent struct {
	mfclients.Client
}

func (vce viewClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": clientView,
		"id":        vce.ID,
	}

	if vce.Name != "" {
		val["name"] = vce.Name
	}
	if len(vce.Tags) > 0 {
		tags := fmt.Sprintf("[%s]", strings.Join(vce.Tags, ","))
		val["tags"] = tags
	}
	if vce.Owner != "" {
		val["owner"] = vce.Owner
	}
	if vce.Credentials.Identity != "" {
		val["identity"] = vce.Credentials.Identity
	}
	if vce.Metadata != nil {
		metadata, err := json.Marshal(vce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if !vce.CreatedAt.IsZero() {
		val["created_at"] = vce.CreatedAt
	}
	if !vce.UpdatedAt.IsZero() {
		val["updated_at"] = vce.UpdatedAt
	}
	if vce.UpdatedBy != "" {
		val["updated_by"] = vce.UpdatedBy
	}
	if vce.Status.String() != "" {
		val["status"] = vce.Status.String()
	}

	return val, nil
}

type listClientEvent struct {
	mfclients.Page
}

func (lce listClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": clientList,
		"total":     lce.Total,
		"offset":    lce.Offset,
		"limit":     lce.Limit,
	}

	if lce.Name != "" {
		val["name"] = lce.Name
	}
	if lce.Order != "" {
		val["order"] = lce.Order
	}
	if lce.Dir != "" {
		val["dir"] = lce.Dir
	}
	if lce.Metadata != nil {
		metadata, err := json.Marshal(lce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lce.Owner != "" {
		val["owner"] = lce.Owner
	}
	if lce.Tag != "" {
		val["tag"] = lce.Tag
	}
	if lce.SharedBy != "" {
		val["sharedBy"] = lce.SharedBy
	}
	if lce.Status.String() != "" {
		val["status"] = lce.Status.String()
	}
	if lce.Action != "" {
		val["action"] = lce.Action
	}
	if lce.Subject != "" {
		val["subject"] = lce.Subject
	}
	if lce.Identity != "" {
		val["identity"] = lce.Identity
	}

	return val, nil
}

type listClientByGroupEvent struct {
	mfclients.Page
	channelID string
}

func (lcge listClientByGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  clientListByGroup,
		"total":      lcge.Total,
		"offset":     lcge.Offset,
		"limit":      lcge.Limit,
		"channel_id": lcge.channelID,
	}

	if lcge.Name != "" {
		val["name"] = lcge.Name
	}
	if lcge.Order != "" {
		val["order"] = lcge.Order
	}
	if lcge.Dir != "" {
		val["dir"] = lcge.Dir
	}
	if lcge.Metadata != nil {
		metadata, err := json.Marshal(lcge.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lcge.Owner != "" {
		val["owner"] = lcge.Owner
	}
	if lcge.Tag != "" {
		val["tag"] = lcge.Tag
	}
	if lcge.SharedBy != "" {
		val["sharedBy"] = lcge.SharedBy
	}
	if lcge.Status.String() != "" {
		val["status"] = lcge.Status.String()
	}
	if lcge.Action != "" {
		val["action"] = lcge.Action
	}
	if lcge.Subject != "" {
		val["subject"] = lcge.Subject
	}
	if lcge.Identity != "" {
		val["identity"] = lcge.Identity
	}

	return val, nil
}

type identifyClientEvent struct {
	thingID string
}

func (ice identifyClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": clientIdentify,
		"thing_id":  ice.thingID,
	}, nil
}
