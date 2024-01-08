// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/absmach/magistrala"
	authgrpc "github.com/absmach/magistrala/auth/api/grpc"
	thingsauth "github.com/absmach/magistrala/things/api/grpc"
)

// Setup loads Auth gRPC configuration and creates new Auth gRPC client.
//
// For example:
//
//	authClient, authHandler, err := auth.Setup(auth.Config{})
func Setup(cfg Config) (magistrala.AuthServiceClient, Handler, error) {
	client, err := newHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return authgrpc.NewClient(client.Connection(), cfg.Timeout), client, nil
}

// Setup loads Authz gRPC configuration and creates new Authz gRPC client.
//
// For example:
//
//	authzClient, authzHandler, err := auth.Setup(auth.Config{})
func SetupAuthz(cfg Config) (magistrala.AuthzServiceClient, Handler, error) {
	client, err := newHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	return thingsauth.NewClient(client.Connection(), cfg.Timeout), client, nil
}
