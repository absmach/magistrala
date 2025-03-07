// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rmMW "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/absmach/supermq/pkg/svcutil"
)

var (
	errView                     = errors.New("not authorized to view channel")
	errList                     = errors.New("not authorized to list user channels")
	errUpdate                   = errors.New("not authorized to update channel")
	errUpdateTags               = errors.New("not authorized to update channel tags")
	errEnable                   = errors.New("not authorized to enable channel")
	errDisable                  = errors.New("not authorized to disable channel")
	errDelete                   = errors.New("not authorized to delete channel")
	errConnect                  = errors.New("not authorized to connect to channel")
	errDisconnect               = errors.New("not authorized to disconnect from channel")
	errSetParentGroup           = errors.New("not authorized to set parent group to channel")
	errRemoveParentGroup        = errors.New("not authorized to remove parent group from channel")
	errDomainCreateChannels     = errors.New("not authorized to create channel in domain")
	errGroupSetChildChannels    = errors.New("not authorized to set child channel for group")
	errGroupRemoveChildChannels = errors.New("not authorized to remove child channel for group")
	errClientDisConnectChannels = errors.New("not authorized to disconnect channel for client")
	errClientConnectChannels    = errors.New("not authorized to connect channel for client")
)

var _ channels.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc    channels.Service
	repo   channels.Repository
	authz  smqauthz.Authorization
	opp    svcutil.OperationPerm
	extOpp svcutil.ExternalOperationPerm
	rmMW.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the channels service.
func AuthorizationMiddleware(svc channels.Service, repo channels.Repository, authz smqauthz.Authorization, channelsOpPerm, rolesOpPerm map[svcutil.Operation]svcutil.Permission, extOpPerm map[svcutil.ExternalOperation]svcutil.Permission) (channels.Service, error) {
	opp := channels.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(channelsOpPerm); err != nil {
		return nil, err
	}
	if err := opp.Validate(); err != nil {
		return nil, err
	}

	extOpp := channels.NewExternalOperationPerm()
	if err := extOpp.AddOperationPermissionMap(extOpPerm); err != nil {
		return nil, err
	}
	if err := extOpp.Validate(); err != nil {
		return nil, err
	}
	ram, err := rmMW.NewRoleManagerAuthorizationMiddleware(policies.ChannelType, svc, authz, rolesOpPerm)
	if err != nil {
		return nil, err
	}

	return &authorizationMiddleware{
		svc:                                svc,
		repo:                               repo,
		authz:                              authz,
		RoleManagerAuthorizationMiddleware: ram,
		opp:                                opp,
		extOpp:                             extOpp,
	}, nil
}

func (am *authorizationMiddleware) CreateChannels(ctx context.Context, session authn.Session, chs ...channels.Channel) ([]channels.Channel, []roles.RoleProvision, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.CreateOp,
			EntityID:         auth.AnyIDs,
		}); err != nil {
			return []channels.Channel{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.extAuthorize(ctx, channels.DomainOpCreateChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return []channels.Channel{}, []roles.RoleProvision{}, errors.Wrap(err, errDomainCreateChannels)
	}

	for _, ch := range chs {
		if ch.ParentGroup != "" {
			if err := am.extAuthorize(ctx, channels.GroupOpSetChildChannel, smqauthz.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.UserType,
				Subject:     session.DomainUserID,
				ObjectType:  policies.GroupType,
				Object:      ch.ParentGroup,
			}); err != nil {
				return []channels.Channel{}, []roles.RoleProvision{}, errors.Wrap(err, errors.Wrap(errGroupSetChildChannels, fmt.Errorf("channel name %s parent group id %s", ch.Name, ch.ParentGroup)))
			}
		}
	}
	return am.svc.CreateChannels(ctx, session, chs...)
}

func (am *authorizationMiddleware) ViewChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.ReadOp,
			EntityID:         id,
		}); err != nil {
			return channels.Channel{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpViewChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      id,
	}); err != nil {
		return channels.Channel{}, errors.Wrap(err, errView)
	}
	return am.svc.ViewChannel(ctx, session, id)
}

func (am *authorizationMiddleware) ListChannels(ctx context.Context, session authn.Session, pm channels.PageMetadata) (channels.Page, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.ListOp,
			EntityID:         auth.AnyIDs,
		}); err != nil {
			return channels.Page{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}
	return am.svc.ListChannels(ctx, session, pm)
}

func (am *authorizationMiddleware) ListUserChannels(ctx context.Context, session authn.Session, userID string, pm channels.PageMetadata) (channels.Page, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.ListOp,
			EntityID:         auth.AnyIDs,
		}); err != nil {
			return channels.Page{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return channels.Page{}, errors.Wrap(err, errList)
	}
	return am.svc.ListUserChannels(ctx, session, userID, pm)
}

func (am *authorizationMiddleware) UpdateChannel(ctx context.Context, session authn.Session, channel channels.Channel) (channels.Channel, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.UpdateOp,
			EntityID:         channel.ID,
		}); err != nil {
			return channels.Channel{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpUpdateChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      channel.ID,
	}); err != nil {
		return channels.Channel{}, errors.Wrap(err, errUpdate)
	}
	return am.svc.UpdateChannel(ctx, session, channel)
}

