// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
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

func (es *eventStore) Register(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	user, err := es.svc.Register(ctx, session, user, selfRegister)
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

func (es *eventStore) Update(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.Update(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "", user)
}

func (es *eventStore) UpdateRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateRole(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "role", user)
}

func (es *eventStore) UpdateTags(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateTags(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "tags", user)
}

func (es *eventStore) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	user, err := es.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "secret", user)
}

func (es *eventStore) UpdateUsername(ctx context.Context, session authn.Session, id, username string) (users.User, error) {
	user, err := es.svc.UpdateUsername(ctx, session, id, username)
	if err != nil {
		return user, err
	}

	event := updateUsernameEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateProfilePicture(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateProfilePicture(ctx, session, user)
	if err != nil {
		return user, err
	}

	event := updateProfilePictureEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateEmail(ctx context.Context, session authn.Session, id, email string) (users.User, error) {
	user, err := es.svc.UpdateEmail(ctx, session, id, email)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "email", user)
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

func (es *eventStore) View(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.View(ctx, session, id)
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

func (es *eventStore) Enable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.Enable(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.delete(ctx, user)
}

func (es *eventStore) Disable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.Disable(ctx, session, id)
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

func (es *eventStore) IssueToken(ctx context.Context, username, secret string) (*grpcTokenV1.Token, error) {
	token, err := es.svc.IssueToken(ctx, username, secret)
	if err != nil {
		return token, err
	}

	event := issueTokenEvent{
		username: username,
	}

	if err := es.Publish(ctx, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) RefreshToken(ctx context.Context, session authn.Session, refreshToken string) (*grpcTokenV1.Token, error) {
	token, err := es.svc.RefreshToken(ctx, session, refreshToken)
	if err != nil {
		return token, err
	}

	event := refreshTokenEvent{}

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

func (es *eventStore) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.Delete(ctx, session, id); err != nil {
		return err
	}

	event := deleteUserEvent{
		id: id,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	if err := es.svc.OAuthAddUserPolicy(ctx, user); err != nil {
		return err
	}

	event := addUserPolicyEvent{
		id:   user.ID,
		role: user.Role.String(),
	}

	return es.Publish(ctx, event)
}
