// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"os"
	"testing"

	"github.com/absmach/magistrala/auth/mocks"
)

var svc *mocks.Service

func TestMain(m *testing.M) {
	svc = new(mocks.Service)
	startGRPCServer(svc, port)

	code := m.Run()

	os.Exit(code)
}
