// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authsvc

import (
	"context"

	"github.com/absmach/magistrala/auth/api/grpc/auth"
	grpcAuthV1 "github.com/absmach/magistrala/internal/grpc/auth/v1"
	"github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/grpcclient"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

type authorization struct {
	authSvcClient grpcAuthV1.AuthServiceClient
}

var _ authz.Authorization = (*authorization)(nil)

func NewAuthorization(ctx context.Context, cfg grpcclient.Config) (authz.Authorization, grpcclient.Handler, error) {
	client, err := grpcclient.NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, grpcclient.ErrSvcNotServing
	}
	authSvcClient := auth.NewAuthClient(client.Connection(), cfg.Timeout)
	return authorization{authSvcClient}, client, nil
}

func (a authorization) Authorize(ctx context.Context, pr authz.PolicyReq) error {
	req := grpcAuthV1.AuthZReq{
		Domain:          pr.Domain,
		SubjectType:     pr.SubjectType,
		SubjectKind:     pr.SubjectKind,
		SubjectRelation: pr.SubjectRelation,
		Subject:         pr.Subject,
		Relation:        pr.Relation,
		Permission:      pr.Permission,
		Object:          pr.Object,
		ObjectType:      pr.ObjectType,
	}
	res, err := a.authSvcClient.Authorize(ctx, &req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}
