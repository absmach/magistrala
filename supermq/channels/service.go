// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/supermq"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	grpcGroupsV1 "github.com/absmach/supermq/api/grpc/groups/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqclients "github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
)

var (
	errAddConnectionsClients    = errors.New("failed to add connections in clients service")
	errRemoveConnectionsClients = errors.New("failed to remove connections from clients service")
	errSetParentGroup           = errors.New("channel already have parent")
)

type service struct {
	repo       Repository
	policy     policies.Service
	idProvider supermq.IDProvider
	clients    grpcClientsV1.ClientsServiceClient
	groups     grpcGroupsV1.GroupsServiceClient
	roles.ProvisionManageService
}

var _ Service = (*service)(nil)

func New(repo Repository, policy policies.Service, idProvider supermq.IDProvider, clients grpcClientsV1.ClientsServiceClient, groups grpcGroupsV1.GroupsServiceClient, sidProvider supermq.IDProvider, availableActions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (Service, error) {
	rpms, err := roles.NewProvisionManageService(policies.ChannelType, repo, policy, sidProvider, availableActions, builtInRoles)
	if err != nil {
		return nil, err
	}

	return service{
		repo:                   repo,
		policy:                 policy,
		idProvider:             idProvider,
		clients:                clients,
		groups:                 groups,
		ProvisionManageService: rpms,
	}, nil
}

func (svc service) CreateChannels(ctx context.Context, session authn.Session, chs ...Channel) (retChs []Channel, retRps []roles.RoleProvision, retErr error) {
	var reChs []Channel
	for _, c := range chs {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []Channel{}, []roles.RoleProvision{}, err
			}
			c.ID = clientID
		}

		if c.Status != smqclients.DisabledStatus && c.Status != smqclients.EnabledStatus {
			return []Channel{}, []roles.RoleProvision{}, svcerr.ErrInvalidStatus
		}
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		reChs = append(reChs, c)
	}

	savedChs, err := svc.repo.Save(ctx, reChs...)
	if err != nil {
		return []Channel{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	chIDs := []string{}
	for _, c := range savedChs {
		chIDs = append(chIDs, c.ID)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := svc.repo.Remove(ctx, chIDs...); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(svcerr.ErrRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []policies.Policy{}

	for _, chID := range chIDs {
		optionalPolicies = append(optionalPolicies,
			policies.Policy{
				SubjectType: policies.DomainType,
				Subject:     session.DomainID,
				Relation:    policies.DomainRelation,
				ObjectType:  policies.ChannelType,
				Object:      chID,
			},
		)
	}
	nrps, err := svc.AddNewEntitiesRoles(ctx, session.DomainID, session.UserID, chIDs, optionalPolicies, newBuiltInRoleMembers)
	if err != nil {
		return []Channel{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	return savedChs, nrps, nil
}

func (svc service) UpdateChannel(ctx context.Context, session authn.Session, ch Channel) (Channel, error) {
	channel := Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Metadata:  ch.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	channel, err := svc.repo.Update(ctx, channel)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}

func (svc service) UpdateChannelTags(ctx context.Context, session authn.Session, ch Channel) (Channel, error) {
	channel := Channel{
		ID:        ch.ID,
		Tags:      ch.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	channel, err := svc.repo.UpdateTags(ctx, channel)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}

func (svc service) EnableChannel(ctx context.Context, session authn.Session, id string) (Channel, error) {
	channel := Channel{
		ID:        id,
		Status:    smqclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	ch, err := svc.changeChannelStatus(ctx, session.UserID, channel)
	if err != nil {
		return Channel{}, errors.Wrap(smqclients.ErrEnableClient, err)
	}

	return ch, nil
}

func (svc service) DisableChannel(ctx context.Context, session authn.Session, id string) (Channel, error) {
	channel := Channel{
		ID:        id,
		Status:    smqclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	ch, err := svc.changeChannelStatus(ctx, session.UserID, channel)
	if err != nil {
		return Channel{}, errors.Wrap(smqclients.ErrDisableClient, err)
	}

	return ch, nil
}

func (svc service) ViewChannel(ctx context.Context, session authn.Session, id string) (Channel, error) {
	channel, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return channel, nil
}

func (svc service) ListChannels(ctx context.Context, session authn.Session, pm PageMetadata) (Page, error) {
	switch session.SuperAdmin {
	case true:
		cp, err := svc.repo.RetrieveAll(ctx, pm)
		if err != nil {
			return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		return cp, nil
	default:
		cp, err := svc.repo.RetrieveUserChannels(ctx, session.DomainID, session.UserID, pm)
		if err != nil {
			return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		return cp, nil
	}
}

func (svc service) ListUserChannels(ctx context.Context, session authn.Session, userID string, pm PageMetadata) (Page, error) {
	cp, err := svc.repo.RetrieveUserChannels(ctx, session.DomainID, userID, pm)
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return cp, nil
}

func (svc service) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	ok, err := svc.repo.DoesChannelHaveConnections(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if ok {
		if _, err := svc.clients.RemoveChannelConnections(ctx, &grpcClientsV1.RemoveChannelConnectionsReq{ChannelId: id}); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}
	ch, err := svc.repo.ChangeStatus(ctx, Channel{ID: id, Status: smqclients.DeletedStatus})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	deletePolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     session.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ChannelType,
			Object:      id,
		},
	}

	if ch.ParentGroup != "" {
		deletePolicies = append(deletePolicies, policies.Policy{
			SubjectType: policies.GroupType,
			Subject:     ch.ParentGroup,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.ChannelType,
			Object:      id,
		})
	}

	filterDeletePolicies := []policies.Policy{
		{
			SubjectType: policies.ChannelType,
			Subject:     id,
		},
		{
			ObjectType: policies.ChannelType,
			Object:     id,
		},
	}

	if err := svc.RemoveEntitiesRoles(ctx, session.DomainID, session.DomainUserID, []string{id}, filterDeletePolicies, deletePolicies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if err := svc.repo.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) (retErr error) {
	for _, chID := range chIDs {
		c, err := svc.repo.RetrieveByID(ctx, chID)
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		if c.Status != smqclients.EnabledStatus {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("channel id %s is not in enabled state", chID))
		}
		if c.Domain != session.DomainID {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("channel id %s has invalid domain id", chID))
		}
	}

	for _, thID := range thIDs {
		resp, err := svc.clients.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: thID})
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		if resp.GetEntity().GetStatus() != uint32(smqclients.EnabledStatus) {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("client id %s is not in enabled state", thID))
		}
		if resp.GetEntity().GetDomainId() != session.DomainID {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("client id %s has invalid domain id", thID))
		}
	}

	conns := []Connection{}
	cliConns := []*grpcCommonV1.Connection{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			for _, connType := range connTypes {
				conns = append(conns, Connection{
					ClientID:  thID,
					ChannelID: chID,
					DomainID:  session.DomainID,
					Type:      connType,
				})
				cliConns = append(cliConns, &grpcCommonV1.Connection{
					ClientId:  thID,
					ChannelId: chID,
					DomainId:  session.DomainID,
					Type:      uint32(connType),
				})
			}
		}
	}
	for _, conn := range conns {
		err := svc.repo.CheckConnection(ctx, conn)

		switch {
		case err == nil:
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("channel %s and client %s are already connected for type %s in domain %s ", conn.ChannelID, conn.ClientID, conn.Type.String(), conn.DomainID))
		case err != repoerr.ErrNotFound:
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
	}
	if _, err := svc.clients.AddConnections(ctx, &grpcCommonV1.AddConnectionsReq{Connections: cliConns}); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, errors.Wrap(errAddConnectionsClients, err))
	}

	if err := svc.repo.AddConnections(ctx, conns); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (svc service) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) (retErr error) {
	for _, chID := range chIDs {
		c, err := svc.repo.RetrieveByID(ctx, chID)
		if err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
		if c.Domain != session.DomainID {
			return errors.Wrap(svcerr.ErrRemoveEntity, fmt.Errorf("channel id %s has invalid domain id", chID))
		}
	}

	for _, thID := range thIDs {
		resp, err := svc.clients.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: thID})
		if err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}

		if resp.GetEntity().GetDomainId() != session.DomainID {
			return errors.Wrap(svcerr.ErrRemoveEntity, fmt.Errorf("client id %s has invalid domain id", thID))
		}
	}

	conns := []Connection{}
	thConns := []*grpcCommonV1.Connection{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			for _, connType := range connTypes {
				conns = append(conns, Connection{
					ClientID:  thID,
					ChannelID: chID,
					DomainID:  session.DomainID,
					Type:      connType,
				})
				thConns = append(thConns, &grpcCommonV1.Connection{
					ClientId:  thID,
					ChannelId: chID,
					DomainId:  session.DomainID,
					Type:      uint32(connType),
				})
			}
		}
	}

	if _, err := svc.clients.RemoveConnections(ctx, &grpcCommonV1.RemoveConnectionsReq{Connections: thConns}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, errors.Wrap(errRemoveConnectionsClients, err))
	}

	if err := svc.repo.RemoveConnections(ctx, conns); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (retErr error) {
	ch, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	resp, err := svc.groups.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: parentGroupID})
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if resp.GetEntity().GetDomainId() != session.DomainID {
		return errors.Wrap(svcerr.ErrUpdateEntity, fmt.Errorf("parent group id %s has invalid domain id", parentGroupID))
	}
	if resp.GetEntity().GetStatus() != uint32(smqclients.EnabledStatus) {
		return errors.Wrap(svcerr.ErrUpdateEntity, fmt.Errorf("parent group id %s is not in enabled state", parentGroupID))
	}

	var pols []policies.Policy
	if ch.ParentGroup != "" {
		return errors.Wrap(svcerr.ErrConflict, errSetParentGroup)
	}
	pols = append(pols, policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     parentGroupID,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.ChannelType,
		Object:      id,
	})

	if err := svc.policy.AddPolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policy.DeletePolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()
	ch = Channel{ID: id, ParentGroup: parentGroupID, UpdatedBy: session.UserID, UpdatedAt: time.Now()}

	if err := svc.repo.SetParentGroup(ctx, ch); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (svc service) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (retErr error) {
	ch, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	if ch.ParentGroup != "" {
		var pols []policies.Policy
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     ch.ParentGroup,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.ChannelType,
			Object:      id,
		})

		if err := svc.policy.DeletePolicies(ctx, pols); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
		defer func() {
			if retErr != nil {
				if errRollback := svc.policy.AddPolicies(ctx, pols); errRollback != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
				}
			}
		}()

		ch := Channel{ID: id, UpdatedBy: session.UserID, UpdatedAt: time.Now()}

		if err := svc.repo.RemoveParentGroup(ctx, ch); err != nil {
			return err
		}
	}

	return nil
}

func (svc service) changeChannelStatus(ctx context.Context, userID string, channel Channel) (Channel, error) {
	dbchannel, err := svc.repo.RetrieveByID(ctx, channel.ID)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbchannel.Status == channel.Status {
		return Channel{}, errors.ErrStatusAlreadyAssigned
	}

	channel.UpdatedBy = userID

	channel, err = svc.repo.ChangeStatus(ctx, channel)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}
