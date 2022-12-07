// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	memberRelationKey = "member"
	authoritiesObjKey = "authorities"
	usersObjKey       = "users"
	EnabledStatusKey  = "enabled"
	DisabledStatusKey = "disabled"
	AllStatusKey      = "all"
)

var (
	// ErrMissingResetToken indicates malformed or missing reset token
	// for reseting password.
	ErrMissingResetToken = errors.New("missing reset token")

	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrGetToken indicates error in getting signed token.
	ErrGetToken = errors.New("failed to fetch signed token")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")

	// ErrAlreadyEnabledUser indicates the user is already enabled.
	ErrAlreadyEnabledUser = errors.New("the user is already enabled")

	// ErrAlreadyDisabledUser indicates the user is already disabled.
	ErrAlreadyDisabledUser = errors.New("the user is already disabled")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Register creates new user account. In case of the failed registration, a
	// non-nil error value is returned. The user registration is only allowed
	// for admin.
	Register(ctx context.Context, token string, user User) (string, error)

	// Login authenticates the user given its credentials. Successful
	// authentication generates new access token. Failed invocations are
	// identified by the non-nil error values in the response.
	Login(ctx context.Context, user User) (string, error)

	// ViewUser retrieves user info for a given user ID and an authorized token.
	ViewUser(ctx context.Context, token, id string) (User, error)

	// ViewProfile retrieves user info for a given token.
	ViewProfile(ctx context.Context, token string) (User, error)

	// ListUsers retrieves users list for a valid admin token.
	ListUsers(ctx context.Context, token string, pm PageMetadata) (UserPage, error)

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

	// SendPasswordReset sends reset password link to email.
	SendPasswordReset(ctx context.Context, host, email, token string) error

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID string, pm PageMetadata) (UserPage, error)

	// EnableUser logically enableds the user identified with the provided ID
	EnableUser(ctx context.Context, token, id string) error

	// DisableUser logically disables the user identified with the provided ID
	DisableUser(ctx context.Context, token, id string) error
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Email    string
	Status   string
	Metadata Metadata
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

func (svc usersService) Register(ctx context.Context, token string, user User) (string, error) {
	if err := svc.checkAuthz(ctx, token); err != nil {
		return "", err
	}

	if err := user.Validate(); err != nil {
		return "", err
	}
	if !svc.passRegex.MatchString(user.Password) {
		return "", ErrPasswordFormat
	}

	uid, err := svc.idProvider.ID()
	if err != nil {
		return "", err
	}
	user.ID = uid

	if err := svc.claimOwnership(ctx, user.ID, usersObjKey, memberRelationKey); err != nil {
		return "", err
	}

	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}
	user.Password = hash
	if user.Status == "" {
		user.Status = EnabledStatusKey
	}

	if user.Status != AllStatusKey &&
		user.Status != EnabledStatusKey &&
		user.Status != DisabledStatusKey {
		return "", apiutil.ErrInvalidStatus
	}

	uid, err = svc.users.Save(ctx, user)
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (svc usersService) checkAuthz(ctx context.Context, token string) error {
	if err := svc.authorize(ctx, "*", "user", "create"); err == nil {
		return nil
	}
	if token == "" {
		return errors.ErrAuthentication
	}

	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	return svc.authorize(ctx, ir.id, authoritiesObjKey, memberRelationKey)
}

func (svc usersService) Login(ctx context.Context, user User) (string, error) {
	dbUser, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	if err := svc.hasher.Compare(user.Password, dbUser.Password); err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	return svc.issue(ctx, dbUser.ID, dbUser.Email, auth.LoginKey)
}

func (svc usersService) ViewUser(ctx context.Context, token, id string) (User, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return User{}, err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(errors.ErrNotFound, err)
	}

	return User{
		ID:       id,
		Email:    dbUser.Email,
		Password: "",
		Metadata: dbUser.Metadata,
		Status:   dbUser.Status,
	}, nil
}

func (svc usersService) ViewProfile(ctx context.Context, token string) (User, error) {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return User{}, err
	}

	dbUser, err := svc.users.RetrieveByEmail(ctx, ir.email)
	if err != nil {
		return User{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return User{
		ID:       dbUser.ID,
		Email:    ir.email,
		Metadata: dbUser.Metadata,
	}, nil
}

func (svc usersService) ListUsers(ctx context.Context, token string, pm PageMetadata) (UserPage, error) {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return UserPage{}, err
	}

	if err := svc.authorize(ctx, id.id, "authorities", "member"); err != nil {
		return UserPage{}, err
	}
	return svc.users.RetrieveAll(ctx, pm.Status, pm.Offset, pm.Limit, nil, pm.Email, pm.Metadata)
}

