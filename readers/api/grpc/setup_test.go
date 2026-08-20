// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"net"
	"testing"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	grpcapi "github.com/absmach/magistrala/readers/api/grpc"
	"github.com/absmach/magistrala/readers/mocks"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// newTestServer starts a fresh gRPC server backed by a fresh mock
// repository, bound to an OS-assigned ephemeral port. Every server-hitting
// test function calls this itself rather than sharing one package-level
// mock and a fixed port (Q3): a shared mock let one test function's
// svc.On/repoCall.Unset race an in-flight call the server goroutine was
// still serving on behalf of another test function, and a fixed port could
// still be held by a previous whole-package run's process when the next one
// started, so go test ./readers/api/grpc/... failed intermittently even
// though every test passed in isolation. Giving each test its own mock and
// its own server on its own port removes both sources of cross-test
// interference.
func newTestServer(t *testing.T) (*mocks.MessageRepository, string) {
	t.Helper()

	svc := new(mocks.MessageRepository)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	server := grpc.NewServer()
	grpcReadersV1.RegisterReadersServiceServer(server, grpcapi.NewReadersServer(svc))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.GracefulStop)

	return svc, listener.Addr().String()
}
