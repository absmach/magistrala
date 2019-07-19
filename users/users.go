//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package users

import (
	"context"
	"regexp"
	"strings"
)

var (
	userRegexp    = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+/=?^_`{|}~.-]+$")
	hostRegexp    = regexp.MustCompile("^[^\\s]+\\.[^\\s]+$")
	userDotRegexp = regexp.MustCompile("(^[.]{1})|([.]{1}$)|([.]{2,})")
)

// User represents a Mainflux user account. Each user is identified given its
// email and password.
type User struct {
	Email    string
	Password string
}

// Validate returns an error if user representation is invalid.
func (u User) Validate() error {
	if u.Email == "" || u.Password == "" {
		return ErrMalformedEntity
	}

	if !isEmail(u.Email) {
		return ErrMalformedEntity
	}

	return nil
}

// UserRepository specifies an account persistence API.
type UserRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(context.Context, User) error

	// RetrieveByID retrieves user by its unique identifier (i.e. email).
	RetrieveByID(context.Context, string) (User, error)
}

func isEmail(email string) bool {
	if len(email) < 6 || len(email) > 254 {
		return false
	}

	at := strings.LastIndex(email, "@")
	if at <= 0 || at > len(email)-3 {
		return false
	}

	user := email[:at]
	host := email[at+1:]

	if len(user) > 64 {
		return false
	}

	if userDotRegexp.MatchString(user) || !userRegexp.MatchString(user) || !hostRegexp.MatchString(host) {
		return false
	}

	return true
}
