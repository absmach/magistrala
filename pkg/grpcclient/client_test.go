// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	tokengrpcapi "github.com/absmach/magistrala/auth/api/grpc/token"
	"github.com/absmach/magistrala/auth/mocks"
	clientsgrpcapi "github.com/absmach/magistrala/clients/api/grpc"
	climocks "github.com/absmach/magistrala/clients/private/mocks"
	domainsgrpcapi "github.com/absmach/magistrala/domains/api/grpc"
	domainsMocks "github.com/absmach/magistrala/domains/mocks"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/internal/grpc/domains/v1"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestSetupToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registerAuthServiceServer := func(srv *grpc.Server) {
		grpcTokenV1.RegisterTokenServiceServer(srv, tokengrpcapi.NewTokenServer(new(mocks.Service)))
	}
	gs := grpcserver.NewServer(ctx, cancel, "auth", server.Config{Port: "12345"}, registerAuthServiceServer, mglog.NewMock())
	go func() {
		err := gs.Start()
		assert.Nil(t, err, fmt.Sprintf(`"Unexpected error creating server %s"`, err))
	}()
	defer func() {
		err := gs.Stop()
		assert.Nil(t, err, fmt.Sprintf(`"Unexpected error stopping server %s"`, err))
	}()

	cases := []struct {
		desc   string
		config grpcclient.Config
		err    error
	}{
		{
			desc: "successful",
			config: grpcclient.Config{
				URL:     "localhost:12345",
				Timeout: time.Second,
			},
			err: nil,
		},
		{
			desc: "failed with empty URL",
			config: grpcclient.Config{
				URL:     "",
				Timeout: time.Second,
			},
			err: errors.New("service is not serving"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client, handler, err := grpcclient.SetupTokenClient(context.Background(), c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestSetupClientsClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	registerClientsServiceServer := func(srv *grpc.Server) {
		grpcClientsV1.RegisterClientsServiceServer(srv, clientsgrpcapi.NewServer(new(climocks.Service)))
	}
	gs := grpcserver.NewServer(ctx, cancel, "clients", server.Config{Port: "12345"}, registerClientsServiceServer, mglog.NewMock())
	go func() {
		err := gs.Start()
		assert.Nil(t, err, fmt.Sprintf(`"Unexpected error creating server %s"`, err))
	}()
	defer func() {
		err := gs.Stop()
		assert.Nil(t, err, fmt.Sprintf(`"Unexpected error stopping server %s"`, err))
	}()

	cases := []struct {
		desc   string
		config grpcclient.Config
		err    error
	}{
		{
			desc: "successful",
			config: grpcclient.Config{
				URL:     "localhost:12345",
				Timeout: time.Second,
			},
			err: nil,
		},
		{
			desc: "failed with empty URL",
			config: grpcclient.Config{
				URL:     "",
				Timeout: time.Second,
			},
			err: errors.New("service is not serving"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client, handler, err := grpcclient.SetupClientsClient(context.Background(), c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestSetupDomainsClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registerDomainsServiceServer := func(srv *grpc.Server) {
		grpcDomainsV1.RegisterDomainsServiceServer(srv, domainsgrpcapi.NewDomainsServer(new(domainsMocks.Service)))
	}
	gs := grpcserver.NewServer(ctx, cancel, "auth", server.Config{Port: "12345"}, registerDomainsServiceServer, mglog.NewMock())
	go func() {
		err := gs.Start()
		assert.Nil(t, err, fmt.Sprintf("Unexpected error creating server %s", err))
	}()
	defer func() {
		err := gs.Stop()
		assert.Nil(t, err, fmt.Sprintf("Unexpected error stopping server %s", err))
	}()

	cases := []struct {
		desc   string
		config grpcclient.Config
		err    error
	}{
		{
			desc: "successfully",
			config: grpcclient.Config{
				URL:     "localhost:12345",
				Timeout: time.Second,
			},
			err: nil,
		},
		{
			desc: "failed with empty URL",
			config: grpcclient.Config{
				URL:     "",
				Timeout: time.Second,
			},
			err: errors.New("service is not serving"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client, handler, err := grpcclient.SetupDomainsClient(context.Background(), c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}
