// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
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
)

type policyAgent struct {
	client           *authzed.ClientWithExperimental
	permissionClient v1.PermissionsServiceClient
	logger           *slog.Logger
}

func NewPolicyAgent(client *authzed.ClientWithExperimental, logger *slog.Logger) auth.PolicyAgent {
	return &policyAgent{
		client:           client,
		permissionClient: client.PermissionsServiceClient,
		logger:           logger,
	}
}

func (pa *policyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
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

	resp, err := pa.permissionClient.CheckPermission(ctx, &checkReq)
	if err != nil {
		return handleSpicedbError(err)
	}
	if resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return nil
	}
	if reason, ok := v1.CheckPermissionResponse_Permissionship_name[int32(resp.Permissionship)]; ok {
		return errors.Wrap(svcerr.ErrAuthorization, errors.New(reason))
	}
	return errors.ErrAuthorization
}

func (pa *policyAgent) AddPolicies(ctx context.Context, prs []auth.PolicyReq) error {
	updates := []*v1.RelationshipUpdate{}
	var preconds []*v1.Precondition
	for _, pr := range prs {
		precond, err := pa.addPolicyPreCondition(ctx, pr)
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
	_, err := pa.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates, OptionalPreconditions: preconds})
	if err != nil {
		return errors.Wrap(errAddPolicies, handleSpicedbError(err))
	}
	return nil
}

func (pa *policyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	precond, err := pa.addPolicyPreCondition(ctx, pr)
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
	_, err = pa.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates, OptionalPreconditions: precond})
	if err != nil {
		return errors.Wrap(errAddPolicies, handleSpicedbError(err))
	}
	return nil
}

func (pa *policyAgent) DeletePolicies(ctx context.Context, prs []auth.PolicyReq) error {
	updates := []*v1.RelationshipUpdate{}
	for _, pr := range prs {
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
	_, err := pa.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
	if err != nil {
		return errors.Wrap(errRemovePolicies, handleSpicedbError(err))
	}
	return nil
}

func (pa *policyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	req := &v1.DeleteRelationshipsRequest{
		RelationshipFilter: &v1.RelationshipFilter{
			ResourceType:       pr.ObjectType,
			OptionalResourceId: pr.Object,
			OptionalRelation:   pr.Relation,
			OptionalSubjectFilter: &v1.SubjectFilter{
				OptionalSubjectId: pr.Subject,
				SubjectType:       pr.SubjectType,
				OptionalRelation: &v1.SubjectFilter_RelationFilter{
					Relation: pr.SubjectRelation,
				},
			},
		},
	}
	if _, err := pa.permissionClient.DeleteRelationships(ctx, req); err != nil {
		return errors.Wrap(errRemovePolicies, handleSpicedbError(err))
	}
	return nil
}

// RetrieveObjects - Listing of things.
func (pa *policyAgent) RetrieveObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
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
	stream, err := pa.permissionClient.LookupResources(ctx, resourceReq)
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
			return []auth.PolicyRes{}, token, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		}
	}
}

func (pa *policyAgent) RetrieveAllObjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
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
	stream, err := pa.permissionClient.LookupResources(ctx, resourceReq)
	if err != nil {
		return nil, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}
	tuples := []auth.PolicyRes{}
	for {
		resp, err := stream.Recv()
		switch {
		case errors.Contains(err, io.EOF):
			return tuples, nil
		case err != nil:
			return tuples, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		default:
			tuples = append(tuples, auth.PolicyRes{Object: resp.ResourceObjectId})
		}
	}
}

func (pa *policyAgent) RetrieveAllObjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	var count int
	nextPageToken := ""
	for {
		relationTuples, npt, err := pa.RetrieveObjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
		if err != nil {
			return count, err
		}
		count = count + len(relationTuples)
		if npt == "" {
			break
		}
		nextPageToken = npt
	}
	return count, nil
}

func (pa *policyAgent) RetrieveSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
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
	stream, err := pa.permissionClient.LookupSubjects(ctx, &subjectsReq)
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
			return []auth.PolicyRes{}, token, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
		}
	}
}

