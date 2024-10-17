// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/users"
)

const streamID = "magistrala.users"

var _ users.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc users.Service
}

// NewEventStoreMiddleware returns wrapper around users service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc users.Service, url string) (users.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es *eventStore) RegisterUser(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	user, err := es.svc.RegisterUser(ctx, session, user, selfRegister)
	if err != nil {
		return user, err
	}

	event := createUserEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateUser(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateUser(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "", user)
}

func (es *eventStore) UpdateUserRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateUserRole(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "role", user)
}

func (es *eventStore) UpdateUserTags(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateUserTags(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "tags", user)
}

func (es *eventStore) UpdateUserSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	user, err := es.svc.UpdateUserSecret(ctx, session, oldSecret, newSecret)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "secret", user)
}

func (es *eventStore) UpdateUserNames(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateUserNames(ctx, session, user)
	if err != nil {
		return user, err
	}

	event := updateUserNamesEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateProfilePicture(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateUser(ctx, session, user)
	if err != nil {
		return user, err
	}

	event := updateProfilePictureEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return es.update(ctx, "profile_picture", user)
}

func (es *eventStore) UpdateUserIdentity(ctx context.Context, session authn.Session, id, identity string) (users.User, error) {
	user, err := es.svc.UpdateUserIdentity(ctx, session, id, identity)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "identity", user)
}

func (es *eventStore) update(ctx context.Context, operation string, user users.User) (users.User, error) {
	event := updateUserEvent{
		user, operation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) ViewUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.ViewUser(ctx, session, id)
	if err != nil {
		return user, err
	}

	event := viewUserEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	user, err := es.svc.ViewProfile(ctx, session)
	if err != nil {
		return user, err
	}

	event := viewProfileEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) ViewUserByUserName(ctx context.Context, session authn.Session, userName string) (users.User, error) {
	user, err := es.svc.ViewUserByUserName(ctx, session, userName)
	if err != nil {
		return user, err
	}

	event := viewUserByUserNameEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	cp, err := es.svc.ListUsers(ctx, session, pm)
	if err != nil {
		return cp, err
	}
	event := listUserEvent{
		pm,
	}

	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	cp, err := es.svc.SearchUsers(ctx, pm)
	if err != nil {
		return cp, err
	}
	event := searchUserEvent{
		pm,
	}

	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm users.Page) (users.MembersPage, error) {
	mp, err := es.svc.ListMembers(ctx, session, objectKind, objectID, pm)
	if err != nil {
		return mp, err
	}
	event := listUserByGroupEvent{
		pm, objectKind, objectID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

func (es *eventStore) EnableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.EnableUser(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.delete(ctx, user)
}

func (es *eventStore) DisableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.DisableUser(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.delete(ctx, user)
}

func (es *eventStore) delete(ctx context.Context, user users.User) (users.User, error) {
	event := removeUserEvent{
		id:        user.ID,
		updatedAt: user.UpdatedAt,
		updatedBy: user.UpdatedBy,
		status:    user.Status.String(),
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) Identify(ctx context.Context, session authn.Session) (string, error) {
	userID, err := es.svc.Identify(ctx, session)
	if err != nil {
		return userID, err
	}

	event := identifyUserEvent{
		userID: userID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return userID, err
	}

	return userID, nil
}

func (es *eventStore) GenerateResetToken(ctx context.Context, email, host string) error {
	err := es.svc.GenerateResetToken(ctx, email, host)
	if err != nil {
		return err
	}

	event := generateResetTokenEvent{
		email: email,
		host:  host,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	token, err := es.svc.IssueToken(ctx, identity, secret, domainID)
	if err != nil {
		return token, err
	}

	event := issueTokenEvent{
		identity: identity,
		domainID: domainID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) RefreshToken(ctx context.Context, session authn.Session, refreshToken, domainID string) (*magistrala.Token, error) {
	token, err := es.svc.RefreshToken(ctx, session, refreshToken, domainID)
	if err != nil {
		return token, err
	}

	event := refreshTokenEvent{domainID: domainID}

	if err := es.Publish(ctx, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	if err := es.svc.ResetSecret(ctx, session, secret); err != nil {
		return err
	}

	event := resetSecretEvent{}

	return es.Publish(ctx, event)
}

func (es *eventStore) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	if err := es.svc.SendPasswordReset(ctx, host, email, user, token); err != nil {
		return err
	}

	event := sendPasswordResetEvent{
		host:  host,
		email: email,
		user:  user,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	token, err := es.svc.OAuthCallback(ctx, user)
	if err != nil {
		return token, err
	}

	event := oauthCallbackEvent{
		userID: user.ID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) DeleteUser(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.DeleteUser(ctx, session, id); err != nil {
		return err
	}

	event := deleteUserEvent{
		id: id,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) OAuthAddUserPolicy(ctx context.Context, client users.User) error {
	if err := es.svc.OAuthAddUserPolicy(ctx, client); err != nil {
		return err
	}

	event := addUserPolicyEvent{
		id:   client.ID,
		role: client.Role.String(),
	}

	return es.Publish(ctx, event)
}
