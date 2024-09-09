// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	authzSvcName  = "magistrala.AuthzService"
	authnSvcName  = "magistrala.AuthnService"
	policySvcName = "magistrala.PolicyService"
)

var _ magistrala.PolicyServiceClient = (*policyGrpcClient)(nil)

type policyGrpcClient struct {
	deleteUserPolicies endpoint.Endpoint
	timeout            time.Duration
}

// NewPolicyClient returns new policy gRPC client instance.
func NewPolicyClient(conn *grpc.ClientConn, timeout time.Duration) magistrala.PolicyServiceClient {
	return &policyGrpcClient{
		deleteUserPolicies: kitgrpc.NewClient(
			conn,
			policySvcName,
			"DeleteUserPolicies",
			encodeDeleteUserPoliciesRequest,
			decodeDeleteUserPoliciesResponse,
			magistrala.DeletePolicyRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client policyGrpcClient) DeleteUserPolicies(ctx context.Context, in *magistrala.DeleteUserPoliciesReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.deleteUserPolicies(ctx, deleteUserPoliciesReq{
		ID: in.GetId(),
	})
	if err != nil {
		return &magistrala.DeletePolicyRes{}, decodeError(err)
	}

	dpr := res.(deletePolicyRes)
	return &magistrala.DeletePolicyRes{Deleted: dpr.deleted}, nil
}

func decodeDeleteUserPoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.DeletePolicyRes)
	return deletePolicyRes{deleted: res.GetDeleted()}, nil
}

func encodeDeleteUserPoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(deleteUserPoliciesReq)
	return &magistrala.DeleteUserPoliciesReq{
		Id: req.ID,
	}, nil
}

func decodeError(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			return errors.Wrap(svcerr.ErrNotFound, errors.New(st.Message()))
		case codes.InvalidArgument:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.AlreadyExists:
			return errors.Wrap(svcerr.ErrConflict, errors.New(st.Message()))
		case codes.Unauthenticated:
			return errors.Wrap(svcerr.ErrAuthentication, errors.New(st.Message()))
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
	return err
}