func (pa *policyAgent) RetrieveAllSubjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
	var tuples []auth.PolicyRes
	nextPageToken := ""
	for i := 0; ; i++ {
		relationTuples, npt, err := pa.RetrieveSubjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
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

func (pa *policyAgent) RetrieveAllSubjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	var count int
	nextPageToken := ""
	for {
		relationTuples, npt, err := pa.RetrieveSubjects(ctx, pr, nextPageToken, defRetrieveAllLimit)
		if err != nil {
			return count, err
		}
		count = count + len(relationTuples)
		if npt == "" {
			break
		}
		nextPageToken = npt
	}
	return count, nil
}

func (pa *policyAgent) RetrievePermissions(ctx context.Context, pr auth.PolicyReq, filterPermission []string) (auth.Permissions, error) {
	var permissionChecks []*v1.BulkCheckPermissionRequestItem
	for _, fp := range filterPermission {
		permissionChecks = append(permissionChecks, &v1.BulkCheckPermissionRequestItem{
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
	resp, err := pa.client.ExperimentalServiceClient.BulkCheckPermission(ctx, &v1.BulkCheckPermissionRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		Items: permissionChecks,
	})
	if err != nil {
		return auth.Permissions{}, errors.Wrap(errRetrievePolicies, handleSpicedbError(err))
	}

	permissions := []string{}
	for _, pair := range resp.Pairs {
		if pair.GetError() != nil {
			s := pair.GetError()
			return auth.Permissions{}, errors.Wrap(errRetrievePolicies, convertGRPCStatusToError(convertToGrpcStatus(s)))
		}
		item := pair.GetItem()
		req := pair.GetRequest()
		if item != nil && req != nil && item.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
			permissions = append(permissions, req.GetPermission())
		}
	}
	return permissions, nil
}

func objectsToAuthPolicies(objects []*v1.LookupResourcesResponse) []auth.PolicyRes {
	var policies []auth.PolicyRes
	for _, obj := range objects {
		policies = append(policies, auth.PolicyRes{
			Object: obj.GetResourceObjectId(),
		})
	}
	return policies
}

func subjectsToAuthPolicies(subjects []*v1.LookupSubjectsResponse) []auth.PolicyRes {
	var policies []auth.PolicyRes
	for _, sub := range subjects {
		policies = append(policies, auth.PolicyRes{
			Subject: sub.Subject.GetSubjectObjectId(),
		})
	}
	return policies
}

func (pa *policyAgent) Watch(ctx context.Context, continueToken string) {
	stream, err := pa.client.WatchServiceClient.Watch(ctx, &v1.WatchRequest{
		OptionalObjectTypes: []string{},
		OptionalStartCursor: &v1.ZedToken{Token: continueToken},
	})
	if err != nil {
		pa.logger.Error(fmt.Sprintf("got error while watching: %s", err.Error()))
	}
	for {
		watchResp, err := stream.Recv()
		switch err {
		case nil:
			pa.publishToStream(watchResp)
		case io.EOF:
			pa.logger.Info("got EOF while watch streaming")
			return
		default:
			pa.logger.Error(fmt.Sprintf("got error while watch streaming : %s", err.Error()))
			return
		}
	}
}

func (pa *policyAgent) publishToStream(resp *v1.WatchResponse) {
	pa.logger.Info(fmt.Sprintf("Publish next token %s", resp.ChangesThrough.Token))

	for _, update := range resp.Updates {
		operation := v1.RelationshipUpdate_Operation_name[int32(update.Operation)]
		objectType := update.Relationship.Resource.ObjectType
		objectID := update.Relationship.Resource.ObjectId
		relation := update.Relationship.Relation
		subjectType := update.Relationship.Subject.Object.ObjectType
		subjectRelation := update.Relationship.Subject.OptionalRelation
		subjectID := update.Relationship.Subject.Object.ObjectId

		pa.logger.Info(fmt.Sprintf(`
		Operation : %s	object_type: %s		object_id: %s 	relation: %s 	subject_type: %s 	subject_relation: %s	subject_id: %s
		`, operation, objectType, objectID, relation, subjectType, subjectRelation, subjectID))
	}
}

func (pa *policyAgent) addPolicyPreCondition(ctx context.Context, pr auth.PolicyReq) ([]*v1.Precondition, error) {
	// Checks are required for following  ( -> means adding)
	// 1.) user -> group (both user groups and channels)
	// 2.) user -> thing
	// 3.) group -> group (both for adding parent_group and channels)
	// 4.) group (channel) -> thing

	switch {
	// 1.) user -> group (both user groups and channels)
	// Checks :
	// - USER with ANY RELATION to DOMAIN
	// - GROUP with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == auth.UserType && pr.ObjectType == auth.GroupType:
		return pa.userGroupPreConditions(ctx, pr)

	// 2.) user -> thing
	// Checks :
	// - USER with ANY RELATION to DOMAIN
	// - THING with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == auth.UserType && pr.ObjectType == auth.ThingType:
		return pa.userThingPreConditions(ctx, pr)

	// 3.) group -> group (both for adding parent_group and channels)
	// Checks :
	// - CHILD_GROUP with out PARENT_GROUP RELATION with any GROUP
	case pr.SubjectType == auth.GroupType && pr.ObjectType == auth.GroupType:
		return groupPreConditions(pr)

	// 4.) group (channel) -> thing
	// Checks :
	// - GROUP (channel) with DOMAIN RELATION to DOMAIN
	// - NO GROUP should not have PARENT_GROUP RELATION with GROUP (channel)
	// - THING with DOMAIN RELATION to DOMAIN
	case pr.SubjectType == auth.GroupType && pr.ObjectType == auth.ThingType:
		return channelThingPreCondition(pr)

	// Check thing and group not belongs to other domain before adding to domain
	case pr.SubjectType == auth.DomainType && pr.Relation == auth.DomainRelation && (pr.ObjectType == auth.ThingType || pr.ObjectType == auth.GroupType):
		preconds := []*v1.Precondition{
			{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       pr.ObjectType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: auth.DomainType,
					},
				},
			},
		}
		return preconds, nil
	}
	return nil, nil
}

