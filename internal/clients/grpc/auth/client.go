// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	grpcclient "github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	authapi "github.com/mainflux/mainflux/users/policies/api/grpc"
)

const envAuthGrpcPrefix = "MF_AUTH_GRPC_"

var errGrpcConfig = errors.New("failed to load grpc configuration")

// Setup loads Auth gRPC configuration from environment variable and creates new Auth gRPC API.
func Setup(svcName string) (policies.AuthServiceClient, grpcclient.ClientHandler, error) {
	config := grpcclient.Config{}
	if err := env.Parse(&config, env.Options{Prefix: envAuthGrpcPrefix}); err != nil {
		return nil, nil, errors.Wrap(errGrpcConfig, err)
	}
	c, ch, err := grpcclient.Setup(config, svcName)
	if err != nil {
		return nil, nil, err
	}

	return authapi.NewClient(c.ClientConn, config.Timeout), ch, nil
}
