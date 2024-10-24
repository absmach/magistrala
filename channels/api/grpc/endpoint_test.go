// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"fmt"
	"net"

	grpcapi "github.com/absmach/magistrala/clients/api/grpc"
	"github.com/absmach/magistrala/clients/private/mocks"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	"google.golang.org/grpc"
)

const port = 7000

var (
	clientID  = "testID"
	clientKey = "testKey"
	channelID = "testID"
	invalid   = "invalid"
)

func startGRPCServer(svc *mocks.Service, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	grpcClientsV1.RegisterClientsServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
}
