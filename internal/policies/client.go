// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package policies

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
	errInvalidSubject   = errors.New("invalid subject kind")
	errAddPolicies      = errors.New("failed to add policies")
	errRetrievePolicies = errors.New("failed to retrieve policies")
	errRemovePolicies   = errors.New("failed to remove the policies")
	errNoPolicies       = errors.New("no policies provided")
	errInternal         = errors.New("spicedb internal error")
	errPlatform         = errors.New("invalid platform id")
)

var (
	defThingsFilterPermissions = []string{
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

type policyClient struct {
	client           *authzed.ClientWithExperimental
	permissionClient v1.PermissionsServiceClient
	logger           *slog.Logger
}

func NewPolicyClient(client *authzed.ClientWithExperimental, logger *slog.Logger) policies.PolicyClient {
	return &policyClient{
		client:           client,
		permissionClient: client.PermissionsServiceClient,
		logger:           logger,
	}
}

func (pc policyClient) AddPolicy(ctx context.Context, pr policies.PolicyReq) error {
	if err := pc.policyValidation(pr); err != nil {
		return errors.Wrap(svcerr.ErrInvalidPolicy, err)
	}
	precond, err := pc.addPolicyPreCondition(ctx, pr)
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
	_, err = pc.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates, OptionalPreconditions: precond})
	if err != nil {
		return errors.Wrap(errAddPolicies, handleSpicedbError(err))
	}

	return nil
}

func (pc policyClient) AddPolicies(ctx context.Context, prs []policies.PolicyReq) error {
	updates := []*v1.RelationshipUpdate{}
	var preconds []*v1.Precondition
	for _, pr := range prs {
		if err := pc.policyValidation(pr); err != nil {
			return errors.Wrap(svcerr.ErrInvalidPolicy, err)
		}
		precond, err := pc.addPolicyPreCondition(ctx, pr)
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
	_, err := pc.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates, OptionalPreconditions: preconds})
	if err != nil {
		return errors.Wrap(errAddPolicies, handleSpicedbError(err))
	}

	return nil
}

