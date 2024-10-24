package private

import (
	"context"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
)

//go:generate mockery --name Service  --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// Authenticate returns client ID for given client key.
	Authenticate(ctx context.Context, key string) (string, error)

	RetrieveById(ctx context.Context, id string) (clients.Client, error)

	RetrieveByIds(ctx context.Context, ids []string) (clients.ClientsPage, error)

	AddConnections(ctx context.Context, conns []clients.Connection) error

	RemoveConnections(ctx context.Context, conns []clients.Connection) error

	RemoveChannelConnections(ctx context.Context, channelID string) error

	UnsetParentGroupFromClient(ctx context.Context, parentGroupID string) error
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

func (svc service) Authenticate(ctx context.Context, key string) (string, error) {
	id, err := svc.cache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	client, err := svc.repo.RetrieveBySecret(ctx, key)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if err := svc.cache.Save(ctx, key, client.ID); err != nil {
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
