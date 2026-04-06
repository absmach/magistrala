// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	gstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defRetrieveAllLimit = 1000

var (
	errAddPolicies      = errors.New("failed to add policies")
	errRetrievePolicies = errors.New("failed to retrieve policies")
	errRemovePolicies   = errors.New("failed to remove the policies")
	errNoPolicies       = errors.New("no policies provided")
	errInternal         = errors.New("spicedb internal error")
	errPlatform         = errors.New("invalid platform id")
)

var (
	defClientsFilterPermissions = []string{
		policies.AdminPermission,
		policies.DeletePermission,
		policies.EditPermission,
		policies.ViewPermission,
		policies.SharePermission,
		policies.PublishPermission,
		policies.SubscribePermission,
	}

	defGroupsFilterPermissions = []string{
		policies.AdminPermission,
		policies.DeletePermission,
		policies.EditPermission,
		policies.ViewPermission,
		policies.MembershipPermission,
		policies.SharePermission,
	}

	defDomainsFilterPermissions = []string{
		policies.AdminPermission,
		policies.EditPermission,
		policies.ViewPermission,
		policies.MembershipPermission,
		policies.SharePermission,
	}

	defPlatformFilterPermissions = []string{
		policies.AdminPermission,
		policies.MembershipPermission,
	}
)

type policyService struct {
	client           *authzed.ClientWithExperimental
	permissionClient v1.PermissionsServiceClient
	logger           *slog.Logger
}

func NewPolicyService(client *authzed.ClientWithExperimental, logger *slog.Logger) policies.Service {
	return &policyService{
		client:           client,
		permissionClient: client.PermissionsServiceClient,
		logger:           logger,
	}
}

func (ps *policyService) AddPolicy(ctx context.Context, pr policies.Policy) error {
	if err := ps.policyValidation(pr); err != nil {
		return errors.Wrap(svcerr.ErrInvalidPolicy, err)
	}
	precond, err := ps.addPolicyPreCondition(ctx, pr)
	if err != nil {
		return err
	}

	updates := []*v1.RelationshipUpdate{
		{
			Operation: v1.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
				Relation: pr.Relation,
				Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
			},
		},
	}
	_, err = ps.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates, OptionalPreconditions: precond})
	if err != nil {
		return errors.Wrap(errAddPolicies, handleSpicedbError(err))
	}

	return nil
}

func (ps *policyService) AddPolicies(ctx context.Context, prs []policies.Policy) error {
	updates := []*v1.RelationshipUpdate{}
	var preconds []*v1.Precondition
	for _, pr := range prs {
		if err := ps.policyValidation(pr); err != nil {
			return errors.Wrap(svcerr.ErrInvalidPolicy, err)
		}
		precond, err := ps.addPolicyPreCondition(ctx, pr)
		if err != nil {
			return err
		}
		preconds = append(preconds, precond...)
		updates = append(updates, &v1.RelationshipUpdate{
			Operation: v1.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
				Relation: pr.Relation,
				Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
			},
		})
	}
	if len(updates) == 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errNoPolicies)
	}
	_, err := ps.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates, OptionalPreconditions: preconds})
	if err != nil {
		return errors.Wrap(errAddPolicies, handleSpicedbError(err))
	}

	return nil
}

func (ps *policyService) DeletePolicyFilter(ctx context.Context, pr policies.Policy) error {
	req := &v1.DeleteRelationshipsRequest{
		RelationshipFilter: &v1.RelationshipFilter{
			ResourceType:             pr.ObjectType,
			OptionalResourceId:       pr.Object,
			OptionalResourceIdPrefix: pr.ObjectPrefix,
		},
	}

	if pr.Relation != "" {
		req.RelationshipFilter.OptionalRelation = pr.Relation
	}

	if pr.SubjectType != "" {
		req.RelationshipFilter.OptionalSubjectFilter = &v1.SubjectFilter{
			SubjectType: pr.SubjectType,
		}
		if pr.Subject != "" {
			req.RelationshipFilter.OptionalSubjectFilter.OptionalSubjectId = pr.Subject
		}
		if pr.SubjectRelation != "" {
			req.RelationshipFilter.OptionalSubjectFilter.OptionalRelation = &v1.SubjectFilter_RelationFilter{
				Relation: pr.SubjectRelation,
			}
		}
	}

	if _, err := ps.permissionClient.DeleteRelationships(ctx, req); err != nil {
		return errors.Wrap(errRemovePolicies, handleSpicedbError(err))
	}

	return nil
}