func (am *authorizationMiddleware) UpdateChannelTags(ctx context.Context, session authn.Session, channel channels.Channel) (channels.Channel, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.UpdateOp,
			EntityID:         channel.ID,
		}); err != nil {
			return channels.Channel{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpUpdateChannelTags, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      channel.ID,
	}); err != nil {
		return channels.Channel{}, errors.Wrap(err, errUpdateTags)
	}
	return am.svc.UpdateChannelTags(ctx, session, channel)
}

func (am *authorizationMiddleware) EnableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.UpdateOp,
			EntityID:         id,
		}); err != nil {
			return channels.Channel{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpEnableChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      id,
	}); err != nil {
		return channels.Channel{}, errors.Wrap(err, errEnable)
	}
	return am.svc.EnableChannel(ctx, session, id)
}

func (am *authorizationMiddleware) DisableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.UpdateOp,
			EntityID:         id,
		}); err != nil {
			return channels.Channel{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpDisableChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      id,
	}); err != nil {
		return channels.Channel{}, errors.Wrap(err, errDisable)
	}
	return am.svc.DisableChannel(ctx, session, id)
}

func (am *authorizationMiddleware) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.DeleteOp,
			EntityID:         id,
		}); err != nil {
			return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpDeleteChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errDelete)
	}
	return am.svc.RemoveChannel(ctx, session, id)
}

func (am *authorizationMiddleware) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	if session.Type == authn.PersonalAccessToken {
		for _, chID := range chIDs {
			if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
				UserID:           session.UserID,
				PatID:            session.PatID,
				EntityType:       auth.ChannelsType,
				OptionalDomainID: session.DomainID,
				Operation:        auth.CreateOp,
				EntityID:         chID,
			}); err != nil {
				return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
			}
		}
		for _, thID := range thIDs {
			if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
				UserID:           session.UserID,
				PatID:            session.PatID,
				EntityType:       auth.ClientsType,
				OptionalDomainID: session.DomainID,
				Operation:        auth.CreateOp,
				EntityID:         thID,
			}); err != nil {
				return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
			}
		}
	}
	for _, chID := range chIDs {
		if err := am.authorize(ctx, channels.OpConnectClient, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.ChannelType,
			Object:      chID,
		}); err != nil {
			return errors.Wrap(err, errConnect)
		}
	}

	for _, thID := range thIDs {
		if err := am.extAuthorize(ctx, channels.ClientsOpConnectChannel, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.ClientType,
			Object:      thID,
		}); err != nil {
			return errors.Wrap(err, errClientConnectChannels)
		}
	}
	return am.svc.Connect(ctx, session, chIDs, thIDs, connTypes)
}

func (am *authorizationMiddleware) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	if session.Type == authn.PersonalAccessToken {
		for _, chID := range chIDs {
			if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
				UserID:           session.UserID,
				PatID:            session.PatID,
				EntityType:       auth.ChannelsType,
				OptionalDomainID: session.DomainID,
				Operation:        auth.DeleteOp,
				EntityID:         chID,
			}); err != nil {
				return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
			}
		}
		for _, thID := range thIDs {
			if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
				UserID:           session.UserID,
				PatID:            session.PatID,
				EntityType:       auth.ClientsType,
				OptionalDomainID: session.DomainID,
				Operation:        auth.DeleteOp,
				EntityID:         thID,
			}); err != nil {
				return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
			}
		}
	}

	for _, chID := range chIDs {
		if err := am.authorize(ctx, channels.OpDisconnectClient, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.ChannelType,
			Object:      chID,
		}); err != nil {
			return errors.Wrap(err, errDisconnect)
		}
	}

	for _, thID := range thIDs {
		if err := am.extAuthorize(ctx, channels.ClientsOpDisconnectChannel, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.ClientType,
			Object:      thID,
		}); err != nil {
			return errors.Wrap(err, errClientDisConnectChannels)
		}
	}
	return am.svc.Disconnect(ctx, session, chIDs, thIDs, connTypes)
}

func (am *authorizationMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.UpdateOp,
			EntityID:         id,
		}); err != nil {
			return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpSetParentGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errSetParentGroup)
	}

	if err := am.extAuthorize(ctx, channels.GroupOpSetChildChannel, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.GroupType,
		Object:      parentGroupID,
	}); err != nil {
		return errors.Wrap(err, errGroupSetChildChannels)
	}
	return am.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (am *authorizationMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:           session.UserID,
			PatID:            session.PatID,
			EntityType:       auth.ChannelsType,
			OptionalDomainID: session.DomainID,
			Operation:        auth.DeleteOp,
			EntityID:         id,
		}); err != nil {
			return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.authorize(ctx, channels.OpSetParentGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ChannelType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errRemoveParentGroup)
	}

	ch, err := am.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	if ch.ParentGroup != "" {
		if err := am.extAuthorize(ctx, channels.GroupOpSetChildChannel, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.GroupType,
			Object:      ch.ParentGroup,
		}); err != nil {
			return errors.Wrap(err, errGroupRemoveChildChannels)
		}
		return am.svc.RemoveParentGroup(ctx, session, id)
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, req smqauthz.PolicyReq) error {
	perm, err := am.opp.GetPermission(op)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) extAuthorize(ctx context.Context, extOp svcutil.ExternalOperation, req smqauthz.PolicyReq) error {
	perm, err := am.extOpp.GetPermission(extOp)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, userID string) error {
	if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}); err != nil {
		return err
	}
	return nil
}
