// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"net/mail"
	"time"

	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/postgres"
)

type User struct {
	ID             string      `json:"id"`
	FirstName      string      `json:"first_name,omitempty"`
	LastName       string      `json:"last_name,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
	Metadata       Metadata    `json:"metadata,omitempty"`
	Status         Status      `json:"status"`                    // 0 for enabled, 1 for disabled
	Role           Role        `json:"role"`                      // 0 for normal user, 1 for admin
	ProfilePicture string      `json:"profile_picture,omitempty"` // profile picture URL
	Credentials    Credentials `json:"credentials,omitempty"`
	Permissions    []string    `json:"permissions,omitempty"`
	Email          string      `json:"email,omitempty"`
	CreatedAt      time.Time   `json:"created_at,omitempty"`
	UpdatedAt      time.Time   `json:"updated_at,omitempty"`
	UpdatedBy      string      `json:"updated_by,omitempty"`
}

type Credentials struct {
	Username string `json:"username,omitempty"` // username or profile name
	Secret   string `json:"secret,omitempty"`   // password or token
}

type UsersPage struct {
	Page
	Users []User
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

// MembersPage contains page related metadata as well as list of members that
// belong to this page.
type MembersPage struct {
	Page
	Members []User
}

// UserRepository struct implements the Repository interface.
type UserRepository struct {
	DB postgres.Database
}

//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// RetrieveByID retrieves user by their unique ID.
	RetrieveByID(ctx context.Context, id string) (User, error)

	// RetrieveAll retrieves all users.
	RetrieveAll(ctx context.Context, pm Page) (UsersPage, error)

	// RetrieveByEmail retrieves user by its unique credentials.
	RetrieveByEmail(ctx context.Context, email string) (User, error)

	// RetrieveByUsername retrieves user by its unique credentials.
	RetrieveByUsername(ctx context.Context, username string) (User, error)

	// Update updates the user name and metadata.
	Update(ctx context.Context, user User) (User, error)

	// UpdateUsername updates the User's names.
	UpdateUsername(ctx context.Context, user User) (User, error)

	// UpdateSecret updates secret for user with given email.
	UpdateSecret(ctx context.Context, user User) (User, error)

	// ChangeStatus changes user status to enabled or disabled
	ChangeStatus(ctx context.Context, user User) (User, error)

	// Delete deletes user with given id
	Delete(ctx context.Context, id string) error

	// Searchusers retrieves users based on search criteria.
	SearchUsers(ctx context.Context, pm Page) (UsersPage, error)

	// RetrieveAllByIDs retrieves for given user IDs .
	RetrieveAllByIDs(ctx context.Context, pm Page) (UsersPage, error)

	CheckSuperAdmin(ctx context.Context, adminID string) error

	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, user User) (User, error)
}

// Validate returns an error if user representation is invalid.
func (u User) Validate() error {
	if !isEmail(u.Email) {
		return errors.ErrMalformedEntity
	}
	return nil
}

func isEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// Page contains page metadata that helps navigation.
type Page struct {
	Total      uint64   `json:"total"`
	Offset     uint64   `json:"offset"`
	Limit      uint64   `json:"limit"`
	Id         string   `json:"id,omitempty"`
	Order      string   `json:"order,omitempty"`
	Dir        string   `json:"dir,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Domain     string   `json:"domain,omitempty"`
	Tag        string   `json:"tag,omitempty"`
	Permission string   `json:"permission,omitempty"`
	Status     Status   `json:"status,omitempty"`
	IDs        []string `json:"ids,omitempty"`
	Role       Role     `json:"-"`
	ListPerms  bool     `json:"-"`
	Username   string   `json:"username,omitempty"`
	FirstName  string   `json:"first_name,omitempty"`
	LastName   string   `json:"last_name,omitempty"`
	Email      string   `json:"email,omitempty"`
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// Register creates new user. In case of the failed registration, a
	// non-nil error value is returned.
	Register(ctx context.Context, session authn.Session, user User, selfRegister bool) (User, error)

	// View retrieves user info for a given user ID and an authorized token.
	View(ctx context.Context, session authn.Session, id string) (User, error)

	// ViewProfile retrieves user info for a given token.
	ViewProfile(ctx context.Context, session authn.Session) (User, error)

	// ListUsers retrieves users list for a valid auth token.
	ListUsers(ctx context.Context, session authn.Session, pm Page) (UsersPage, error)

	// ListMembers retrieves everything that is assigned to a group/client identified by objectID.
	ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm Page) (MembersPage, error)

	// SearchUsers searches for users with provided filters for a valid auth token.
	SearchUsers(ctx context.Context, pm Page) (UsersPage, error)

	// Update updates the user's name and metadata.
	Update(ctx context.Context, session authn.Session, user User) (User, error)

	// UpdateTags updates the user's tags.
	UpdateTags(ctx context.Context, session authn.Session, user User) (User, error)

	// UpdateEmail updates the user's email.
	UpdateEmail(ctx context.Context, session authn.Session, id, email string) (User, error)

	// UpdateUsername updates the user's username.
	UpdateUsername(ctx context.Context, session authn.Session, id, username string) (User, error)

	// UpdateProfilePicture updates the user's profile picture.
	UpdateProfilePicture(ctx context.Context, session authn.Session, user User) (User, error)

	// GenerateResetToken email where mail will be sent.
	// host is used for generating reset link.
	GenerateResetToken(ctx context.Context, email, host string) error

	// UpdateSecret updates the user's secret.
	UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (User, error)

	// ResetSecret change users secret in reset flow.
	// token can be authentication token or secret reset token.
	ResetSecret(ctx context.Context, session authn.Session, secret string) error

	// SendPasswordReset sends reset password link to email.
	SendPasswordReset(ctx context.Context, host, email, user, token string) error

	// UpdateRole updates the user's Role.
	UpdateRole(ctx context.Context, session authn.Session, user User) (User, error)

	// Enable logically enables the user identified with the provided ID.
	Enable(ctx context.Context, session authn.Session, id string) (User, error)

	// Disable logically disables the user identified with the provided ID.
	Disable(ctx context.Context, session authn.Session, id string) (User, error)

	// Delete deletes user with given ID.
	Delete(ctx context.Context, session authn.Session, id string) error

	// Identify returns the user id from the given token.
	Identify(ctx context.Context, session authn.Session) (string, error)

	// IssueToken issues a new access and refresh token when provided with either a username or email.
	IssueToken(ctx context.Context, identity, secret string) (*grpcTokenV1.Token, error)

	// RefreshToken refreshes expired access tokens.
	// After an access token expires, the refresh token is used to get
	// a new pair of access and refresh tokens.
	RefreshToken(ctx context.Context, session authn.Session, refreshToken string) (*grpcTokenV1.Token, error)

	// OAuthCallback handles the callback from any supported OAuth provider.
	// It processes the OAuth tokens and either signs in or signs up the user based on the provided state.
	OAuthCallback(ctx context.Context, user User) (User, error)

	// OAuthAddUserPolicy adds a policy to the user for an OAuth request.
	OAuthAddUserPolicy(ctx context.Context, user User) error
}
