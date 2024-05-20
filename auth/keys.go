// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrKeyExpired indicates that the Key is expired.
var ErrKeyExpired = errors.New("use of expired key")

type Token struct {
	AccessToken  string // AccessToken contains the security credentials for a login session and identifies the client.
	RefreshToken string // RefreshToken is a credential artifact that OAuth can use to get a new access token without client interaction.
	AccessType   string // AccessType is the specific type of access token issued. It can be Bearer, Client or Basic.
}

type KeyType uint32

const (
	// AccessKey is temporary User key received on successful login.
	AccessKey KeyType = iota
	// RefreshKey is a temporary User key used to generate a new access key.
	RefreshKey
	// RecoveryKey represents a key for resseting password.
	RecoveryKey
	// APIKey enables the one to act on behalf of the user.
	APIKey
	// InvitationKey is a key for inviting new users.
	InvitationKey
)

func (kt KeyType) String() string {
	switch kt {
	case AccessKey:
		return "access"
	case RefreshKey:
		return "refresh"
	case RecoveryKey:
		return "recovery"
	case APIKey:
		return "API"
	default:
		return "unknown"
	}
}

// Key represents API key.
type Key struct {
	ID        string    `json:"id,omitempty"`
	Type      KeyType   `json:"type,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	Subject   string    `json:"subject,omitempty"` // user ID
	User      string    `json:"user,omitempty"`
	Domain    string    `json:"domain,omitempty"` // domain user ID
	IssuedAt  time.Time `json:"issued_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (key Key) String() string {
	return fmt.Sprintf(`{
	id: %s,
	type: %s,
	issuer_id: %s,
	subject: %s,
	user: %s,
	domain: %s,
	iat: %v,
	eat: %v
}`, key.ID, key.Type, key.Issuer, key.Subject, key.User, key.Domain, key.IssuedAt, key.ExpiresAt)
}

// Expired verifies if the key is expired.
func (key Key) Expired() bool {
	if key.Type == APIKey && key.ExpiresAt.IsZero() {
		return false
	}
	return key.ExpiresAt.UTC().Before(time.Now().UTC())
}

// KeyRepository specifies Key persistence API.
//
//go:generate mockery --name KeyRepository --output=./mocks --filename keys.go --quiet --note "Copyright (c) Abstract Machines"
type KeyRepository interface {
	// Save persists the Key. A non-nil error is returned to indicate
	// operation failure
	Save(ctx context.Context, key Key) (id string, err error)

	// Retrieve retrieves Key by its unique identifier.
	Retrieve(ctx context.Context, issuer string, id string) (key Key, err error)

	// Remove removes Key with provided ID.
	Remove(ctx context.Context, issuer string, id string) error
}
