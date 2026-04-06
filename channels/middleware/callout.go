// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/channels/operations"
	dOperations "github.com/absmach/magistrala/domains/operations"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/callout"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	rolemw "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
)

var _ channels.Service = (*calloutMiddleware)(nil)

type calloutMiddleware struct {
	svc         channels.Service
	repo        channels.Repository
	callout     callout.Callout
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemw.RoleManagerCalloutMiddleware
}

func NewCallout(svc channels.Service, repo channels.Repository, entitiesOps permissions.EntitiesOperations[permissions.Operation], roleOps permissions.Operations[permissions.RoleOperation], callout callout.Callout) (channels.Service, error) {
	call, err := rolemw.NewCallout(policies.ChannelType, svc, callout, roleOps)
	if err != nil {
		return nil, err
	}

	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}

	return &calloutMiddleware{
		svc:                          svc,
		repo:                         repo,
		callout:                      callout,
		entitiesOps:                  entitiesOps,
		RoleManagerCalloutMiddleware: call,
	}, nil
}

func (cm *calloutMiddleware) CreateChannels(ctx context.Context, session authn.Session, chs ...channels.Channel) ([]channels.Channel, []roles.RoleProvision, error) {
	params := map[string]any{
		"entities": chs,
		"count":    len(chs),
	}

	if err := cm.callOut(ctx, session, policies.DomainType, dOperations.OpCreateDomainChannels, params); err != nil {
		return []channels.Channel{}, []roles.RoleProvision{}, err
	}

	return cm.svc.CreateChannels(ctx, session, chs...)
}

func (cm *calloutMiddleware) ViewChannel(ctx context.Context, session authn.Session, id string, withRoles bool) (channels.Channel, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpViewChannel, params); err != nil {
		return channels.Channel{}, err
	}

	return cm.svc.ViewChannel(ctx, session, id, withRoles)
}

func (cm *calloutMiddleware) ListChannels(ctx context.Context, session authn.Session, pm channels.Page) (channels.ChannelsPage, error) {
	params := map[string]any{
		"pagemeta": pm,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, dOperations.OpListDomainChannels, params); err != nil {
		return channels.ChannelsPage{}, err
	}

	return cm.svc.ListChannels(ctx, session, pm)
}

func (cm *calloutMiddleware) ListUserChannels(ctx context.Context, session authn.Session, userID string, pm channels.Page) (channels.ChannelsPage, error) {
	params := map[string]any{
		"user_id":  userID,
		"pagemeta": pm,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpListUserChannels, params); err != nil {
		return channels.ChannelsPage{}, err
	}

	return cm.svc.ListUserChannels(ctx, session, userID, pm)
}

func (cm *calloutMiddleware) UpdateChannel(ctx context.Context, session authn.Session, channel channels.Channel) (channels.Channel, error) {
	params := map[string]any{
		"entity_id": channel.ID,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpUpdateChannel, params); err != nil {
		return channels.Channel{}, err
	}

	return cm.svc.UpdateChannel(ctx, session, channel)
}

func (cm *calloutMiddleware) UpdateChannelTags(ctx context.Context, session authn.Session, channel channels.Channel) (channels.Channel, error) {
	params := map[string]any{
		"entity_id": channel.ID,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpUpdateChannelTags, params); err != nil {
		return channels.Channel{}, err
	}

	return cm.svc.UpdateChannelTags(ctx, session, channel)
}

func (cm *calloutMiddleware) EnableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpEnableChannel, params); err != nil {
		return channels.Channel{}, err
	}

	return cm.svc.EnableChannel(ctx, session, id)
}

func (cm *calloutMiddleware) DisableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpDisableChannel, params); err != nil {
		return channels.Channel{}, err
	}

	return cm.svc.DisableChannel(ctx, session, id)
}

func (cm *calloutMiddleware) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpDeleteChannel, params); err != nil {
		return err
	}

	return cm.svc.RemoveChannel(ctx, session, id)
}

func (cm *calloutMiddleware) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	params := map[string]any{
		"channel_ids":      chIDs,
		"client_ids":       thIDs,
		"connection_types": connTypes,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpConnectClient, params); err != nil {
		return err
	}

	return cm.svc.Connect(ctx, session, chIDs, thIDs, connTypes)
}

func (cm *calloutMiddleware) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	params := map[string]any{
		"channel_ids":      chIDs,
		"client_ids":       thIDs,
		"connection_types": connTypes,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpDisconnectClient, params); err != nil {
		return err
	}

	return cm.svc.Disconnect(ctx, session, chIDs, thIDs, connTypes)
}

func (cm *calloutMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error {
	params := map[string]any{
		"entity_id":       id,
		"parent_group_id": parentGroupID,
	}

	if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpSetParentGroup, params); err != nil {
		return err
	}

	return cm.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (cm *calloutMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	ch, err := cm.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	if ch.ParentGroup != "" {
		params := map[string]any{
			"entity_id":       id,
			"parent_group_id": ch.ParentGroup,
		}

		if err := cm.callOut(ctx, session, policies.ChannelType, operations.OpRemoveParentGroup, params); err != nil {
			return err
		}
	}

	return cm.svc.RemoveParentGroup(ctx, session, id)
}

func (cm *calloutMiddleware) callOut(ctx context.Context, session authn.Session, entityType string, op permissions.Operation, pld map[string]any) error {
	var entityID string
	if id, ok := pld["entity_id"].(string); ok {
		entityID = id
	}

	req := callout.Request{
		BaseRequest: callout.BaseRequest{
			Operation:  cm.entitiesOps.OperationName(entityType, op),
			EntityType: entityType,
			EntityID:   entityID,
			CallerID:   session.UserID,
			CallerType: policies.UserType,
			DomainID:   session.DomainID,
			Time:       time.Now().UTC(),
		},
		Payload: pld,
	}

	if err := cm.callout.Callout(ctx, req); err != nil {
		return err
	}

	return nil
}
