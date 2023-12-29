// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/clients"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// RegisterClient creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	RegisterClient(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// ViewClient retrieves client info for a given client ID and an authorized token.
	ViewClient(ctx context.Context, token, id string) (clients.Client, error)

	// ViewProfile retrieves client info for a given token.
	ViewProfile(ctx context.Context, token string) (clients.Client, error)

	// ListClients retrieves clients list for a valid auth token.
	ListClients(ctx context.Context, token string, pm clients.Page) (clients.ClientsPage, error)

	// ListMembers retrieves everything that is assigned to a group/thing identified by objectID.
	ListMembers(ctx context.Context, token, objectKind, objectID string, pm clients.Page) (clients.MembersPage, error)

	// UpdateClient updates the client's name and metadata.
	UpdateClient(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// UpdateClientTags updates the client's tags.
	UpdateClientTags(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// UpdateClientIdentity updates the client's identity.
	UpdateClientIdentity(ctx context.Context, token, id, identity string) (clients.Client, error)

	// GenerateResetToken email where mail will be sent.
	// host is used for generating reset link.
	GenerateResetToken(ctx context.Context, email, host string) error

	// UpdateClientSecret updates the client's secret.
	UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (clients.Client, error)

	// ResetSecret change users secret in reset flow.
	// token can be authentication token or secret reset token.
	ResetSecret(ctx context.Context, resetToken, secret string) error

	// SendPasswordReset sends reset password link to email.
	SendPasswordReset(ctx context.Context, host, email, user, token string) error

	// UpdateClientRole updates the client's Role.
	UpdateClientRole(ctx context.Context, token string, client clients.Client) (clients.Client, error)

	// EnableClient logically enableds the client identified with the provided ID.
	EnableClient(ctx context.Context, token, id string) (clients.Client, error)

	// DisableClient logically disables the client identified with the provided ID.
	DisableClient(ctx context.Context, token, id string) (clients.Client, error)

	// Identify returns the client id from the given token.
	Identify(ctx context.Context, tkn string) (string, error)

	// IssueToken issues a new access and refresh token.
	IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error)

	// RefreshToken refreshes expired access tokens.
	// After an access token expires, the refresh token is used to get
	// a new pair of access and refresh tokens.
	RefreshToken(ctx context.Context, accessToken, domainID string) (*magistrala.Token, error)
}
