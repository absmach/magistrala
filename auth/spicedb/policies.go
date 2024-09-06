// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errInternal = errors.New("spicedb internal error")

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
	return svcerr.ErrAuthorization
}

func handleSpicedbError(err error) error {
	if st, ok := status.FromError(err); ok {
		return convertGRPCStatusToError(st)
	}
	return err
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