func (ps *policyService) DeletePolicies(ctx context.Context, prs []policies.Policy) error {
	updates := []*v1.RelationshipUpdate{}
	for _, pr := range prs {
		if err := ps.policyValidation(pr); err != nil {
			return errors.Wrap(svcerr.ErrInvalidPolicy, err)
		}
		updates = append(updates, &v1.RelationshipUpdate{
			Operation: v1.RelationshipUpdate_OPERATION_DELETE,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
				Relation: pr.Relation,
				Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
			},
		})
	}
	if len(updates) == 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errNoPolicies)
	}
	_, err := ps.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
	if err != nil {
		return errors.Wrap(errRemovePolicies, handleSpicedbError(err))
	}

	return nil
}

func (ps *policyService) ListObjects(ctx context.Context, pr policies.Policy, nextPageToken string, limit uint64) (policies.PolicyPage, error) {
	if limit <= 0 {
		limit = 100
	}
	res, npt, err := ps.retrieveObjects(ctx, pr, nextPageToken, limit)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	var page policies.PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Object)
	}
	page.NextPageToken = npt

	return page, nil
}

func (ps *policyService) ListAllObjects(ctx context.Context, pr policies.Policy) (policies.PolicyPage, error) {
	res, err := ps.retrieveAllObjects(ctx, pr)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	var page policies.PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Object)
	}

	return page, nil
}

func (ps *policyService) CountObjects(ctx context.Context, pr policies.Policy) (uint64, error) {
	var count uint64
	nextPageToken := ""
	for {
		relationTuples, npt, err := ps.retrieveObjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
		if err != nil {
			return count, err
		}
		count = count + uint64(len(relationTuples))
		if npt == "" {
			break
		}
		nextPageToken = npt
	}

	return count, nil
}

func (ps *policyService) ListSubjects(ctx context.Context, pr policies.Policy, nextPageToken string, limit uint64) (policies.PolicyPage, error) {
	if limit <= 0 {
		limit = 100
	}
	res, npt, err := ps.retrieveSubjects(ctx, pr, nextPageToken, limit)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	var page policies.PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Subject)
	}
	page.NextPageToken = npt

	return page, nil
}

func (ps *policyService) ListAllSubjects(ctx context.Context, pr policies.Policy) (policies.PolicyPage, error) {
	res, err := ps.retrieveAllSubjects(ctx, pr)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	var page policies.PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Subject)
	}

	return page, nil
}

func (ps *policyService) CountSubjects(ctx context.Context, pr policies.Policy) (uint64, error) {
	var count uint64
	nextPageToken := ""
	for {
		relationTuples, npt, err := ps.retrieveSubjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
		if err != nil {
			return count, err
		}
		count = count + uint64(len(relationTuples))
		if npt == "" {
			break
		}
		nextPageToken = npt
	}

	return count, nil
}

func (ps *policyService) ListPermissions(ctx context.Context, pr policies.Policy, permissionsFilter []string) (policies.Permissions, error) {
	if len(permissionsFilter) == 0 {
		switch pr.ObjectType {
		case policies.ClientType:
			permissionsFilter = defClientsFilterPermissions
		case policies.GroupType:
			permissionsFilter = defGroupsFilterPermissions
		case policies.PlatformType:
			permissionsFilter = defPlatformFilterPermissions
		case policies.DomainType:
			permissionsFilter = defDomainsFilterPermissions
		default:
			return nil, svcerr.ErrMalformedEntity
		}
	}
	pers, err := ps.retrievePermissions(ctx, pr, permissionsFilter)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return pers, nil
}

