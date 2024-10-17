// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
)

type Session struct {
	DomainUserID string
	UserID       string
	DomainID     string
	SuperAdmin   bool
}

// Authn is magistrala authentication library.
//
//go:generate mockery --name Authentication --output=./mocks --filename authn.go --quiet --note "Copyright (c) Abstract Machines"
type Authentication interface {
	Authenticate(ctx context.Context, token string) (Session, error)
}
