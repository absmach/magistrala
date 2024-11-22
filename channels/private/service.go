package private

import (
	"context"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
)

type Service interface {
	Authorize(ctx context.Context, req channels.AuthzReq) error
	UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) error
	RemoveClientConnections(ctx context.Context, clientID string) error
	RetrieveByID(ctx context.Context, id string) (channels.Channel, error)
}

type service struct {
	repo      channels.Repository
	evaluator policies.Evaluator
	policy    policies.Service
}

var _ Service = (*service)(nil)

func New(repo channels.Repository, evaluator policies.Evaluator, policy policies.Service) Service {
	return service{repo, evaluator, policy}
}

func (svc service) Authorize(ctx context.Context, req channels.AuthzReq) error {
	switch req.ClientType {
	case policies.UserType:
		pr := policies.Policy{
			Subject:     req.ClientID,
			SubjectType: policies.UserType,
			Object:      req.ChannelID,
			ObjectType:  policies.ChannelType,
		}
		if err := svc.evaluator.CheckPolicy(ctx, pr); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return nil
	case policies.ClientType:
		// Optimization: Add cache
		if err := svc.repo.ClientAuthorize(ctx, channels.Connection{
			ChannelID: req.ChannelID,
			ClientID:  req.ClientID,
			Type:      req.Type,
		}); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return nil
	default:
		return svcerr.ErrAuthentication
	}
}

func (svc service) RemoveClientConnections(ctx context.Context, clientID string) error {
	return svc.repo.RemoveClientConnections(ctx, clientID)
}

func (svc service) UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) (retErr error) {
	chs, err := svc.repo.RetrieveParentGroupChannels(ctx, parentGroupID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if len(chs) > 0 {
		prs := []policies.Policy{}
		for _, ch := range chs {
			prs = append(prs, policies.Policy{
				SubjectType: policies.GroupType,
				Subject:     ch.ParentGroup,
				Relation:    policies.ParentGroupRelation,
				ObjectType:  policies.ChannelType,
				Object:      ch.ID,
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

		if err := svc.repo.UnsetParentGroupFromChannels(ctx, parentGroupID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (svc service) RetrieveByID(ctx context.Context, id string) (channels.Channel, error) {
	return svc.repo.RetrieveByID(ctx, id)
}