func (ps *policyService) policyValidation(pr policies.Policy) error {
	if pr.ObjectType == policies.PlatformType && pr.Object != policies.MagistralaObject {
		return errPlatform
	}

	return nil
}

func (ps *policyService) addPolicyPreCondition(ctx context.Context, pr policies.Policy) ([]*v1.Precondition, error) {
	// Checks are required for following  ( -> means adding)
	// 1.) user -> group (both user groups and channels)
	// 2.) user -> client
	// 3.) group -> group (both for adding parent_group and channels)
	// 4.) group (channel) -> client
	// 5.) user -> domain

	switch {
	// 1.) user -> group (both user groups and channels)
	// Checks :
	// - USER with ANY RELATION to DOMAIN
	// - GROUP with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == policies.UserType && pr.ObjectType == policies.GroupType:
		return ps.userGroupPreConditions(ctx, pr)

	// 2.) user -> client
	// Checks :
	// - USER with ANY RELATION to DOMAIN
	// - CLIENT with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == policies.UserType && pr.ObjectType == policies.ClientType:
		return ps.userClientPreConditions(ctx, pr)

	// 3.) group -> group (both for adding parent_group and channels)
	// Checks :
	// - CHILD_GROUP with out PARENT_GROUP RELATION with any GROUP
	case pr.SubjectType == policies.GroupType && pr.ObjectType == policies.GroupType:
		return groupPreConditions(pr)

	// 4.) group (channel) -> client
	// Checks :
	// - GROUP (channel) with DOMAIN RELATION to DOMAIN
	// - NO GROUP should not have PARENT_GROUP RELATION with GROUP (channel)
	// - CLIENT with DOMAIN RELATION to DOMAIN
	// case pr.SubjectType == policies.GroupType && pr.ObjectType == policies.ClientType:
	// 	return channelClientPreCondition(pr)

	// 5.) user -> domain
	// Checks :
	// - User doesn't have any relation with domain
	case pr.SubjectType == policies.UserType && pr.ObjectType == policies.DomainType:
		return ps.userDomainPreConditions(ctx, pr)

	// Check client and group not belongs to other domain before adding to domain
	case pr.SubjectType == policies.DomainType && pr.Relation == policies.DomainRelation && (pr.ObjectType == policies.ClientType || pr.ObjectType == policies.GroupType):
		preconds := []*v1.Precondition{
			{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       pr.ObjectType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: policies.DomainType,
					},
				},
			},
		}
		return preconds, nil
	}

	return nil, nil
}

func (ps *policyService) userGroupPreConditions(ctx context.Context, pr policies.Policy) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	// user should not have any relation with group
	preconds = append(preconds, &v1.Precondition{
		Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
		Filter: &v1.RelationshipFilter{
			ResourceType:       policies.GroupType,
			OptionalResourceId: pr.Object,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       policies.UserType,
				OptionalSubjectId: pr.Subject,
			},
		},
	})
	isSuperAdmin := false
	if err := ps.checkPolicy(ctx, policies.Policy{
		Subject:     pr.Subject,
		SubjectType: pr.SubjectType,
		Permission:  policies.AdminPermission,
		Object:      policies.MagistralaObject,
		ObjectType:  policies.PlatformType,
	}); err == nil {
		isSuperAdmin = true
	}

	if !isSuperAdmin {
		preconds = append(preconds, &v1.Precondition{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       policies.DomainType,
				OptionalResourceId: pr.Domain,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       policies.UserType,
					OptionalSubjectId: pr.Subject,
				},
			},
		})
	}
	switch {
	case pr.ObjectKind == policies.NewGroupKind || pr.ObjectKind == policies.NewChannelKind:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: policies.DomainType,
					},
				},
			},
		)
	default:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType:       policies.DomainType,
						OptionalSubjectId: pr.Domain,
					},
				},
			},
		)
	}

	return preconds, nil
}

