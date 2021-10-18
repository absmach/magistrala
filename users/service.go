// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	// ErrConflict indicates usage of the existing email during account
	// registration.
	ErrConflict = errors.New("email already taken")

	// ErrGroupConflict indicates group name already taken.
	ErrGroupConflict = errors.New("group already exists")

	// ErrMalformedEntity indicates malformed entity specification
	// (e.g. invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrUserNotFound indicates a non-existent user request.
	ErrUserNotFound = errors.New("non-existent user")

	// ErrScanMetadata indicates problem with metadata in db.
	ErrScanMetadata = errors.New("failed to scan metadata")

	// ErrMissingEmail indicates missing email for password reset request.
	ErrMissingEmail = errors.New("missing email for password reset")

	// ErrMissingResetToken indicates malformed or missing reset token
	// for reseting password.
	ErrMissingResetToken = errors.New("missing reset token")

	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrGetToken indicates error in getting signed token.
	ErrGetToken = errors.New("failed to fetch signed token")

	// ErrCreateUser indicates error in creating user.
	ErrCreateUser = errors.New("failed to create user")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Register creates new user account. In case of the failed registration, a
	// non-nil error value is returned.
	Register(ctx context.Context, user User) (string, error)

	// Login authenticates the user given its credentials. Successful
	// authentication generates new access token. Failed invocations are
	// identified by the non-nil error values in the response.
	Login(ctx context.Context, user User) (string, error)

	// ViewUser retrieves user info for a given user ID and an authorized token.
	ViewUser(ctx context.Context, token, id string) (User, error)

	// ViewProfile retrieves user info for a given token.
	ViewProfile(ctx context.Context, token string) (User, error)

	// ListUsers retrieves users list for a valid admin token.
	ListUsers(ctx context.Context, token string, offset, limit uint64, email string, meta Metadata) (UserPage, error)

	// UpdateUser updates the user metadata.
	UpdateUser(ctx context.Context, token string, user User) error

	// GenerateResetToken email where mail will be sent.
	// host is used for generating reset link.
	GenerateResetToken(ctx context.Context, email, host string) error

	// ChangePassword change users password for authenticated user.
	ChangePassword(ctx context.Context, authToken, password, oldPassword string) error

	// ResetPassword change users password in reset flow.
	// token can be authentication token or password reset token.
	ResetPassword(ctx context.Context, resetToken, password string) error

	//SendPasswordReset sends reset password link to email.
	SendPasswordReset(ctx context.Context, host, email, token string) error

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, meta Metadata) (UserPage, error)
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total  uint64
	Offset uint64
	Limit  uint64
	Name   string
}

// GroupPage contains a page of groups.
type GroupPage struct {
	PageMetadata
	Groups []auth.Group
}

// UserPage contains a page of users.
type UserPage struct {
	PageMetadata
	Users []User
}

var _ Service = (*usersService)(nil)

type usersService struct {
	users      UserRepository
	hasher     Hasher
	email      Emailer
	auth       mainflux.AuthServiceClient
	idProvider mainflux.IDProvider
	passRegex  *regexp.Regexp
}

// New instantiates the users service implementation
func New(users UserRepository, hasher Hasher, auth mainflux.AuthServiceClient, e Emailer, idp mainflux.IDProvider, passRegex *regexp.Regexp) Service {
	return &usersService{
		users:      users,
		hasher:     hasher,
		auth:       auth,
		email:      e,
		idProvider: idp,
		passRegex:  passRegex,
	}
}

func (svc usersService) Register(ctx context.Context, user User) (string, error) {
	if err := user.Validate(); err != nil {
		return "", err
	}
	if !svc.passRegex.MatchString(user.Password) {
		return "", ErrPasswordFormat
	}
	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}
	user.Password = hash
	uid, err := svc.idProvider.ID()
	if err != nil {
		return "", errors.Wrap(ErrCreateUser, err)
	}
	user.ID = uid
	uid, err = svc.users.Save(ctx, user)
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (svc usersService) Login(ctx context.Context, user User) (string, error) {
	dbUser, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil {
		return "", errors.Wrap(ErrUnauthorizedAccess, err)
	}
	if err := svc.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return svc.issue(ctx, dbUser.ID, dbUser.Email, auth.UserKey)
}

