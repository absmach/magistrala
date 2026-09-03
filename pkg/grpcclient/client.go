// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient

import (
	"context"

	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcDevicesV1 "github.com/absmach/magistrala/api/grpc/devices/v1"
	grpcGroupsV1 "github.com/absmach/magistrala/api/grpc/groups/v1"
	grpcUsersV1 "github.com/absmach/magistrala/api/grpc/users/v1"
	grpcWorkspacesV1 "github.com/absmach/magistrala/api/grpc/workspaces/v1"
)

// SetupDomiansClient loads workspaces gRPC configuration and creates a new workspaces gRPC client.
//
// For example:
//
// workspacesClient, workspacesHandler, err := grpcclient.SetupWorkspacesClient(ctx, grpcclient.Config{}).
func SetupWorkspacesClient(ctx context.Context, cfg Config) (grpcWorkspacesV1.WorkspacesServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return grpcWorkspacesV1.NewWorkspacesServiceClient(client.Connection()), client, nil
}

// SetupDevicesClient loads devices gRPC configuration and creates new devices gRPC client.
//
// For example:
//
// clientClient, clientHandler, err := grpcclient.SetupDevices(ctx, grpcclient.Config{}).
func SetupDevicesClient(ctx context.Context, cfg Config) (grpcDevicesV1.DevicesServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return grpcDevicesV1.NewDevicesServiceClient(client.Connection()), client, nil
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

	return grpcChannelsV1.NewChannelsServiceClient(client.Connection()), client, nil
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

	return grpcGroupsV1.NewGroupsServiceClient(client.Connection()), client, nil
}

// SetupUsersClient loads users gRPC configuration and creates new users gRPC client.
//
// For example:
//
// usersClient, usersHandler, err := grpcclient.SetupUsersClient(ctx, grpcclient.Config{}).
func SetupUsersClient(ctx context.Context, cfg Config) (grpcUsersV1.UsersServiceClient, Handler, error) {
	client, err := NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return grpcUsersV1.NewUsersServiceClient(client.Connection()), client, nil
}