func (pc policyClient) DeletePolicyFilter(ctx context.Context, pr policies.PolicyReq) error {
	req := &v1.DeleteRelationshipsRequest{
		RelationshipFilter: &v1.RelationshipFilter{
			ResourceType:       pr.ObjectType,
			OptionalResourceId: pr.Object,
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

	if _, err := pc.permissionClient.DeleteRelationships(ctx, req); err != nil {
		return errors.Wrap(errRemovePolicies, handleSpicedbError(err))
	}

	return nil
}

func (pc policyClient) DeletePolicies(ctx context.Context, prs []policies.PolicyReq) error {
	updates := []*v1.RelationshipUpdate{}
	for _, pr := range prs {
		if err := pc.policyValidation(pr); err != nil {
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
	_, err := pc.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
	if err != nil {
		return errors.Wrap(errRemovePolicies, handleSpicedbError(err))
	}

	return nil
}

func (pc policyClient) ListObjects(ctx context.Context, pr policies.PolicyReq, nextPageToken string, limit uint64) (policies.PolicyPage, error) {
	if limit <= 0 {
		limit = 100
	}
	res, npt, err := pc.retrieveObjects(ctx, pr, nextPageToken, limit)
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

func (pc policyClient) ListAllObjects(ctx context.Context, pr policies.PolicyReq) (policies.PolicyPage, error) {
	res, err := pc.retrieveAllObjects(ctx, pr)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	var page policies.PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Object)
	}

	return page, nil
}

func (pc policyClient) CountObjects(ctx context.Context, pr policies.PolicyReq) (uint64, error) {
	var count uint64
	nextPageToken := ""
	for {
		relationTuples, npt, err := pc.retrieveObjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
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

func (pc policyClient) ListSubjects(ctx context.Context, pr policies.PolicyReq, nextPageToken string, limit uint64) (policies.PolicyPage, error) {
	if limit <= 0 {
		limit = 100
	}
	res, npt, err := pc.retrieveSubjects(ctx, pr, nextPageToken, limit)
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

func (pc policyClient) ListAllSubjects(ctx context.Context, pr policies.PolicyReq) (policies.PolicyPage, error) {
	res, err := pc.retrieveAllSubjects(ctx, pr)
	if err != nil {
		return policies.PolicyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	var page policies.PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Subject)
	}

	return page, nil
}

func (pc policyClient) CountSubjects(ctx context.Context, pr policies.PolicyReq) (uint64, error) {
	var count uint64
	nextPageToken := ""
	for {
		relationTuples, npt, err := pc.retrieveSubjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
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

func (pc policyClient) ListPermissions(ctx context.Context, pr policies.PolicyReq, permissionsFilter []string) (policies.Permissions, error) {
	if len(permissionsFilter) == 0 {
		switch pr.ObjectType {
		case policies.ThingType:
			permissionsFilter = defThingsFilterPermissions
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
	pers, err := pc.retrievePermissions(ctx, pr, permissionsFilter)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return pers, nil
}

func (pc policyClient) policyValidation(pr policies.PolicyReq) error {
	if pr.ObjectType == policies.PlatformType && pr.Object != policies.MagistralaObject {
		return errPlatform
	}

	return nil
}

func (pc policyClient) addPolicyPreCondition(ctx context.Context, pr policies.PolicyReq) ([]*v1.Precondition, error) {
	// Checks are required for following  ( -> means adding)
	// 1.) user -> group (both user groups and channels)
	// 2.) user -> thing
	// 3.) group -> group (both for adding parent_group and channels)
	// 4.) group (channel) -> thing
	// 5.) user -> domain

	switch {
	// 1.) user -> group (both user groups and channels)
	// Checks :
	// - USER with ANY RELATION to DOMAIN
	// - GROUP with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == policies.UserType && pr.ObjectType == policies.GroupType:
		return pc.userGroupPreConditions(ctx, pr)

	// 2.) user -> thing
	// Checks :
	// - USER with ANY RELATION to DOMAIN
	// - THING with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == policies.UserType && pr.ObjectType == policies.ThingType:
		return pc.userThingPreConditions(ctx, pr)

	// 3.) group -> group (both for adding parent_group and channels)
	// Checks :
	// - CHILD_GROUP with out PARENT_GROUP RELATION with any GROUP
	case pr.SubjectType == policies.GroupType && pr.ObjectType == policies.GroupType:
		return groupPreConditions(pr)

	// 4.) group (channel) -> thing
	// Checks :
	// - GROUP (channel) with DOMAIN RELATION to DOMAIN
	// - NO GROUP should not have PARENT_GROUP RELATION with GROUP (channel)
	// - THING with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == policies.GroupType && pr.ObjectType == policies.ThingType:
		return channelThingPreCondition(pr)

	// 5.) user -> domain
	// Checks :
	// - User doesn't have any relation with domain
	case pr.SubjectType == policies.UserType && pr.ObjectType == policies.DomainType:
		return pc.userDomainPreConditions(ctx, pr)

	// Check thing and group not belongs to other domain before adding to domain
	case pr.SubjectType == policies.DomainType && pr.Relation == policies.DomainRelation && (pr.ObjectType == policies.ThingType || pr.ObjectType == policies.GroupType):
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

func (pc policyClient) userGroupPreConditions(ctx context.Context, pr policies.PolicyReq) ([]*v1.Precondition, error) {
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
	if err := pc.checkPolicy(ctx, policies.PolicyReq{
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

func (pc policyClient) userThingPreConditions(ctx context.Context, pr policies.PolicyReq) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	// user should not have any relation with thing
	preconds = append(preconds, &v1.Precondition{
		Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
		Filter: &v1.RelationshipFilter{
			ResourceType:       policies.ThingType,
			OptionalResourceId: pr.Object,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       policies.UserType,
				OptionalSubjectId: pr.Subject,
			},
		},
	})

	isSuperAdmin := false
	if err := pc.checkPolicy(ctx, policies.PolicyReq{
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
	// For New thing
	// - THING without DOMAIN RELATION to ANY DOMAIN
	case pr.ObjectKind == policies.NewThingKind:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.ThingType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   policies.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: policies.DomainType,
					},
				},
			},
		)
	default:
		// For existing thing
		// - THING without DOMAIN RELATION to ANY DOMAIN
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       policies.ThingType,
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

func (pc policyClient) userDomainPreConditions(ctx context.Context, pr policies.PolicyReq) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	if err := pc.checkPolicy(ctx, policies.PolicyReq{
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

func (pc policyClient) checkPolicy(ctx context.Context, pr policies.PolicyReq) error {
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

	resp, err := pc.permissionClient.CheckPermission(ctx, &checkReq)
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

func (pc policyClient) retrieveObjects(ctx context.Context, pr policies.PolicyReq, nextPageToken string, limit uint64) ([]policies.PolicyRes, string, error) {
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
	stream, err := pc.permissionClient.LookupResources(ctx, resourceReq)
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
			return []policies.PolicyRes{}, token, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		}
	}
}

func (pc policyClient) retrieveAllObjects(ctx context.Context, pr policies.PolicyReq) ([]policies.PolicyRes, error) {
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
	stream, err := pc.permissionClient.LookupResources(ctx, resourceReq)
	if err != nil {
		return nil, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}
	tuples := []policies.PolicyRes{}
	for {
		resp, err := stream.Recv()
		switch {
		case errors.Contains(err, io.EOF):
			return tuples, nil
		case err != nil:
			return tuples, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		default:
			tuples = append(tuples, policies.PolicyRes{Object: resp.ResourceObjectId})
		}
	}
}

func (pc policyClient) retrieveSubjects(ctx context.Context, pr policies.PolicyReq, nextPageToken string, limit uint64) ([]policies.PolicyRes, string, error) {
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
	stream, err := pc.permissionClient.LookupSubjects(ctx, &subjectsReq)
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
			return []policies.PolicyRes{}, token, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		}
	}
}

func (pc policyClient) retrieveAllSubjects(ctx context.Context, pr policies.PolicyReq) ([]policies.PolicyRes, error) {
	var tuples []policies.PolicyRes
	nextPageToken := ""
	for i := 0; ; i++ {
		relationTuples, npt, err := pc.retrieveSubjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
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

func (pc policyClient) retrievePermissions(ctx context.Context, pr policies.PolicyReq, filterPermission []string) (policies.Permissions, error) {
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
	resp, err := pc.client.PermissionsServiceClient.CheckBulkPermissions(ctx, &v1.CheckBulkPermissionsRequest{
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

func groupPreConditions(pr policies.PolicyReq) ([]*v1.Precondition, error) {
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

func channelThingPreCondition(pr policies.PolicyReq) ([]*v1.Precondition, error) {
	if pr.SubjectKind != policies.ChannelsKind {
		return nil, errors.Wrap(errors.ErrMalformedEntity, errInvalidSubject)
	}
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
		{
			Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:     policies.GroupType,
				OptionalRelation: policies.ParentGroupRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       policies.GroupType,
					OptionalSubjectId: pr.Subject,
				},
			},
		},
		{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       policies.ThingType,
				OptionalResourceId: pr.Object,
				OptionalRelation:   policies.DomainRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       policies.DomainType,
					OptionalSubjectId: pr.Domain,
				},
			},
		},
	}
	return precond, nil
}

func objectsToAuthPolicies(objects []*v1.LookupResourcesResponse) []policies.PolicyRes {
	var policyList []policies.PolicyRes
	for _, obj := range objects {
		policyList = append(policyList, policies.PolicyRes{
			Object: obj.GetResourceObjectId(),
		})
	}
	return policyList
}

func subjectsToAuthPolicies(subjects []*v1.LookupSubjectsResponse) []policies.PolicyRes {
	var policyList []policies.PolicyRes
	for _, sub := range subjects {
		policyList = append(policyList, policies.PolicyRes{
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
