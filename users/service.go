// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"errors"
)

var (
	// ErrConflict indicates usage of the existing email during account
	// registration.
	ErrConflict = errors.New("email already taken")

	// ErrMalformedEntity indicates malformed entity specification
	// (e.g. invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request
	ErrNotFound = errors.New("non-existent entity")

	// ErrUserNotFound indicates a non-existent user request
	ErrUserNotFound = errors.New("non-existent user")

	// ErrScanMetadata indicates problem with metadata in db
	ErrScanMetadata = errors.New("Failed to scan metadata")

	// ErrMissingEmail indicates missing email for password reset request
	ErrMissingEmail = errors.New("missing email for password reset")

	// ErrMissingResetToken indicates malformed or missing reset token
	// for reseting password
	ErrMissingResetToken = errors.New("error missing reset token")

	// ErrGeneratingResetToken indicates error in generating password recovery
	// token
	ErrGeneratingResetToken = errors.New("error missing reset token")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Register creates new user account. In case of the failed registration, a
	// non-nil error value is returned.
	Register(context.Context, User) error

	// Login authenticates the user given its credentials. Successful
	// authentication generates new access token. Failed invocations are
	// identified by the non-nil error values in the response.
	Login(context.Context, User) (string, error)

	// Identify validates user's token. If token is valid, user's id
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error values are returned in response.
	Identify(string) (string, error)

	// Get authenticated user info for the given token
	UserInfo(ctx context.Context, token string) (User, error)

	// UpdateUser updates the user metadata
	UpdateUser(ctx context.Context, token string, user User) error

	// GenerateResetToken email where mail will be sent.
	// host is used for generating reset link.
	GenerateResetToken(_ context.Context, email, host string) error

	// ChangePassword change users password for authenticated user.
	ChangePassword(_ context.Context, authToken, password, oldPassword string) error

	// ResetPassword change users password in reset flow.
	// token can be authentication token or password reset token.
	ResetPassword(_ context.Context, resetToken, password string) error

	//SendPasswordReset sends reset password link to email
	SendPasswordReset(_ context.Context, host, email, token string) error
}

var _ Service = (*usersService)(nil)

type usersService struct {
	users  UserRepository
	hasher Hasher
	idp    IdentityProvider
	token  Tokenizer
	email  Emailer
}

// New instantiates the users service implementation
func New(users UserRepository, hasher Hasher, idp IdentityProvider, m Emailer, t Tokenizer) Service {
	return &usersService{users: users, hasher: hasher, idp: idp, email: m, token: t}
}

func (svc usersService) Register(ctx context.Context, user User) error {
	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return ErrMalformedEntity
	}

	user.Password = hash
	return svc.users.Save(ctx, user)
}

func (svc usersService) Login(ctx context.Context, user User) (string, error) {
	dbUser, err := svc.users.RetrieveByID(ctx, user.Email)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	if err := svc.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", ErrUnauthorizedAccess
	}

	return svc.idp.TemporaryKey(user.Email)
}

func (svc usersService) Identify(token string) (string, error) {
	id, err := svc.idp.Identity(token)
	if err != nil {
		return "", ErrUnauthorizedAccess
	}
	return id, nil
}

func (svc usersService) UserInfo(ctx context.Context, token string) (User, error) {
	id, err := svc.idp.Identity(token)
	if err != nil {
		return User{}, ErrUnauthorizedAccess
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, ErrUnauthorizedAccess
	}

	return User{
		Email:    id,
		Password: "",
		Metadata: dbUser.Metadata,
	}, nil

}

func (svc usersService) UpdateUser(ctx context.Context, token string, u User) error {
	email, err := svc.idp.Identity(token)
	if err != nil || email != u.Email {
		return ErrUnauthorizedAccess
	}

	user := User{
		Email:    email,
		Metadata: u.Metadata,
	}

	return svc.users.UpdateUser(ctx, user)
}

func (svc usersService) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByID(ctx, email)
	if err != nil || user.Email == "" {
		return ErrUserNotFound
	}

	tok, err := svc.token.Generate(email, 0)
	if err != nil {
		return ErrGeneratingResetToken
	}
	return svc.SendPasswordReset(ctx, host, email, tok)
}

func (svc usersService) ResetPassword(ctx context.Context, resetToken, password string) error {
	email, err := svc.token.Verify(resetToken)
	if err != nil {
		return err
	}

	u, err := svc.users.RetrieveByID(ctx, email)
	if err != nil || u.Email == "" {
		return ErrUserNotFound
	}

	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, email, password)
}

func (svc usersService) ChangePassword(ctx context.Context, authToken, password, oldPassword string) error {
	email, err := svc.idp.Identity(authToken)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	u := User{
		Email:    email,
		Password: oldPassword,
	}
	if _, err = svc.Login(ctx, u); err != nil {
		return ErrUnauthorizedAccess
	}

	u, err = svc.users.RetrieveByID(ctx, email)
	if err != nil || u.Email == "" {
		return ErrUserNotFound
	}

	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, email, password)
}

// SendPasswordReset sends password recovery link to user
func (svc usersService) SendPasswordReset(_ context.Context, host, email, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, token)
}