func (ps *policyService) userClientPreConditions(ctx context.Context, pr policies.Policy) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	// user should not have any relation with client
	preconds = append(preconds, &v1.Precondition{
		Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
		Filter: &v1.RelationshipFilter{
			ResourceType:       policies.ClientType,
			OptionalResourceId: pr.Object,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       policies.UserType,
				OptionalSubjectId: pr.Subject,
			},
		},
	})

	isSuperAdmin := false
	if err := ps.checkPolicy(ctx, policies.Policy{
		Subject:     pr.Subject,
		SubjectType: pr.SubjectType,
		Permission:  policies.AdminPermission,
		Object:      policies.MagistralaObject,
		ObjectType:  policies.PlatformType,
	}); err == nil {
		isSuperAdmin = true
	}

	if !isSuperAdmin {
		preconds = append(preconds, &v1.Precondition{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       policies.DomainType,
				OptionalResourceId: pr.Domain,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       policies.UserType,
					OptionalSubjectId: pr.Subject,
				},
			},
		})
	}
	switch {
	// For New client
	// - CLIENT without DOMAIN RELATION to ANY DOMAIN
	case pr.ObjectKind == policies.NewClientKind:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.ClientType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: policies.DomainType,
					},
				},
			},
		)
	default:
		// For existing client
		// - CLIENT without DOMAIN RELATION to ANY DOMAIN
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.ClientType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType:       policies.DomainType,
						OptionalSubjectId: pr.Domain,
					},
				},
			},
		)
	}

	return preconds, nil
}

func (ps *policyService) userDomainPreConditions(ctx context.Context, pr policies.Policy) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	if err := ps.checkPolicy(ctx, policies.Policy{
		Subject:     pr.Subject,
		SubjectType: pr.SubjectType,
		Permission:  policies.AdminPermission,
		Object:      policies.MagistralaObject,
		ObjectType:  policies.PlatformType,
	}); err == nil {
		return preconds, fmt.Errorf("use already exists in domain")
	}

	// user should not have any relation with domain.
	preconds = append(preconds, &v1.Precondition{
		Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
		Filter: &v1.RelationshipFilter{
			ResourceType:       policies.DomainType,
			OptionalResourceId: pr.Object,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       policies.UserType,
				OptionalSubjectId: pr.Subject,
			},
		},
	})

	return preconds, nil
}

func (ps *policyService) checkPolicy(ctx context.Context, pr policies.Policy) error {
	checkReq := v1.CheckPermissionRequest{
		// FullyConsistent means little caching will be available, which means performance will suffer.
		// Only use if a ZedToken is not available or absolutely latest information is required.
		// If we want to avoid FullyConsistent and to improve the performance of  spicedb, then we need to cache the ZEDTOKEN whenever RELATIONS is created or updated.
		// Instead of using FullyConsistent we need to use Consistency_AtLeastAsFresh, code looks like below one.
		// Consistency: &v1.Consistency{
		// 	Requirement: &v1.Consistency_AtLeastAsFresh{
		// 		AtLeastAsFresh: getRelationTupleZedTokenFromCache() ,
		// 	}
		// },
		// Reference: https://authzed.com/docs/reference/api-consistency
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		Resource:   &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
		Permission: pr.Permission,
		Subject:    &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
	}

	resp, err := ps.permissionClient.CheckPermission(ctx, &checkReq)
	if err != nil {
		return handleSpicedbError(err)
	}
	if resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return nil
	}
	if reason, ok := v1.CheckPermissionResponse_Permissionship_name[int32(resp.Permissionship)]; ok {
		return errors.Wrap(svcerr.ErrAuthorization, errors.New(reason))
	}
	return svcerr.ErrAuthorization
}

