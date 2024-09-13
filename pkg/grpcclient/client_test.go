// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	authgrpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/auth/mocks"
	mglog "github.com/absmach/magistrala/logger"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	thingsgrpcapi "github.com/absmach/magistrala/things/api/grpc"
	thmocks "github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestSetupAuth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registerAuthServiceServer := func(srv *grpc.Server) {
		magistrala.RegisterAuthzServiceServer(srv, authgrpcapi.NewAuthzServer(new(mocks.Service)))
		magistrala.RegisterAuthnServiceServer(srv, authgrpcapi.NewAuthnServer(new(mocks.Service)))
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
			client, handler, err := grpcclient.SetupAuthClient(context.Background(), c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestSetupThingsClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	registerThingsServiceServer := func(srv *grpc.Server) {
		magistrala.RegisterAuthzServiceServer(srv, thingsgrpcapi.NewServer(new(thmocks.Service), new(authmocks.AuthClient)))
	}
	gs := grpcserver.NewServer(ctx, cancel, "things", server.Config{Port: "12345"}, registerThingsServiceServer, mglog.NewMock())
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
			client, handler, err := grpcclient.SetupThingsClient(context.Background(), c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestSetupPolicyClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registerPolicyServiceServer := func(srv *grpc.Server) {
		magistrala.RegisterPolicyServiceServer(srv, authgrpcapi.NewPolicyServer(new(mocks.Service)))
	}
	gs := grpcserver.NewServer(ctx, cancel, "auth", server.Config{Port: "12345"}, registerPolicyServiceServer, mglog.NewMock())
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
			client, handler, err := grpcclient.SetupPolicyClient(context.Background(), c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}
