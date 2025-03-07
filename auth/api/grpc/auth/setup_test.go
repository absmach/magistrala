// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"os"
	"testing"

	"github.com/absmach/supermq/auth/mocks"
)

var svc *mocks.Service

func TestMain(m *testing.M) {
	svc = new(mocks.Service)
	server := startGRPCServer(svc, port)

	code := m.Run()

	server.GracefulStop()

	os.Exit(code)
}
