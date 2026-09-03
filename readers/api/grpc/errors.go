// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// encodeError maps a service error onto the gRPC status the reader returns.
func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, svcerr.ErrInvalidPolicy),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingMemberType,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct,
		err == apiutil.ErrMissingUserID,
		err == apiutil.ErrMissingPATID:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, svcerr.ErrAuthentication),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, svcerr.ErrWorkspaceAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, svcerr.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Contains(err, svcerr.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
