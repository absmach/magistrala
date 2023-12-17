// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"os"
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
)

var (
	svc   auth.Service
	krepo *mocks.Keys
	prepo *mocks.PolicyAgent
)

func TestMain(m *testing.M) {
	krepo = new(mocks.Keys)
	prepo = new(mocks.PolicyAgent)
	drepo := new(mocks.DomainsRepo)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))

	svc = auth.New(krepo, drepo, idProvider, t, prepo, loginDuration, refreshDuration, invalidDuration)
	startGRPCServer(svc, port)

	code := m.Run()

	os.Exit(code)
}