func (svc usersService) ViewUser(ctx context.Context, token, id string) (User, error) {
	_, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return User{
		ID:       id,
		Email:    dbUser.Email,
		Password: "",
		Metadata: dbUser.Metadata,
	}, nil
}

func (svc usersService) ViewProfile(ctx context.Context, token string) (User, error) {
	email, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, err
	}

	dbUser, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil {
		return User{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return User{
		ID:       dbUser.ID,
		Email:    email,
		Password: "",
		Metadata: dbUser.Metadata,
	}, nil
}

func (svc usersService) ListUsers(ctx context.Context, token string, offset, limit uint64, email string, m Metadata) (UserPage, error) {
	_, err := svc.identify(ctx, token)
	if err != nil {
		return UserPage{}, err
	}

	return svc.users.RetrieveAll(ctx, offset, limit, nil, email, m)
}

func (svc usersService) UpdateUser(ctx context.Context, token string, u User) error {
	email, err := svc.identify(ctx, token)
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	user := User{
		Email:    email,
		Metadata: u.Metadata,
	}
	return svc.users.UpdateUser(ctx, user)
}

func (svc usersService) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil || user.Email == "" {
		return ErrUserNotFound
	}
	t, err := svc.issue(ctx, user.ID, user.Email, auth.RecoveryKey)
	if err != nil {
		return errors.Wrap(ErrRecoveryToken, err)
	}
	return svc.SendPasswordReset(ctx, host, email, t)
}

func (svc usersService) ResetPassword(ctx context.Context, resetToken, password string) error {
	email, err := svc.identify(ctx, resetToken)
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	u, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil || u.Email == "" {
		return ErrUserNotFound
	}
	if !svc.passRegex.MatchString(password) {
		return ErrPasswordFormat
	}
	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, email, password)
}

func (svc usersService) ChangePassword(ctx context.Context, authToken, password, oldPassword string) error {
	email, err := svc.identify(ctx, authToken)
	if err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}
	if !svc.passRegex.MatchString(password) {
		return ErrPasswordFormat
	}
	u := User{
		Email:    email,
		Password: oldPassword,
	}
	if _, err := svc.Login(ctx, u); err != nil {
		return ErrUnauthorizedAccess
	}
	u, err = svc.users.RetrieveByEmail(ctx, email)
	if err != nil || u.Email == "" {
		return ErrUserNotFound
	}

	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, email, password)
}

func (svc usersService) SendPasswordReset(_ context.Context, host, email, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, token)
}

func (svc usersService) ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, m Metadata) (UserPage, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return UserPage{}, err
	}

	userIDs, err := svc.members(ctx, token, groupID, offset, limit)
	if err != nil {
		return UserPage{}, err
	}

	if len(userIDs) == 0 {
		return UserPage{
			Users: []User{},
			PageMetadata: PageMetadata{
				Total:  0,
				Offset: offset,
				Limit:  limit,
			},
		}, nil
	}

	return svc.users.RetrieveAll(ctx, offset, limit, userIDs, "", m)
}

// Auth helpers
func (svc usersService) issue(ctx context.Context, id, email string, keyType uint32) (string, error) {
	key, err := svc.auth.Issue(ctx, &mainflux.IssueReq{Id: id, Email: email, Type: keyType})
	if err != nil {
		return "", errors.Wrap(ErrUserNotFound, err)
	}
	return key.GetValue(), nil
}

func (svc usersService) identify(ctx context.Context, token string) (string, error) {
	identity, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return identity.GetEmail(), nil
}

func (svc usersService) members(ctx context.Context, token, groupID string, limit, offset uint64) ([]string, error) {
	req := mainflux.MembersReq{
		Token:   token,
		GroupID: groupID,
		Offset:  offset,
		Limit:   limit,
		Type:    "users",
	}

	res, err := svc.auth.Members(ctx, &req)
	if err != nil {
		return nil, err
	}
	return res.Members, nil
}
