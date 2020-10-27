// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/pkg/errors"
	uuidProvider "github.com/mainflux/mainflux/pkg/uuid"
)

var (
	groupRegexp = regexp.MustCompile("^[a-zA-Z0-9]+$")

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

	// ErrCreateGroup indicates error in creating group.
	ErrCreateGroup = errors.New("failed to create group")

	// ErrUpdateGroup indicates error in updating group.
	ErrUpdateGroup = errors.New("failed to update group")

	// ErrDeleteGroupMissing indicates in delete operation that group doesnt exist.
	ErrDeleteGroupMissing = errors.New("group is not existing, already deleted")

	// ErrAssignUserToGroup indicates an error in assigning user to a group.
	ErrAssignUserToGroup = errors.New("failed assigning user to a group")
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
	ListUsers(ctx context.Context, token string, offset, limit uint64, email string, m Metadata) (UserPage, error)

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

	// CreateGroup creates new user group.
	CreateGroup(ctx context.Context, token string, group Group) (Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, group Group) error

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ListGroups retrieves groups that are children to group identified by parenID
	// if parentID is empty all groups are listed.
	ListGroups(ctx context.Context, token, parentID string, offset, limit uint64, m Metadata) (GroupPage, error)

	// Members retrieves users that are assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, m Metadata) (UserPage, error)

	// ListMemberships retrieves groups that user identified with userID belongs to.
	ListMemberships(ctx context.Context, token, groupID string, offset, limit uint64, m Metadata) (GroupPage, error)

	// RemoveGroup removes the group identified with the provided ID.
	RemoveGroup(ctx context.Context, token, id string) error

	// Assign adds user with userID into the group identified by groupID.
	Assign(ctx context.Context, token, userID, groupID string) error

	// Unassign removes user with userID from group identified by groupID.
	Unassign(ctx context.Context, token, userID, groupID string) error
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
	Groups []Group
}

// UserPage contains a page of users.
type UserPage struct {
	PageMetadata
	Users []User
}

var _ Service = (*usersService)(nil)

type usersService struct {
	users  UserRepository
	groups GroupRepository
	hasher Hasher
	email  Emailer
	auth   mainflux.AuthNServiceClient
}

// New instantiates the users service implementation
func New(users UserRepository, groups GroupRepository, hasher Hasher, auth mainflux.AuthNServiceClient, m Emailer) Service {
	return &usersService{
		users:  users,
		groups: groups,
		hasher: hasher,
		auth:   auth,
		email:  m,
	}
}

func (svc usersService) Register(ctx context.Context, user User) (string, error) {
	if err := user.Validate(); err != nil {
		return "", err
	}
	hash, err := svc.hasher.Hash(user.Password)
	if err != nil {
		return "", errors.Wrap(ErrMalformedEntity, err)
	}
	user.Password = hash
	uid, err := uuidProvider.New().ID()
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
	return svc.issue(ctx, dbUser.ID, dbUser.Email, authn.UserKey)
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

	return svc.users.RetrieveAll(ctx, offset, limit, email, m)
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
	t, err := svc.issue(ctx, user.ID, user.Email, authn.RecoveryKey)
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

func (svc usersService) CreateGroup(ctx context.Context, token string, group Group) (Group, error) {
	if group.Name == "" || !groupRegexp.MatchString(group.Name) {
		return Group{}, ErrMalformedEntity
	}

	email, err := svc.identify(ctx, token)
	if err != nil {
		return Group{}, err
	}

	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil {
		return Group{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	uid, err := uuidProvider.New().ID()
	if err != nil {
		return Group{}, errors.Wrap(ErrCreateUser, err)
	}

	group.ID = uid
	group.OwnerID = user.ID

	return svc.groups.Save(ctx, group)
}

func (svc usersService) ListGroups(ctx context.Context, token string, parentID string, offset, limit uint64, m Metadata) (GroupPage, error) {
	_, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return GroupPage{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return svc.groups.RetrieveAllWithAncestors(ctx, parentID, offset, limit, m)
}

func (svc usersService) ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, m Metadata) (UserPage, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return UserPage{}, err
	}
	return svc.users.RetrieveMembers(ctx, groupID, offset, limit, m)
}

func (svc usersService) RemoveGroup(ctx context.Context, token, id string) error {
	if _, err := svc.identify(ctx, token); err != nil {
		return err
	}
	return svc.groups.Delete(ctx, id)
}

func (svc usersService) Unassign(ctx context.Context, token, userID, groupID string) error {
	if _, err := svc.identify(ctx, token); err != nil {
		return err
	}
	return svc.groups.Unassign(ctx, userID, groupID)
}

func (svc usersService) UpdateGroup(ctx context.Context, token string, group Group) error {
	if _, err := svc.identify(ctx, token); err != nil {
		return err
	}
	return svc.groups.Update(ctx, group)
}

func (svc usersService) ViewGroup(ctx context.Context, token, id string) (Group, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return Group{}, err
	}
	return svc.groups.RetrieveByID(ctx, id)
}

func (svc usersService) Assign(ctx context.Context, token, userID, groupID string) error {
	if _, err := svc.identify(ctx, token); err != nil {
		return err
	}
	return svc.groups.Assign(ctx, userID, groupID)
}

func (svc usersService) ListMemberships(ctx context.Context, token, userID string, offset, limit uint64, m Metadata) (GroupPage, error) {
	if _, err := svc.identify(ctx, token); err != nil {
		return GroupPage{}, err
	}
	return svc.groups.RetrieveMemberships(ctx, userID, offset, limit, m)
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
	email, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return email.GetEmail(), nil
}
