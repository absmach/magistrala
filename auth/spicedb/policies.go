// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"context"
	"fmt"
	"io"

	"github.com/absmach/magistrala/auth"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
)

const defRetrieveAllLimit = 1000

type policyAgent struct {
	client           *authzed.Client
	permissionClient v1.PermissionsServiceClient
	logger           mglog.Logger
}

func NewPolicyAgent(client *authzed.Client, logger mglog.Logger) auth.PolicyAgent {
	return &policyAgent{
		client:           client,
		permissionClient: client.PermissionsServiceClient,
		logger:           logger,
	}
}

func (pa *policyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	checkReq := v1.CheckPermissionRequest{
		Resource:   &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
		Permission: pr.Permission,
		Subject:    &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
	}

	resp, err := pa.permissionClient.CheckPermission(ctx, &checkReq)
	if err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to check permission: %w", err))
	}
	if resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return nil
	}
	if reason, ok := v1.CheckPermissionResponse_Permissionship_name[int32(resp.Permissionship)]; ok {
		return errors.Wrap(errors.ErrAuthorization, fmt.Errorf("%s", reason))
	}
	return errors.ErrAuthorization
}

func (pa *policyAgent) AddPolicies(ctx context.Context, prs []auth.PolicyReq) error {
	updates := []*v1.RelationshipUpdate{}
	for _, pr := range prs {
		updates = append(updates, &v1.RelationshipUpdate{
			Operation: v1.RelationshipUpdate_OPERATION_CREATE,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
				Relation: pr.Relation,
				Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
			},
		})
	}
	if len(updates) > 0 {
		_, err := pa.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
		if err != nil {
			return errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to add policy: %w", err))
		}
	}
	return nil
}

func (pa *policyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
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
	_, err := pa.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
	if err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to add policy: %w", err))
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
	if len(updates) > 0 {
		_, err := pa.permissionClient.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
		if err != nil {
			return errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to delete policy: %w", err))
		}
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
	_, err := pa.permissionClient.DeleteRelationships(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to remove the policy: %w", err))
	}
	return nil
}

// RetrieveObjects - Listing of things.
func (pa *policyAgent) RetrieveObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
	resourceReq := &v1.LookupResourcesRequest{
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
		return nil, "", errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to retrieve policies: %w", err))
	}
	resources := []*v1.LookupResourcesResponse{}
	var retErr error
loop:
	for {
		resp, err := stream.Recv()
		switch err {
		case nil:
			resources = append(resources, resp)
		case io.EOF:
			break loop
		default:
			retErr = err
			break loop
		}
	}
	var token string
	if len(resources) > 0 {
		token = resources[len(resources)-1].AfterResultCursor.Token
	}
	if retErr != nil {
		retErr = errors.Wrap(errors.ErrViewEntity, retErr)
	}
	return objectsToAuthPolicies(resources), token, retErr
}

func (pa *policyAgent) RetrieveAllObjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
	resourceReq := &v1.LookupResourcesRequest{
		ResourceObjectType: pr.ObjectType,
		Permission:         pr.Permission,
		Subject:            &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
	}
	stream, err := pa.permissionClient.LookupResources(ctx, resourceReq)
	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to retrieve policies: %w", err))
	}
	tuples := []auth.PolicyRes{}
	for {
		resp, err := stream.Recv()
		switch {
		case errors.Contains(err, io.EOF):
			return tuples, nil
		case err != nil:
			return tuples, err
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
		return nil, "", errors.Wrap(errors.ErrMalformedEntity, fmt.Errorf("failed to retrieve policies: %w", err))
	}
	subjects := []*v1.LookupSubjectsResponse{}
	var retErr error
loop:
	for {
		resp, err := stream.Recv()

		switch err {
		case nil:
			subjects = append(subjects, resp)
		case io.EOF:
			break loop
		default:
			retErr = err
			break loop
		}
	}
	if retErr != nil {
		retErr = errors.Wrap(errors.ErrViewEntity, retErr)
	}
	return subjectsToAuthPolicies(subjects), "", retErr
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

func (pa *policyAgent) Watch(continueToken string) {
	stream, err := pa.client.WatchServiceClient.Watch(context.Background(), &v1.WatchRequest{
		OptionalObjectTypes: []string{},
		OptionalStartCursor: &v1.ZedToken{Token: continueToken},
	})
	if err != nil {
		pa.logger.Error(fmt.Sprintf("got error while watching: %s", err.Error()))
	}
loop:
	for {
		watchResp, err := stream.Recv()
		switch err {
		case nil:
			pa.publishToStream(watchResp)
		case io.EOF:
			pa.logger.Info("got EOF while watch streaming")
			break loop
		default:
			pa.logger.Error(fmt.Sprintf("got error while watch streaming : %s", err.Error()))
			break loop
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
