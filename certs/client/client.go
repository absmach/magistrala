// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	grpcCertsV1 "github.com/absmach/supermq/api/grpc/certs/v1"
	grpc "github.com/absmach/supermq/pkg/grpcclient"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

func NewCertsClient(ctx context.Context, cfg grpc.Config) (grpc.Handler, grpcCertsV1.CertsServiceClient, error) {
	client, err := grpc.NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "certs",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, grpc.ErrSvcNotServing
	}
	return client, grpcCertsV1.NewCertsServiceClient(client.Connection()), nil
}