func (ps *policyService) retrieveObjects(ctx context.Context, pr policies.Policy, nextPageToken string, limit uint64) ([]policies.Policy, string, error) {
	resourceReq := &v1.LookupResourcesRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		ResourceObjectType: pr.ObjectType,
		Permission:         pr.Permission,
		Subject:            &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
		OptionalLimit:      uint32(limit),
	}
	if nextPageToken != "" {
		resourceReq.OptionalCursor = &v1.Cursor{Token: nextPageToken}
	}
	stream, err := ps.permissionClient.LookupResources(ctx, resourceReq)
	if err != nil {
		return nil, "", errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}
	resources := []*v1.LookupResourcesResponse{}
	var token string
	for {
		resp, err := stream.Recv()
		switch err {
		case nil:
			resources = append(resources, resp)
		case io.EOF:
			if len(resources) > 0 && resources[len(resources)-1].AfterResultCursor != nil {
				token = resources[len(resources)-1].AfterResultCursor.Token
			}
			return objectsToAuthPolicies(resources), token, nil
		default:
			if len(resources) > 0 && resources[len(resources)-1].AfterResultCursor != nil {
				token = resources[len(resources)-1].AfterResultCursor.Token
			}
			return []policies.Policy{}, token, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		}
	}
}

func (ps *policyService) retrieveAllObjects(ctx context.Context, pr policies.Policy) ([]policies.Policy, error) {
	resourceReq := &v1.LookupResourcesRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		ResourceObjectType: pr.ObjectType,
		Permission:         pr.Permission,
		Subject:            &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
	}
	stream, err := ps.permissionClient.LookupResources(ctx, resourceReq)
	if err != nil {
		return nil, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}
	tuples := []policies.Policy{}
	for {
		resp, err := stream.Recv()
		switch {
		case errors.Contains(err, io.EOF):
			return tuples, nil
		case err != nil:
			return tuples, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		default:
			tuples = append(tuples, policies.Policy{Object: resp.ResourceObjectId})
		}
	}
}

func (ps *policyService) retrieveSubjects(ctx context.Context, pr policies.Policy, nextPageToken string, limit uint64) ([]policies.Policy, string, error) {
	subjectsReq := v1.LookupSubjectsRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		Resource:                &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
		Permission:              pr.Permission,
		SubjectObjectType:       pr.SubjectType,
		OptionalSubjectRelation: pr.SubjectRelation,
		OptionalConcreteLimit:   uint32(limit),
		WildcardOption:          v1.LookupSubjectsRequest_WILDCARD_OPTION_INCLUDE_WILDCARDS,
	}
	if nextPageToken != "" {
		subjectsReq.OptionalCursor = &v1.Cursor{Token: nextPageToken}
	}
	stream, err := ps.permissionClient.LookupSubjects(ctx, &subjectsReq)
	if err != nil {
		return nil, "", errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}
	subjects := []*v1.LookupSubjectsResponse{}
	var token string
	for {
		resp, err := stream.Recv()

		switch err {
		case nil:
			subjects = append(subjects, resp)
		case io.EOF:
			if len(subjects) > 0 && subjects[len(subjects)-1].AfterResultCursor != nil {
				token = subjects[len(subjects)-1].AfterResultCursor.Token
			}
			return subjectsToAuthPolicies(subjects), token, nil
		default:
			if len(subjects) > 0 && subjects[len(subjects)-1].AfterResultCursor != nil {
				token = subjects[len(subjects)-1].AfterResultCursor.Token
			}
			return []policies.Policy{}, token, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		}
	}
}

func (ps *policyService) retrieveAllSubjects(ctx context.Context, pr policies.Policy) ([]policies.Policy, error) {
	var tuples []policies.Policy
	nextPageToken := ""
	for i := 0; ; i++ {
		relationTuples, npt, err := ps.retrieveSubjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
		if err != nil {
			return tuples, err
		}
		tuples = append(tuples, relationTuples...)
		if npt == "" || (len(tuples) < defRetrieveAllLimit) {
			break
		}
		nextPageToken = npt
	}
	return tuples, nil
}

