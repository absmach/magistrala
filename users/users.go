// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/postgres"
	"golang.org/x/net/idna"
)

const (
	maxLocalLen  = 64
	maxDomainLen = 255
	maxTLDLen    = 24 // longest TLD currently in existence

	atSeparator  = "@"
	dotSeparator = "."
)

var (
	userRegexp    = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+/=?^_`{|}~.-]+$")
	hostRegexp    = regexp.MustCompile(`^[^\s]+\.[^\s]+$`)
	userDotRegexp = regexp.MustCompile("(^[.]{1})|([.]{1}$)|([.]{2,})")
)

type User struct {
	ID             string      `json:"id"`
	Name           string      `json:"name,omitempty"`
	UserName       string      `json:"user_name,omitempty"`
	FirstName      string      `json:"first_name,omitempty"`
	LastName       string      `json:"last_name,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
	Credentials    Credentials `json:"credentials,omitempty"`
	Metadata       Metadata    `json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at,omitempty"`
	UpdatedAt      time.Time   `json:"updated_at,omitempty"`
	UpdatedBy      string      `json:"updated_by,omitempty"`
	Status         Status      `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Role           Role        `json:"role,omitempty"`   // 1 for admin, 0 for normal user
	Permissions    []string    `json:"permissions,omitempty"`
	ProfilePicture string      `json:"profile_picture,omitempty"`
	DomainID       string      `json:"domain_id,omitempty"`
}

type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
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

	// RetrieveByIdentity retrieves user by its unique credentials.
	RetrieveByIdentity(ctx context.Context, identity string) (User, error)

	// RetrieveByUserName retrieves user by their user name
	RetrieveByUserName(ctx context.Context, userName string) (User, error)

	// RetrieveAll retrieves all users.
	RetrieveAll(ctx context.Context, pm Page) (UsersPage, error)

	// Update updates the user name and metadata.
	Update(ctx context.Context, user User) (User, error)

	// UpdateTags updates the user tags.
	UpdateTags(ctx context.Context, user User) (User, error)

	// UpdateIdentity updates identity for user with given id.
	UpdateIdentity(ctx context.Context, user User) (User, error)

	// UpdateUserNames updates the User's names.
	UpdateUserNames(ctx context.Context, user User) (User, error)

	// UpdateProfilePicture updates profile picture for user with given ID.
	UpdateProfilePicture(ctx context.Context, user User) (User, error)

	// UpdateSecret updates secret for user with given identity.
	UpdateSecret(ctx context.Context, user User) (User, error)

	// UpdateRole updates role for user with given id.
	UpdateRole(ctx context.Context, user User) (User, error)

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
	if !isEmail(u.Credentials.Identity) {
		return errors.ErrMalformedEntity
	}
	return nil
}

func isEmail(email string) bool {
	if email == "" {
		return false
	}

	es := strings.Split(email, atSeparator)
	if len(es) != 2 {
		return false
	}
	local, host := es[0], es[1]

	if local == "" || len(local) > maxLocalLen {
		return false
	}

	hs := strings.Split(host, dotSeparator)
	if len(hs) < 2 {
		return false
	}
	domain, ext := hs[0], hs[1]

	// Check subdomain and validate
	if len(hs) > 2 {
		if domain == "" {
			return false
		}

		for i := 1; i < len(hs)-1; i++ {
			sub := hs[i]
			if sub == "" {
				return false
			}
			domain = fmt.Sprintf("%s.%s", domain, sub)
		}

		ext = hs[len(hs)-1]
	}

	if domain == "" || len(domain) > maxDomainLen {
		return false
	}
	if ext == "" || len(ext) > maxTLDLen {
		return false
	}

	punyLocal, err := idna.ToASCII(local)
	if err != nil {
		return false
	}
	punyHost, err := idna.ToASCII(host)
	if err != nil {
		return false
	}

	if userDotRegexp.MatchString(punyLocal) || !userRegexp.MatchString(punyLocal) || !hostRegexp.MatchString(punyHost) {
		return false
	}

	return true
}

// Page contains page metadata that helps navigation.
type Page struct {
	Total      uint64   `json:"total"`
	Offset     uint64   `json:"offset"`
	Limit      uint64   `json:"limit"`
	Name       string   `json:"name,omitempty"`
	Id         string   `json:"id,omitempty"`
	Order      string   `json:"order,omitempty"`
	Dir        string   `json:"dir,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Domain     string   `json:"domain,omitempty"`
	Tag        string   `json:"tag,omitempty"`
	Permission string   `json:"permission,omitempty"`
	Status     Status   `json:"status,omitempty"`
	IDs        []string `json:"ids,omitempty"`
	Identity   string   `json:"identity,omitempty"`
	Role       Role     `json:"-"`
	ListPerms  bool     `json:"-"`
	UserName   string   `json:"user_name,omitempty"`
	FirstName  string   `json:"first_name,omitempty"`
	LastName   string   `json:"last_name,omitempty"`
}