func (svc usersService) UpdateUser(ctx context.Context, token string, u User) error {
	ir, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	user := User{
		Email:    ir.email,
		Metadata: u.Metadata,
	}
	return svc.users.UpdateUser(ctx, user)
}

func (svc usersService) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil || user.Email == "" {
		return errors.ErrNotFound
	}
	t, err := svc.issue(ctx, user.ID, user.Email, auth.RecoveryKey)
	if err != nil {
		return errors.Wrap(ErrRecoveryToken, err)
	}
	return svc.SendPasswordReset(ctx, host, email, t)
}

func (svc usersService) ResetPassword(ctx context.Context, resetToken, password string) error {
	ir, err := svc.identify(ctx, resetToken)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}
	u, err := svc.users.RetrieveByEmail(ctx, ir.email)
	if err != nil {
		return err
	}
	if u.Email == "" {
		return errors.ErrNotFound
	}
	if !svc.passRegex.MatchString(password) {
		return ErrPasswordFormat
	}
	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, ir.email, password)
}

func (svc usersService) ChangePassword(ctx context.Context, authToken, password, oldPassword string) error {
	ir, err := svc.identify(ctx, authToken)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}
	if !svc.passRegex.MatchString(password) {
		return ErrPasswordFormat
	}
	u := User{
		Email:    ir.email,
		Password: oldPassword,
	}
	if _, err := svc.Login(ctx, u); err != nil {
		return errors.ErrAuthentication
	}
	u, err = svc.users.RetrieveByEmail(ctx, ir.email)
	if err != nil || u.Email == "" {
		return errors.ErrNotFound
	}

	password, err = svc.hasher.Hash(password)
	if err != nil {
		return err
	}
	return svc.users.UpdatePassword(ctx, ir.email, password)
}

func (svc usersService) SendPasswordReset(_ context.Context, host, email, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, token)
}

func (svc usersService) ListMembers(ctx context.Context, token, groupID string, pm PageMetadata) (UserPage, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return UserPage{}, err
	}

	userIDs, err := svc.members(ctx, token, groupID, pm.Offset, pm.Limit)
	if err != nil {
		return UserPage{}, err
	}

	if len(userIDs) == 0 {
		return UserPage{
			Users: []User{},
			PageMetadata: PageMetadata{
				Total:  0,
				Offset: pm.Offset,
				Limit:  pm.Limit,
			},
		}, nil
	}

	return svc.users.RetrieveAll(ctx, pm.Status, pm.Offset, pm.Limit, userIDs, pm.Email, pm.Metadata)
}

func (svc usersService) EnableUser(ctx context.Context, token, id string) error {
	if err := svc.changeStatus(ctx, token, id, EnabledStatusKey); err != nil {
		return err
	}
	return nil
}

func (svc usersService) DisableUser(ctx context.Context, token, id string) error {
	if err := svc.changeStatus(ctx, token, id, DisabledStatusKey); err != nil {
		return err
	}
	return nil
}

func (svc usersService) changeStatus(ctx context.Context, token, id, status string) error {
	if _, err := svc.identify(ctx, token); err != nil {
		return err
	}

	dbUser, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(errors.ErrNotFound, err)
	}
	if dbUser.Status == status {
		if status == DisabledStatusKey {
			return ErrAlreadyDisabledUser
		}
		return ErrAlreadyEnabledUser
	}

	return svc.users.ChangeStatus(ctx, id, status)
}

// Auth helpers
func (svc usersService) issue(ctx context.Context, id, email string, keyType uint32) (string, error) {
	key, err := svc.auth.Issue(ctx, &mainflux.IssueReq{Id: id, Email: email, Type: keyType})
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}
	return key.GetValue(), nil
}

type userIdentity struct {
	id    string
	email string
}

func (svc usersService) identify(ctx context.Context, token string) (userIdentity, error) {
	identity, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return userIdentity{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return userIdentity{identity.Id, identity.Email}, nil
}

func (svc usersService) authorize(ctx context.Context, subject, object, relation string) error {
	req := &mainflux.AuthorizeReq{
		Sub: subject,
		Obj: object,
		Act: relation,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}

func (svc usersService) claimOwnership(ctx context.Context, subject, object, relation string) error {
	req := &mainflux.AddPolicyReq{
		Sub: subject,
		Obj: object,
		Act: relation,
	}
	res, err := svc.auth.AddPolicy(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
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
