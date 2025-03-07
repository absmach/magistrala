// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package clients

import (
	"context"
	"fmt"
	"time"

	smq "github.com/absmach/supermq"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	grpcGroupsV1 "github.com/absmach/supermq/api/grpc/groups/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
)

var (
	errRollbackRepo   = errors.New("failed to rollback repo")
	errSetParentGroup = errors.New("client already have parent")
)
var _ Service = (*service)(nil)

type service struct {
	repo       Repository
	policy     policies.Service
	channels   grpcChannelsV1.ChannelsServiceClient
	groups     grpcGroupsV1.GroupsServiceClient
	cache      Cache
	idProvider smq.IDProvider
	roles.ProvisionManageService
}

// NewService returns a new Clients service implementation.
func NewService(repo Repository, policy policies.Service, cache Cache, channels grpcChannelsV1.ChannelsServiceClient, groups grpcGroupsV1.GroupsServiceClient, idProvider smq.IDProvider, sIDProvider smq.IDProvider, availableActions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (Service, error) {
	rpms, err := roles.NewProvisionManageService(policies.ClientType, repo, policy, sIDProvider, availableActions, builtInRoles)
	if err != nil {
		return service{}, err
	}
	return service{
		repo:                   repo,
		policy:                 policy,
		channels:               channels,
		groups:                 groups,
		cache:                  cache,
		idProvider:             idProvider,
		ProvisionManageService: rpms,
	}, nil
}

func (svc service) CreateClients(ctx context.Context, session authn.Session, cls ...Client) (retClients []Client, retRps []roles.RoleProvision, retErr error) {
	var clients []Client
	for _, c := range cls {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []Client{}, []roles.RoleProvision{}, err
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []Client{}, []roles.RoleProvision{}, err
			}
			c.Credentials.Secret = key
		}
		if c.Status != DisabledStatus && c.Status != EnabledStatus {
			return []Client{}, []roles.RoleProvision{}, svcerr.ErrInvalidStatus
		}
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	newClients, err := svc.repo.Save(ctx, clients...)
	if err != nil {
		return []Client{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	newClientIDs := []string{}
	for _, newClient := range newClients {
		newClientIDs = append(newClientIDs, newClient.ID)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := svc.repo.Delete(ctx, newClientIDs...); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []policies.Policy{}

	for _, newClientID := range newClientIDs {
		optionalPolicies = append(optionalPolicies,
			policies.Policy{
				Domain:      session.DomainID,
				SubjectType: policies.DomainType,
				Subject:     session.DomainID,
				Relation:    policies.DomainRelation,
				ObjectType:  policies.ClientType,
				Object:      newClientID,
			},
		)
	}

	nrps, err := svc.AddNewEntitiesRoles(ctx, session.DomainID, session.UserID, newClientIDs, optionalPolicies, newBuiltInRoleMembers)
	if err != nil {
		return []Client{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return newClients, nrps, nil
}

func (svc service) View(ctx context.Context, session authn.Session, id string) (Client, error) {
	client, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return client, nil
}

func (svc service) ListClients(ctx context.Context, session authn.Session, pm Page) (ClientsPage, error) {
	switch session.SuperAdmin {
	case true:
		cp, err := svc.repo.RetrieveAll(ctx, pm)
		if err != nil {
			return ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		return cp, nil
	default:
		cp, err := svc.repo.RetrieveUserClients(ctx, session.DomainID, session.UserID, pm)
		if err != nil {
			return ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		return cp, nil
	}
}

func (svc service) ListUserClients(ctx context.Context, session authn.Session, userID string, pm Page) (ClientsPage, error) {
	cp, err := svc.repo.RetrieveUserClients(ctx, session.DomainID, userID, pm)
	if err != nil {
		return ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return cp, nil
}

func (svc service) Update(ctx context.Context, session authn.Session, cli Client) (Client, error) {
	client := Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.repo.Update(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateTags(ctx context.Context, session authn.Session, cli Client) (Client, error) {
	client := Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.repo.UpdateTags(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (Client, error) {
	client := Client{
		ID: id,
		Credentials: Credentials{
			Secret: key,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    EnabledStatus,
	}
	client, err := svc.repo.UpdateSecret(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) Enable(ctx context.Context, session authn.Session, id string) (Client, error) {
	client := Client{
		ID:        id,
		Status:    EnabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return Client{}, errors.Wrap(ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) Disable(ctx context.Context, session authn.Session, id string) (Client, error) {
	client := Client{
		ID:        id,
		Status:    DisabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return Client{}, errors.Wrap(ErrDisableClient, err)
	}

	if err := svc.cache.Remove(ctx, client.ID); err != nil {
		return client, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return client, nil
}

func (svc service) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (retErr error) {
	cli, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	switch cli.ParentGroup {
	case parentGroupID:
		return nil
	case "":
		// No action needed, proceed to next code after switch
	default:
		return errors.Wrap(svcerr.ErrConflict, errSetParentGroup)
	}

	resp, err := svc.groups.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: parentGroupID})
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if resp.GetEntity().GetDomainId() != session.DomainID {
		return errors.Wrap(svcerr.ErrUpdateEntity, fmt.Errorf("parent group id %s has invalid domain id", parentGroupID))
	}
	if resp.GetEntity().GetStatus() != uint32(EnabledStatus) {
		return errors.Wrap(svcerr.ErrUpdateEntity, fmt.Errorf("parent group id %s is not in enabled state", parentGroupID))
	}

	var pols []policies.Policy

	pols = append(pols, policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     parentGroupID,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.ClientType,
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
	cli = Client{ID: id, ParentGroup: parentGroupID, UpdatedBy: session.UserID, UpdatedAt: time.Now()}

	if err := svc.repo.SetParentGroup(ctx, cli); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (svc service) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (retErr error) {
	cli, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if cli.ParentGroup != "" {
		var pols []policies.Policy
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     cli.ParentGroup,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.ClientType,
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

		cli := Client{ID: id, UpdatedBy: session.UserID, UpdatedAt: time.Now()}

		if err := svc.repo.RemoveParentGroup(ctx, cli); err != nil {
			return err
		}
	}
	return nil
}

func (svc service) Delete(ctx context.Context, session authn.Session, id string) error {
	ok, err := svc.repo.DoesClientHaveConnections(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	if ok {
		if _, err := svc.channels.RemoveClientConnections(ctx, &grpcChannelsV1.RemoveClientConnectionsReq{ClientId: id}); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}

	if _, err := svc.repo.ChangeStatus(ctx, Client{ID: id, Status: DeletedStatus}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := svc.cache.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	filterDeletePolicies := []policies.Policy{
		{
			SubjectType: policies.ClientType,
			Subject:     id,
		},
		{
			ObjectType: policies.ClientType,
			Object:     id,
		},
	}
	deletePolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     session.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ClientType,
			Object:      id,
		},
	}

	if err := svc.RemoveEntitiesRoles(ctx, session.DomainID, session.DomainUserID, []string{id}, filterDeletePolicies, deletePolicies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if err := svc.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) changeClientStatus(ctx context.Context, session authn.Session, client Client) (Client, error) {
	dbClient, err := svc.repo.RetrieveByID(ctx, client.ID)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == client.Status {
		return Client{}, errors.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = session.UserID

	client, err = svc.repo.ChangeStatus(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}
