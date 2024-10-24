// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient

import (
	"context"

	tokengrpc "github.com/absmach/magistrala/auth/api/grpc/token"
	channelsgrpc "github.com/absmach/magistrala/channels/api/grpc"
	clientsauth "github.com/absmach/magistrala/clients/api/grpc"
	domainsgrpc "github.com/absmach/magistrala/domains/api/grpc"
	groupsgrpc "github.com/absmach/magistrala/groups/api/grpc"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/internal/grpc/domains/v1"
	grpcGroupsV1 "github.com/absmach/magistrala/internal/grpc/groups/v1"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

// SetupTokenClient loads auth services token gRPC configuration and creates new Token services gRPC client.
//
// For example:
//
// tokenClient, tokenHandler, err := grpcclient.SetupTokenClient(ctx, grpcclient.Config{}).
func SetupTokenClient(ctx context.Context, cfg Config) (grpcTokenV1.TokenServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		// Health Service name is the svcName provided during gRPC server creation `grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerAuthServiceServer, logger)`
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, ErrSvcNotServing
	}

	return tokengrpc.NewTokenClient(client.Connection(), cfg.Timeout), client, nil
}

// SetupDomiansClient loads domains gRPC configuration and creates a new domains gRPC client.
//
// For example:
//
// domainsClient, domainsHandler, err := grpcclient.SetupDomainsClient(ctx, grpcclient.Config{}).
func SetupDomainsClient(ctx context.Context, cfg Config) (grpcDomainsV1.DomainsServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return domainsgrpc.NewDomainsClient(client.Connection(), cfg.Timeout), client, nil
}

// SetupClientsClient loads clients gRPC configuration and creates new clients gRPC client.
//
// For example:
//
// clientClient, clientHandler, err := grpcclient.SetupClients(ctx, grpcclient.Config{}).
func SetupClientsClient(ctx context.Context, cfg Config) (grpcClientsV1.ClientsServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return clientsauth.NewClient(client.Connection(), cfg.Timeout), client, nil
}

// SetupChannelsClient loads channels gRPC configuration and creates new channels gRPC client.
//
// For example:
//
// channelClient, channelHandler, err := grpcclient.SetupChannelsClient(ctx, grpcclient.Config{}).
func SetupChannelsClient(ctx context.Context, cfg Config) (grpcChannelsV1.ChannelsServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return channelsgrpc.NewClient(client.Connection(), cfg.Timeout), client, nil
}

// SetupGroupsClient loads groups gRPC configuration and creates new groups gRPC client.
//
// For example:
//
// groupClient, groupHandler, err := grpcclient.SetupGroupsClient(ctx, grpcclient.Config{}).
func SetupGroupsClient(ctx context.Context, cfg Config) (grpcGroupsV1.GroupsServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return groupsgrpc.NewClient(client.Connection(), cfg.Timeout), client, nil
}
