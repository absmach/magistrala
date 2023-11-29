// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
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

// Credentials represent client credentials: its
// "identity" which can be a username, email, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}

// Client represents generic Client.
type Client struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Owner       string      `json:"owner,omitempty"` // nullable
	Credentials Credentials `json:"credentials,omitempty"`
	Metadata    Metadata    `json:"metadata,omitempty"`
	CreatedAt   time.Time   `json:"created_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
	UpdatedBy   string      `json:"updated_by,omitempty"`
	Status      Status      `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Role        Role        `json:"role,omitempty"`   // 1 for admin, 0 for normal user
	Permissions []string    `json:"permissions,omitempty"`
}

// ClientsPage contains page related metadata as well as list
// of Clients that belong to the page.
type ClientsPage struct {
	Page
	Clients []Client
}

// MembersPage contains page related metadata as well as list of members that
// belong to this page.
type MembersPage struct {
	Page
	Members []Client
}

// Repository specifies an account persistence API.
type Repository interface {
	// RetrieveByID retrieves client by its unique ID.
	RetrieveByID(ctx context.Context, id string) (Client, error)

	// RetrieveByIdentity retrieves client by its unique credentials
	RetrieveByIdentity(ctx context.Context, identity string) (Client, error)

	// RetrieveAll retrieves all clients.
	RetrieveAll(ctx context.Context, pm Page) (ClientsPage, error)

	// RetrieveAllBasicInfo list all clients only with basic information.
	RetrieveAllBasicInfo(ctx context.Context, pm Page) (ClientsPage, error)

	// RetrieveAllByIDs retrieves for given client IDs .
	RetrieveAllByIDs(ctx context.Context, pm Page) (ClientsPage, error)

	// Update updates the client name and metadata.
	Update(ctx context.Context, client Client) (Client, error)

	// UpdateTags updates the client tags.
	UpdateTags(ctx context.Context, client Client) (Client, error)

	// UpdateIdentity updates identity for client with given id.
	UpdateIdentity(ctx context.Context, client Client) (Client, error)

	// UpdateSecret updates secret for client with given identity.
	UpdateSecret(ctx context.Context, client Client) (Client, error)

	// UpdateOwner updates owner for client with given id.
	UpdateOwner(ctx context.Context, client Client) (Client, error)

	// UpdateRole updates role for client with given id.
	UpdateRole(ctx context.Context, client Client) (Client, error)

	// ChangeStatus changes client status to enabled or disabled
	ChangeStatus(ctx context.Context, client Client) (Client, error)
}

// Validate returns an error if client representation is invalid.
func (u Client) Validate() error {
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