func (pa *policyAgent) userGroupPreConditions(ctx context.Context, pr auth.PolicyReq) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	// user should not have any relation with group
	preconds = append(preconds, &v1.Precondition{
		Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
		Filter: &v1.RelationshipFilter{
			ResourceType:       auth.GroupType,
			OptionalResourceId: pr.Object,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       auth.UserType,
				OptionalSubjectId: pr.Subject,
			},
		},
	})
	isSuperAdmin := false
	if err := pa.CheckPolicy(ctx, auth.PolicyReq{
		Subject:     pr.Subject,
		SubjectType: pr.SubjectType,
		Permission:  auth.AdminPermission,
		Object:      auth.MagistralaObject,
		ObjectType:  auth.PlatformType,
	}); err == nil {
		isSuperAdmin = true
	}

	if !isSuperAdmin {
		preconds = append(preconds, &v1.Precondition{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       auth.DomainType,
				OptionalResourceId: pr.Domain,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       auth.UserType,
					OptionalSubjectId: pr.Subject,
				},
			},
		})
	}
	switch {
	case pr.ObjectKind == auth.NewGroupKind || pr.ObjectKind == auth.NewChannelKind:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       auth.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: auth.DomainType,
					},
				},
			},
		)
	default:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       auth.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType:       auth.DomainType,
						OptionalSubjectId: pr.Domain,
					},
				},
			},
		)
	}
	return preconds, nil
}

