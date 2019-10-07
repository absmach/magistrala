// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	svc = newService()
	startGRPCServer(svc, port)

	code := m.Run()

	os.Exit(code)
}
