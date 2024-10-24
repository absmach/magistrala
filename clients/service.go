// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package clients

import (
	"context"
	"fmt"
	"time"

	mg "github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/magistrala/internal/grpc/common/v1"
	grpcGroupsV1 "github.com/absmach/magistrala/internal/grpc/groups/v1"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	"golang.org/x/sync/errgroup"
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
	idProvider mg.IDProvider
	roles.ProvisionManageService
}

// NewService returns a new Clients service implementation.
func NewService(repo Repository, policy policies.Service, cache Cache, channels grpcChannelsV1.ChannelsServiceClient, groups grpcGroupsV1.GroupsServiceClient, idProvider mg.IDProvider, sIDProvider mg.IDProvider) (Service, error) {
	rpms, err := roles.NewProvisionManageService(policies.ClientType, repo, policy, sIDProvider, AvailableActions(), BuiltInRoles())
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

func (svc service) CreateClients(ctx context.Context, session authn.Session, cls ...Client) (retClients []Client, retErr error) {
	var clients []Client
	for _, c := range cls {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []Client{}, err
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []Client{}, err
			}
			c.Credentials.Secret = key
		}
		if c.Status != DisabledStatus && c.Status != EnabledStatus {
			return []Client{}, svcerr.ErrInvalidStatus
		}
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	saved, err := svc.repo.Save(ctx, clients...)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	clientIDs := []string{}
	for _, c := range saved {
		clientIDs = append(clientIDs, c.ID)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := svc.repo.Delete(ctx, clientIDs...); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		ClientBuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []policies.Policy{}

	for _, clientID := range clientIDs {
		optionalPolicies = append(optionalPolicies,
			policies.Policy{
				Domain:      session.DomainID,
				SubjectType: policies.DomainType,
				Subject:     session.DomainID,
				Relation:    policies.DomainRelation,
				ObjectType:  policies.ClientType,
				Object:      clientID,
			},
		)
	}

	if _, err := svc.AddNewEntitiesRoles(ctx, session.DomainID, session.UserID, clientIDs, optionalPolicies, newBuiltInRoleMembers); err != nil {
		return []Client{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return saved, nil
}

func (svc service) View(ctx context.Context, session authn.Session, id string) (Client, error) {
	client, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return client, nil
}

func (svc service) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm Page) (ClientsPage, error) {
	var ids []string
	var err error
	switch {
	case (reqUserID != "" && reqUserID != session.UserID):
		rtids, err := svc.listClientIDs(ctx, mgauth.EncodeDomainUserID(session.DomainID, reqUserID), pm.Permission)
		if err != nil {
			return ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
		ids, err = svc.filterAllowedClientIDs(ctx, session.DomainUserID, pm.Permission, rtids)
		if err != nil {
			return ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	default:
		switch session.SuperAdmin {
		case true:
			pm.Domain = session.DomainID
		default:
			ids, err = svc.listClientIDs(ctx, session.DomainUserID, pm.Permission)
			if err != nil {
				return ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
			}
		}
	}

	if len(ids) == 0 && pm.Domain == "" {
		return ClientsPage{}, nil
	}
	pm.IDs = ids
	tp, err := svc.repo.SearchClients(ctx, pm)
	if err != nil {
		return ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if pm.ListPerms && len(tp.Clients) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range tp.Clients {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrievePermissions(ctx, session.DomainUserID, &tp.Clients[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return ClientsPage{}, err
		}
	}
	return tp, nil
}

// Experimental functions used for async calling of svc.listUserClientPermission. This might be helpful during listing of large number of entities.
func (svc service) retrievePermissions(ctx context.Context, userID string, client *Client) error {
	permissions, err := svc.listUserClientPermission(ctx, userID, client.ID)
	if err != nil {
		return err
	}
	client.Permissions = permissions
	return nil
}

func (svc service) listUserClientPermission(ctx context.Context, userID, clientID string) ([]string, error) {
	permissions, err := svc.policy.ListPermissions(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      clientID,
		ObjectType:  policies.ClientType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return permissions, nil
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.policy.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ClientType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (svc service) filterAllowedClientIDs(ctx context.Context, userID, permission string, clientIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.policy.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ClientType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	for _, clientID := range clientIDs {
		for _, tid := range tids.Policies {
			if clientID == tid {
				ids = append(ids, clientID)
			}
		}
	}
	return ids, nil
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
			return errors.Wrap(svcerr.ErrAddPolicies, err)
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
