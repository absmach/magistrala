// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
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

// Metadata to be used for Mainflux thing or channel for customized
// describing of particular thing or channel.
type Metadata map[string]interface{}

// User represents a Mainflux user account. Each user is identified given its
// email and password.
type User struct {
	ID       string
	Email    string
	Password string
	Metadata Metadata
	Status   string
}

// Validate returns an error if user representation is invalid.
func (u User) Validate() error {
	if !isEmail(u.Email) {
		return errors.ErrMalformedEntity
	}
	return nil
}

// UserRepository specifies an account persistence API.
type UserRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, u User) (string, error)

	// Update updates the user metadata.
	UpdateUser(ctx context.Context, u User) error

	// RetrieveByEmail retrieves user by its unique identifier (i.e. email).
	RetrieveByEmail(ctx context.Context, email string) (User, error)

	// RetrieveByID retrieves user by its unique identifier ID.
	RetrieveByID(ctx context.Context, id string) (User, error)

	// RetrieveAll retrieves all users for given array of userIDs.
	RetrieveAll(ctx context.Context, userIDs []string, pm PageMetadata) (UserPage, error)

	// UpdatePassword updates password for user with given email
	UpdatePassword(ctx context.Context, email, password string) error

	// ChangeStatus changes users status to enabled or disabled
	ChangeStatus(ctx context.Context, id, status string) error
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
