// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authsvc

import (
	"context"

	"github.com/absmach/magistrala/auth/api/grpc/auth"
	grpcAuthV1 "github.com/absmach/magistrala/internal/grpc/auth/v1"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/grpcclient"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

type authentication struct {
	authSvcClient grpcAuthV1.AuthServiceClient
}

var _ authn.Authentication = (*authentication)(nil)

func NewAuthentication(ctx context.Context, cfg grpcclient.Config) (authn.Authentication, grpcclient.Handler, error) {
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
	return authentication{authSvcClient}, client, nil
}

func (a authentication) Authenticate(ctx context.Context, token string) (authn.Session, error) {
	res, err := a.authSvcClient.Authenticate(ctx, &grpcAuthV1.AuthNReq{Token: token})
	if err != nil {
		return authn.Session{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	return authn.Session{DomainUserID: res.GetId(), UserID: res.GetUserId(), DomainID: res.GetDomainId()}, nil
}
