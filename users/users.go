// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/postgres"
)

type User struct {
	ID             string      `json:"id"`
	FirstName      string      `json:"first_name,omitempty"`
	LastName       string      `json:"last_name,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
	Metadata       Metadata    `json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at,omitempty"`
	UpdatedAt      time.Time   `json:"updated_at,omitempty"`
	UpdatedBy      string      `json:"updated_by,omitempty"`
	Status         Status      `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Role           Role        `json:"role,omitempty"`   // 1 for admin, 0 for normal user
	ProfilePicture string      `json:"profile_picture,omitempty"`
	DomainID       string      `json:"domain_id,omitempty"`
	Credentials    Credentials `json:"credentials,omitempty"`
	Permissions    []string    `json:"permissions,omitempty"`
}

type Credentials struct {
	UserName string `json:"user_name,omitempty"` // username or profile name
	Secret   string `json:"secret,omitempty"`    // password or token
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

	// RetrieveByUserName retrieves user by their user name
	RetrieveByUserName(ctx context.Context, userName string) (User, error)

	// RetrieveAll retrieves all users.
	RetrieveAll(ctx context.Context, pm Page) (UsersPage, error)

	// Update updates the user name and metadata.
	Update(ctx context.Context, user User) (User, error)

	// UpdateUserNames updates the User's names.
	UpdateUserNames(ctx context.Context, user User) (User, error)

	// UpdateSecret updates secret for user with given identity.
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
	UserName   string   `json:"user_name,omitempty"`
	FirstName  string   `json:"first_name,omitempty"`
	LastName   string   `json:"last_name,omitempty"`
}
