// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/readersclient"
	"google.golang.org/grpc"
)

// NewReadersClient returns a reader gRPC client.
//
// Deprecated: use readersclient.New.
func NewReadersClient(conn *grpc.ClientConn, timeout time.Duration) grpcReadersV1.ReadersServiceClient {
	return readersclient.New(conn, timeout)
}
