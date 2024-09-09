// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient

import (
	"context"

	"github.com/absmach/magistrala"
	authgrpc "github.com/absmach/magistrala/auth/api/grpc"
	authclient "github.com/absmach/magistrala/internal/auth"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	thingsauth "github.com/absmach/magistrala/things/api/grpc"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

var errSvcNotServing = errors.New("service is not serving")

// SetupAuthClient loads Auth gRPC configuration and creates new Auth gRPC client.
//
// For example:
//
// authClient, authHandler, err := auth.SetupAuth(ctx, auth.Config{}).
func SetupAuthClient(ctx context.Context, cfg Config) (auth.AuthClient, Handler, error) {
	client, err := newHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, errSvcNotServing
	}

	return authclient.NewAuthClient(client.Connection(), cfg.Timeout), client, nil
}

// SetupPolicyClient loads Policy gRPC configuration and creates a new Policy gRPC client.
//
// For example:
//
// policyClient, policyHandler, err := auth.SetupPolicyClient(ctx, auth.Config{}).
func SetupPolicyClient(ctx context.Context, cfg Config) (magistrala.PolicyServiceClient, Handler, error) {
	client, err := newHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, errSvcNotServing
	}

	return authgrpc.NewPolicyClient(client.Connection(), cfg.Timeout), client, nil
}

// SetupThingsClient loads things gRPC configuration and creates new things gRPC client.
//
// For example:
//
// thingClient, thingHandler, err := auth.SetupThings(ctx, auth.Config{}).
func SetupThingsClient(ctx context.Context, cfg Config) (magistrala.AuthzServiceClient, Handler, error) {
	client, err := newHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "things",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, errSvcNotServing
	}

	return thingsauth.NewClient(client.Connection(), cfg.Timeout), client, nil
}