func (pa *policyAgent) userThingPreConditions(ctx context.Context, pr auth.PolicyReq) ([]*v1.Precondition, error) {
	var preconds []*v1.Precondition

	// user should not have any relation with thing
	preconds = append(preconds, &v1.Precondition{
		Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
		Filter: &v1.RelationshipFilter{
			ResourceType:       auth.ThingType,
			OptionalResourceId: pr.Object,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       auth.UserType,
				OptionalSubjectId: pr.Subject,
			},
		},
	})

	isSuperAdmin := false
	if err := pa.CheckPolicy(ctx, auth.PolicyReq{
		Subject:     pr.Subject,
		SubjectType: pr.SubjectType,
		Permission:  auth.AdminPermission,
		Object:      auth.MagistralaObject,
		ObjectType:  auth.PlatformType,
	}); err == nil {
		isSuperAdmin = true
	}

	if !isSuperAdmin {
		preconds = append(preconds, &v1.Precondition{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       auth.DomainType,
				OptionalResourceId: pr.Domain,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       auth.UserType,
					OptionalSubjectId: pr.Subject,
				},
			},
		})
	}
	switch {
	// For New thing
	// - THING without DOMAIN RELATION to ANY DOMAIN
	case pr.ObjectKind == auth.NewThingKind:
		preconds = append(preconds,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       auth.ThingType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: auth.DomainType,
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
					ResourceType:       auth.ThingType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType:       auth.DomainType,
						OptionalSubjectId: pr.Domain,
					},
				},
			},
		)
	}
	return preconds, nil
}

func groupPreConditions(pr auth.PolicyReq) ([]*v1.Precondition, error) {
	// - PARENT_GROUP (subject) with DOMAIN RELATION to DOMAIN
	precond := []*v1.Precondition{
		{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       auth.GroupType,
				OptionalResourceId: pr.Subject,
				OptionalRelation:   auth.DomainRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       auth.DomainType,
					OptionalSubjectId: pr.Domain,
				},
			},
		},
	}
	if pr.ObjectKind != auth.ChannelsKind {
		precond = append(precond,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       auth.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.ParentGroupRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: auth.GroupType,
					},
				},
			},
		)
	}
	switch {
	// - NEW CHILD_GROUP (object) with out DOMAIN RELATION to ANY DOMAIN
	case pr.ObjectType == auth.GroupType && pr.ObjectKind == auth.NewGroupKind:
		precond = append(precond,
			&v1.Precondition{
				Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &v1.RelationshipFilter{
					ResourceType:       auth.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType: auth.DomainType,
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
					ResourceType:       auth.GroupType,
					OptionalResourceId: pr.Object,
					OptionalRelation:   auth.DomainRelation,
					OptionalSubjectFilter: &v1.SubjectFilter{
						SubjectType:       auth.DomainType,
						OptionalSubjectId: pr.Domain,
					},
				},
			},
		)
	}
	return precond, nil
}

func channelThingPreCondition(pr auth.PolicyReq) ([]*v1.Precondition, error) {
	if pr.SubjectKind != auth.ChannelsKind {
		return nil, errors.Wrap(errors.ErrMalformedEntity, errInvalidSubject)
	}
	precond := []*v1.Precondition{
		{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       auth.GroupType,
				OptionalResourceId: pr.Subject,
				OptionalRelation:   auth.DomainRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       auth.DomainType,
					OptionalSubjectId: pr.Domain,
				},
			},
		},
		{
			Operation: v1.Precondition_OPERATION_MUST_NOT_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:     auth.GroupType,
				OptionalRelation: auth.ParentGroupRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       auth.GroupType,
					OptionalSubjectId: pr.Subject,
				},
			},
		},
		{
			Operation: v1.Precondition_OPERATION_MUST_MATCH,
			Filter: &v1.RelationshipFilter{
				ResourceType:       auth.ThingType,
				OptionalResourceId: pr.Object,
				OptionalRelation:   auth.DomainRelation,
				OptionalSubjectFilter: &v1.SubjectFilter{
					SubjectType:       auth.DomainType,
					OptionalSubjectId: pr.Domain,
				},
			},
		},
	}
	return precond, nil
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
		return errors.Wrap(errors.ErrNotFound, errors.New(st.Message()))
	case codes.InvalidArgument:
		return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
	case codes.AlreadyExists:
		return errors.Wrap(errors.ErrConflict, errors.New(st.Message()))
	case codes.Unauthenticated:
		return errors.Wrap(errors.ErrAuthentication, errors.New(st.Message()))
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
		return errors.Wrap(errors.ErrAuthorization, errors.New(st.Message()))
	default:
		return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
	}
}
