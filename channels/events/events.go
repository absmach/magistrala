// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/roles"
)

const (
	channelPrefix       = "channel."
	channelCreate       = channelPrefix + "create"
	channelUpdate       = channelPrefix + "update"
	channelUpdateTags   = channelPrefix + "update_tags"
	channelEnable       = channelPrefix + "enable"
	channelDisable      = channelPrefix + "disable"
	channelRemove       = channelPrefix + "remove"
	channelView         = channelPrefix + "view"
	channelList         = channelPrefix + "list"
	channelListByUser   = channelPrefix + "list_by_user"
	channelConnect      = channelPrefix + "connect"
	channelDisconnect   = channelPrefix + "disconnect"
	channelSetParent    = channelPrefix + "set_parent"
	channelRemoveParent = channelPrefix + "remove_parent"
)

var (
	_ events.Event = (*createChannelEvent)(nil)
	_ events.Event = (*updateChannelEvent)(nil)
	_ events.Event = (*changeChannelStatusEvent)(nil)
	_ events.Event = (*viewChannelEvent)(nil)
	_ events.Event = (*listChannelEvent)(nil)
	_ events.Event = (*removeChannelEvent)(nil)
	_ events.Event = (*connectEvent)(nil)
	_ events.Event = (*disconnectEvent)(nil)
)

type createChannelEvent struct {
	channels.Channel
	rolesProvisioned []roles.RoleProvision
	authn.Session
	requestID string
}

func (cce createChannelEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":         channelCreate,
		"id":                cce.ID,
		"roles_provisioned": cce.rolesProvisioned,
		"route":             cce.Route,
		"status":            cce.Status.String(),
		"created_at":        cce.CreatedAt,
		"domain":            cce.DomainID,
		"user_id":           cce.UserID,
		"token_type":        cce.Type.String(),
		"super_admin":       cce.SuperAdmin,
		"request_id":        cce.requestID,
	}

	if cce.Name != "" {
		val["name"] = cce.Name
	}
	if len(cce.Tags) > 0 {
		val["tags"] = cce.Tags
	}
	if cce.Metadata != nil {
		val["metadata"] = cce.Metadata
	}

	return val, nil
}

type updateChannelEvent struct {
	channels.Channel
	authn.Session
	operation string
	requestID string
}

func (uce updateChannelEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   uce.operation,
		"updated_at":  uce.UpdatedAt,
		"updated_by":  uce.UpdatedBy,
		"domain":      uce.DomainID,
		"user_id":     uce.UserID,
		"token_type":  uce.Type.String(),
		"super_admin": uce.SuperAdmin,
		"request_id":  uce.requestID,
	}

	if uce.ID != "" {
		val["id"] = uce.ID
	}
	if uce.Route != "" {
		val["route"] = uce.Route
	}
	if uce.Name != "" {
		val["name"] = uce.Name
	}
	if len(uce.Tags) > 0 {
		val["tags"] = uce.Tags
	}
	if uce.Metadata != nil {
		val["metadata"] = uce.Metadata
	}
	if !uce.CreatedAt.IsZero() {
		val["created_at"] = uce.CreatedAt
	}
	if uce.Status.String() != "" {
		val["status"] = uce.Status.String()
	}

	return val, nil
}

type changeChannelStatusEvent struct {
	id        string
	operation string
	status    string
	updatedAt time.Time
	updatedBy string
	authn.Session
	requestID string
}

func (cse changeChannelStatusEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   cse.operation,
		"id":          cse.id,
		"status":      cse.status,
		"updated_at":  cse.updatedAt,
		"updated_by":  cse.updatedBy,
		"domain":      cse.DomainID,
		"user_id":     cse.UserID,
		"token_type":  cse.Type.String(),
		"super_admin": cse.SuperAdmin,
		"request_id":  cse.requestID,
	}, nil
}

type viewChannelEvent struct {
	channels.Channel
	authn.Session
	requestID string
}

