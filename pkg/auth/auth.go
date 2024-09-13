// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/absmach/magistrala"
	"google.golang.org/grpc"
)

type Session struct {
	DomainUserID string
	UserID       string
	DomainID     string
	SuperAdmin   bool
}

// AuthClient specifies a gRPC client for  authentication and authorization for magistrala services.
//
//go:generate mockery --name AuthClient --output=./mocks --filename client.go --quiet --note "Copyright (c) Abstract Machines"
type AuthClient interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, in *magistrala.IssueReq, opts ...grpc.CallOption) (*magistrala.Token, error)

	// Refresh iisues a refresh Key, returning its token value alongside.
	Refresh(ctx context.Context, in *magistrala.RefreshReq, opts ...grpc.CallOption) (*magistrala.Token, error)

	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(ctx context.Context, in *magistrala.IdentityReq, opts ...grpc.CallOption) (*magistrala.IdentityRes, error)

	// Authorize checks if the `subject` is allowed to perform the `relation` on the `object`.
	// Returns a non-nil error if the `subject` is not authorized.
	Authorize(ctx context.Context, in *magistrala.AuthorizeReq, opts ...grpc.CallOption) (*magistrala.AuthorizeRes, error)
}
