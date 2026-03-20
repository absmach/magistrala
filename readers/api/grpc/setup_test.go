// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"os"
	"testing"

	"github.com/absmach/supermq/readers/mocks"
)

var svc *mocks.MessageRepository

func TestMain(m *testing.M) {
	svc = new(mocks.MessageRepository)
	server := startGRPCServer(svc, port)

	code := m.Run()

	server.GracefulStop()

	os.Exit(code)
}