func (vce viewChannelEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   channelView,
		"id":          vce.ID,
		"domain":      vce.DomainID,
		"user_id":     vce.UserID,
		"token_type":  vce.Type.String(),
		"super_admin": vce.SuperAdmin,
		"request_id":  vce.requestID,
	}

	if vce.Name != "" {
		val["name"] = vce.Name
	}
	if vce.Route != "" {
		val["route"] = vce.Route
	}
	if len(vce.Tags) > 0 {
		val["tags"] = vce.Tags
	}
	if vce.Metadata != nil {
		val["metadata"] = vce.Metadata
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

type listChannelEvent struct {
	channels.Page
	authn.Session
	requestID string
}

func (lce listChannelEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   channelList,
		"total":       lce.Total,
		"offset":      lce.Offset,
		"limit":       lce.Limit,
		"domain":      lce.DomainID,
		"user_id":     lce.UserID,
		"token_type":  lce.Type.String(),
		"super_admin": lce.SuperAdmin,
		"request_id":  lce.requestID,
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
		val["metadata"] = lce.Metadata
	}
	if len(lce.Tags.Elements) > 0 {
		val["tag"] = lce.Tags.Elements
	}
	if lce.Status.String() != "" {
		val["status"] = lce.Status.String()
	}
	if len(lce.IDs) > 0 {
		val["ids"] = lce.IDs
	}

	return val, nil
}

type listUserChannelsEvent struct {
	userID string
	channels.Page
	authn.Session
	requestID string
}

func (luce listUserChannelsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   channelListByUser,
		"req_user_id": luce.userID,
		"total":       luce.Total,
		"offset":      luce.Offset,
		"limit":       luce.Limit,
		"domain":      luce.DomainID,
		"user_id":     luce.UserID,
		"token_type":  luce.Type.String(),
		"super_admin": luce.SuperAdmin,
		"request_id":  luce.requestID,
	}

	if luce.Name != "" {
		val["name"] = luce.Name
	}
	if luce.Order != "" {
		val["order"] = luce.Order
	}
	if luce.Dir != "" {
		val["dir"] = luce.Dir
	}
	if luce.Metadata != nil {
		val["metadata"] = luce.Metadata
	}
	if luce.Domain != "" {
		val["domain"] = luce.Domain
	}
	if len(luce.Tags.Elements) > 0 {
		val["tag"] = luce.Tags.Elements
	}
	if luce.Status.String() != "" {
		val["status"] = luce.Status.String()
	}
	if len(luce.IDs) > 0 {
		val["ids"] = luce.IDs
	}

	return val, nil
}

type removeChannelEvent struct {
	id string
	authn.Session
	requestID string
}

func (dce removeChannelEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   channelRemove,
		"id":          dce.id,
		"domain":      dce.DomainID,
		"user_id":     dce.UserID,
		"token_type":  dce.Type.String(),
		"super_admin": dce.SuperAdmin,
		"request_id":  dce.requestID,
	}, nil
}

type connectEvent struct {
	chIDs []string
	thIDs []string
	types []connections.ConnType
	authn.Session
	requestID string
}

func (ce connectEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   channelConnect,
		"client_ids":  ce.thIDs,
		"channel_ids": ce.chIDs,
		"types":       ce.types,
		"domain":      ce.DomainID,
		"user_id":     ce.UserID,
		"token_type":  ce.Type.String(),
		"super_admin": ce.SuperAdmin,
		"request_id":  ce.requestID,
	}, nil
}

type disconnectEvent struct {
	chIDs []string
	thIDs []string
	types []connections.ConnType
	authn.Session
	requestID string
}

func (de disconnectEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   channelDisconnect,
		"client_ids":  de.thIDs,
		"channel_ids": de.chIDs,
		"types":       de.types,
		"domain":      de.DomainID,
		"user_id":     de.UserID,
		"token_type":  de.Type.String(),
		"super_admin": de.SuperAdmin,
		"request_id":  de.requestID,
	}, nil
}

type setParentGroupEvent struct {
	id            string
	parentGroupID string
	authn.Session
	requestID string
}

func (spge setParentGroupEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":       channelSetParent,
		"id":              spge.id,
		"parent_group_id": spge.parentGroupID,
		"domain":          spge.DomainID,
		"user_id":         spge.UserID,
		"token_type":      spge.Type.String(),
		"super_admin":     spge.SuperAdmin,
		"request_id":      spge.requestID,
	}, nil
}

type removeParentGroupEvent struct {
	id string
	authn.Session
	requestID string
}

func (rpge removeParentGroupEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   channelRemoveParent,
		"id":          rpge.id,
		"domain":      rpge.DomainID,
		"user_id":     rpge.UserID,
		"token_type":  rpge.Type.String(),
		"super_admin": rpge.SuperAdmin,
		"request_id":  rpge.requestID,
	}, nil
}
