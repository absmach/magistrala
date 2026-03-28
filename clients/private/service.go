// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

const defLimit = 100

type Service interface {
	// Authenticate returns client ID for given client key.
	Authenticate(ctx context.Context, key string) (string, error)

	RetrieveById(ctx context.Context, id string) (clients.Client, error)

	RetrieveByIds(ctx context.Context, ids []string) (clients.ClientsPage, error)

	AddConnections(ctx context.Context, conns []clients.Connection) error

	RemoveConnections(ctx context.Context, conns []clients.Connection) error

	RemoveChannelConnections(ctx context.Context, channelID string) error

	UnsetParentGroupFromClient(ctx context.Context, parentGroupID string) error

	DeleteDomainClients(ctx context.Context, domainID string) error
}

var _ Service = (*service)(nil)

func New(repo clients.Repository, cache clients.Cache, evaluator policies.Evaluator, policy policies.Service) Service {
	return service{
		repo:      repo,
		cache:     cache,
		evaluator: evaluator,
		policy:    policy,
	}
}

type service struct {
	repo      clients.Repository
	cache     clients.Cache
	evaluator policies.Evaluator
	policy    policies.Service
}

func (svc service) Authenticate(ctx context.Context, token string) (string, error) {
	id, err := svc.cache.ID(ctx, token)
	if err == nil {
		return id, nil
	}
	prefix, id, key, err := authn.AuthUnpack(token)
	if err != nil && err != authn.ErrNotEncoded {
		return "", err
	}
	client, err := svc.repo.RetrieveBySecret(ctx, key, id, prefix)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if err := svc.cache.Save(ctx, token, client.ID); err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return client.ID, nil
}

func (svc service) RetrieveById(ctx context.Context, ids string) (clients.Client, error) {
	return svc.repo.RetrieveByID(ctx, ids)
}

func (svc service) RetrieveByIds(ctx context.Context, ids []string) (clients.ClientsPage, error) {
	return svc.repo.RetrieveByIds(ctx, ids)
}

func (svc service) AddConnections(ctx context.Context, conns []clients.Connection) (err error) {
	return svc.repo.AddConnections(ctx, conns)
}

func (svc service) RemoveConnections(ctx context.Context, conns []clients.Connection) (err error) {
	return svc.repo.RemoveConnections(ctx, conns)
}

func (svc service) RemoveChannelConnections(ctx context.Context, channelID string) error {
	return svc.repo.RemoveChannelConnections(ctx, channelID)
}

func (svc service) UnsetParentGroupFromClient(ctx context.Context, parentGroupID string) (retErr error) {
	clients, err := svc.repo.RetrieveParentGroupClients(ctx, parentGroupID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if len(clients) > 0 {
		prs := []policies.Policy{}
		for _, client := range clients {
			prs = append(prs, policies.Policy{
				SubjectType: policies.GroupType,
				Subject:     client.ParentGroup,
				Relation:    policies.ParentGroupRelation,
				ObjectType:  policies.ClientType,
				Object:      client.ID,
			})
		}

		if err := svc.policy.DeletePolicies(ctx, prs); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
		defer func() {
			if retErr != nil {
				if errRollback := svc.policy.AddPolicies(ctx, prs); err != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(errors.ErrRollbackTx, errRollback))
				}
			}
		}()

		if err := svc.repo.UnsetParentGroupFromClient(ctx, parentGroupID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (svc service) DeleteDomainClients(ctx context.Context, domainID string) error {
	clientsPage, err := svc.repo.RetrieveAll(ctx, clients.Page{Domain: domainID, Limit: defLimit})
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if clientsPage.Total > defLimit {
		for i := defLimit; i < int(clientsPage.Total); i += defLimit {
			page := clients.Page{Domain: domainID, Offset: uint64(i), Limit: defLimit}
			cp, err := svc.repo.RetrieveAll(ctx, page)
			if err != nil {
				return err
			}
			clientsPage.Clients = append(clientsPage.Clients, cp.Clients...)
		}
	}

	for _, client := range clientsPage.Clients {
		if err := svc.cache.Remove(ctx, client.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
		if err := svc.deleteClientPolicies(ctx, domainID, client.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
		if err := svc.repo.Delete(ctx, client.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (svc service) deleteClientPolicies(ctx context.Context, domainID, clientID string) error {
	ears, emrs, err := svc.repo.RetrieveEntitiesRolesActionsMembers(ctx, []string{clientID})
	if err != nil {
		return err
	}
	deletePolicies := []policies.Policy{}
	for _, ear := range ears {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:         ear.RoleID,
			SubjectRelation: policies.MemberRelation,
			SubjectType:     policies.RoleType,
			Relation:        ear.Action,
			ObjectType:      policies.ClientType,
			Object:          ear.EntityID,
		})
	}
	for _, emr := range emrs {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:     policies.EncodeDomainUserID(domainID, emr.MemberID),
			SubjectType: policies.UserType,
			Relation:    policies.MemberRelation,
			ObjectType:  policies.RoleType,
			Object:      emr.RoleID,
		})
	}
	if err := svc.policy.DeletePolicies(ctx, deletePolicies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	filterDeletePolicies := []policies.Policy{
		{
			SubjectType: policies.ClientType,
			Subject:     clientID,
		},
		{
			ObjectType: policies.ClientType,
			Object:     clientID,
		},
	}
	for _, filter := range filterDeletePolicies {
		if err := svc.policy.DeletePolicyFilter(ctx, filter); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
	}

	return nil
}