func (ps *policyService) retrievePermissions(ctx context.Context, pr policies.Policy, filterPermission []string) (policies.Permissions, error) {
	var permissionChecks []*v1.CheckBulkPermissionsRequestItem
	for _, fp := range filterPermission {
		permissionChecks = append(permissionChecks, &v1.CheckBulkPermissionsRequestItem{
			Resource: &v1.ObjectReference{
				ObjectType: pr.ObjectType,
				ObjectId:   pr.Object,
			},
			Permission: fp,
			Subject: &v1.SubjectReference{
				Object: &v1.ObjectReference{
					ObjectType: pr.SubjectType,
					ObjectId:   pr.Subject,
				},
				OptionalRelation: pr.SubjectRelation,
			},
		})
	}
	resp, err := ps.client.PermissionsServiceClient.CheckBulkPermissions(ctx, &v1.CheckBulkPermissionsRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		Items: permissionChecks,
	})
	if err != nil {
		return policies.Permissions{}, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}

	permissions := []string{}
	for _, pair := range resp.Pairs {
		if pair.GetError() != nil {
			s := pair.GetError()
			return policies.Permissions{}, errors.Wrap(errRetrievePolicies, convertGRPCStatusToError(convertToGrpcStatus(s)))
		}
		item := pair.GetItem()
		req := pair.GetRequest()
		if item != nil && req != nil && item.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
			permissions = append(permissions, req.GetPermission())
		}
	}
	return permissions, nil
}

func groupPreConditions(pr policies.Policy) ([]*v1.Precondition, error) {
	// - PARENT_GROUP (subject) with DOMAIN RELATION to DOMAIN
	precond := []*v1.Precondition{
		{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       policies.GroupType,
				OptionalResourceId: pr.Subject,
				OptionalRelation:   policies.DomainRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       policies.DomainType,
					OptionalSubjectId: pr.Domain,
				},
			},
		},
	}
	if pr.ObjectKind != policies.ChannelsKind {
		precond = append(precond,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.ParentGroupRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: policies.GroupType,
					},
				},
			},
		)
	}
	switch {
	// - NEW CHILD_GROUP (object) with out DOMAIN RELATION to ANY DOMAIN
	case pr.ObjectType == policies.GroupType && pr.ObjectKind == policies.NewGroupKind:
		precond = append(precond,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: policies.DomainType,
					},
				},
			},
		)
	default:
		// - CHILD_GROUP (object) with DOMAIN RELATION to DOMAIN
		precond = append(precond,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType:       policies.DomainType,
						OptionalSubjectId: pr.Domain,
					},
				},
			},
		)
	}
	return precond, nil
}

func objectsToAuthPolicies(objects []*v1.LookupResourcesResponse) []policies.Policy {
	var policyList []policies.Policy
	for _, obj := range objects {
		policyList = append(policyList, policies.Policy{
			Object: obj.GetResourceObjectId(),
		})
	}
	return policyList
}

func subjectsToAuthPolicies(subjects []*v1.LookupSubjectsResponse) []policies.Policy {
	var policyList []policies.Policy
	for _, sub := range subjects {
		policyList = append(policyList, policies.Policy{
			Subject: sub.Subject.GetSubjectObjectId(),
		})
	}
	return policyList
}

func handleSpicedbError(err error) error {
	if st, ok := status.FromError(err); ok {
		return convertGRPCStatusToError(st)
	}
	return err
}

func convertToGrpcStatus(gst *gstatus.Status) *status.Status {
	st := status.New(codes.Code(gst.Code), gst.GetMessage())
	return st
}

func convertGRPCStatusToError(st *status.Status) error {
	switch st.Code() {
	case codes.NotFound:
		return errors.Wrap(repoerr.ErrNotFound, errors.New(st.Message()))
	case codes.InvalidArgument:
		return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
	case codes.AlreadyExists:
		return errors.Wrap(repoerr.ErrConflict, errors.New(st.Message()))
	case codes.Unauthenticated:
		return errors.Wrap(svcerr.ErrAuthentication, errors.New(st.Message()))
	case codes.Internal:
		return errors.Wrap(errInternal, errors.New(st.Message()))
	case codes.OK:
		if msg := st.Message(); msg != "" {
			return errors.Wrap(errors.ErrUnidentified, errors.New(msg))
		}
		return nil
	case codes.FailedPrecondition:
		return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
	case codes.PermissionDenied:
		return errors.Wrap(svcerr.ErrAuthorization, errors.New(st.Message()))
	default:
		return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
	}
}
