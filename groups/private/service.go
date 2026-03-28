// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

const defLimit = 100

type Service interface {
	RetrieveById(ctx context.Context, id string) (groups.Group, error)
	DeleteDomainGroups(ctx context.Context, domainID string) error
}

var _ Service = (*service)(nil)

func New(repo groups.Repository, policy policies.Service) Service {
	return service{repo, policy}
}

type service struct {
	repo   groups.Repository
	policy policies.Service
}

func (svc service) RetrieveById(ctx context.Context, ids string) (groups.Group, error) {
	return svc.repo.RetrieveByID(ctx, ids)
}

func (svc service) DeleteDomainGroups(ctx context.Context, domainID string) error {
	groupsPage, err := svc.repo.RetrieveAll(ctx, groups.PageMeta{DomainID: domainID, Limit: defLimit})
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if groupsPage.Total > defLimit {
		for i := defLimit; i < int(groupsPage.Total); i += defLimit {
			page := groups.PageMeta{DomainID: domainID, Offset: uint64(i), Limit: defLimit}
			gp, err := svc.repo.RetrieveAll(ctx, page)
			if err != nil {
				return errors.Wrap(svcerr.ErrViewEntity, err)
			}
			groupsPage.Groups = append(groupsPage.Groups, gp.Groups...)
		}
	}

	for _, group := range groupsPage.Groups {
		if err := svc.deleteGroupsPolicies(ctx, domainID, group.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
		if err := svc.repo.Delete(ctx, group.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (svc service) deleteGroupsPolicies(ctx context.Context, domainID, groupID string) error {
	ears, emrs, err := svc.repo.RetrieveEntitiesRolesActionsMembers(ctx, []string{groupID})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	deletePolicies := []policies.Policy{}
	for _, ear := range ears {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:         ear.RoleID,
			SubjectRelation: policies.MemberRelation,
			SubjectType:     policies.RoleType,
			Relation:        ear.Action,
			ObjectType:      policies.GroupType,
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
			SubjectType: policies.GroupType,
			Subject:     groupID,
		},
		{
			ObjectType: policies.GroupType,
			Object:     groupID,
		},
	}
	for _, filter := range filterDeletePolicies {
		if err := svc.policy.DeletePolicyFilter(ctx, filter); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
	}

	return nil
}
