// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/supermq/channels"
	dom "github.com/absmach/supermq/domains"
	pkgDomains "github.com/absmach/supermq/pkg/domains"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

const defLimit = 100

var errDisabledDomain = errors.New("domain is disabled or frozen")

type Service interface {
	Authorize(ctx context.Context, req channels.AuthzReq) error
	UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) error
	RemoveClientConnections(ctx context.Context, clientID string) error
	RetrieveByID(ctx context.Context, id string) (channels.Channel, error)
	RetrieveIDByRoute(ctx context.Context, route, domainID string) (string, error)
	DeleteDomainChannels(ctx context.Context, domainID string) error
}

type service struct {
	repo      channels.Repository
	cache     channels.Cache
	evaluator policies.Evaluator
	policy    policies.Service
	domains   pkgDomains.Authorization
}

var _ Service = (*service)(nil)

func New(repo channels.Repository, cache channels.Cache, evaluator policies.Evaluator, policy policies.Service, domains pkgDomains.Authorization) Service {
	return service{repo, cache, evaluator, policy, domains}
}

func (svc service) Authorize(ctx context.Context, req channels.AuthzReq) error {
	status, err := svc.domains.RetrieveStatus(ctx, req.DomainID)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if status != dom.EnabledStatus {
		return errors.Wrap(svcerr.ErrAuthorization, errDisabledDomain)
	}
	switch req.ClientType {
	case policies.UserType:
		permission, err := req.Type.Permission()
		if err != nil {
			return err
		}
		pr := policies.Policy{
			Subject:     req.ClientID,
			SubjectType: policies.UserType,
			Object:      req.ChannelID,
			Permission:  permission,
			ObjectType:  policies.ChannelType,
		}
		if err := svc.evaluator.CheckPolicy(ctx, pr); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return nil
	case policies.ClientType:
		// Optimization: Add cache
		if err := svc.repo.ClientAuthorize(ctx, channels.Connection{
			DomainID:  req.DomainID,
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

func (svc service) RetrieveIDByRoute(ctx context.Context, route, domainID string) (string, error) {
	id, err := svc.cache.ID(ctx, route, domainID)
	if err == nil {
		return id, nil
	}
	chn, err := svc.repo.RetrieveByRoute(ctx, route, domainID)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if err := svc.cache.Save(ctx, route, domainID, chn.ID); err != nil {
		return "", errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return chn.ID, nil
}

func (svc service) DeleteDomainChannels(ctx context.Context, domainID string) error {
	channelsPage, err := svc.repo.RetrieveAll(ctx, channels.Page{Domain: domainID, Limit: defLimit})
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if channelsPage.Total > defLimit {
		for i := defLimit; i < int(channelsPage.Total); i += defLimit {
			page := channels.Page{Domain: domainID, Offset: uint64(i), Limit: defLimit}
			cp, err := svc.repo.RetrieveAll(ctx, page)
			if err != nil {
				return err
			}
			channelsPage.Channels = append(channelsPage.Channels, cp.Channels...)
		}
	}

	for _, ch := range channelsPage.Channels {
		if ch.Route != "" {
			if err := svc.cache.Remove(ctx, ch.Route, ch.Domain); err != nil {
				return errors.Wrap(svcerr.ErrRemoveEntity, err)
			}
		}
		if err := svc.deleteChannelPolicies(ctx, domainID, ch.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
		if err := svc.repo.Remove(ctx, ch.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (svc service) deleteChannelPolicies(ctx context.Context, domainID, channelID string) error {
	ears, emrs, err := svc.repo.RetrieveEntitiesRolesActionsMembers(ctx, []string{channelID})
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
			ObjectType:      policies.ChannelType,
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
			SubjectType: policies.ChannelType,
			Subject:     channelID,
		},
		{
			ObjectType: policies.ChannelType,
			Object:     channelID,
		},
	}
	for _, filter := range filterDeletePolicies {
		if err := svc.policy.DeletePolicyFilter(ctx, filter); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
	}

	return nil
}
