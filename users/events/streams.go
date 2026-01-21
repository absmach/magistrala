// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/users"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	supermqPrefix            = "supermq."
	createStream             = supermqPrefix + userCreate
	sendVerificationStream   = supermqPrefix + userSendVerification
	verifyEmailStream        = supermqPrefix + userVerifyEmail
	updateStream             = supermqPrefix + userUpdate
	updateRoleStream         = supermqPrefix + userUpdateRole
	updateTagsStream         = supermqPrefix + userUpdateTags
	updateSecretStream       = supermqPrefix + userUpdateSecret
	updateUsernameStream     = supermqPrefix + userUpdateUsername
	updatePictureStream      = supermqPrefix + userUpdateProfilePicture
	UpdateEmailStream        = supermqPrefix + userUpdateEmail
	enableStream             = supermqPrefix + userEnable
	disableStream            = supermqPrefix + userDisable
	viewStream               = supermqPrefix + userView
	viewProfileStream        = supermqPrefix + profileView
	listStream               = supermqPrefix + userList
	searchStream             = supermqPrefix + userSearch
	identifyStream           = supermqPrefix + userIdentify
	issueTokenStream         = supermqPrefix + issueToken
	refreshTokenStream       = supermqPrefix + refreshToken
	revokeRefreshTokenStream = supermqPrefix + revokeRefreshToken
	resetSecretStream        = supermqPrefix + resetSecret
	sendPasswordResetStream  = supermqPrefix + sendPasswordReset
	oauthStream              = supermqPrefix + oauthCallback
	addPolicyStream          = supermqPrefix + addClientPolicy
	deleteStream             = supermqPrefix + deleteUser
)

var _ users.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc users.Service
}

// NewEventStoreMiddleware returns wrapper around users service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc users.Service, url string) (users.Service, error) {
	publisher, err := store.NewPublisher(ctx, url)
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
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, createStream, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) SendVerification(ctx context.Context, session authn.Session) error {
	err := es.svc.SendVerification(ctx, session)
	if err != nil {
		return err
	}

	event := sendVerificationEvent{
		session,
		middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, sendVerificationStream, event)
}

func (es *eventStore) VerifyEmail(ctx context.Context, verificationToken string) (users.User, error) {
	user, err := es.svc.VerifyEmail(ctx, verificationToken)
	if err != nil {
		return user, err
	}

	event := verifyEmailEvent{
		email:      user.Email,
		userID:     user.ID,
		verifiedAt: user.VerifiedAt,
		requestID:  middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, verifyEmailStream, event); err != nil {
		return user, err
	}
	return user, nil
}

func (es *eventStore) Update(ctx context.Context, session authn.Session, id string, usr users.UserReq) (users.User, error) {
	user, err := es.svc.Update(ctx, session, id, usr)
	if err != nil {
		return user, err
	}

	return es.update(ctx, session, userUpdate, updateStream, user)
}

func (es *eventStore) UpdateRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	user, err := es.svc.UpdateRole(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, session, userUpdateRole, updateRoleStream, user)
}

func (es *eventStore) UpdateTags(ctx context.Context, session authn.Session, id string, usr users.UserReq) (users.User, error) {
	user, err := es.svc.UpdateTags(ctx, session, id, usr)
	if err != nil {
		return user, err
	}

	return es.update(ctx, session, userUpdateTags, updateTagsStream, user)
}

func (es *eventStore) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	user, err := es.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
	if err != nil {
		return user, err
	}

	return es.update(ctx, session, userUpdateSecret, updateSecretStream, user)
}

func (es *eventStore) UpdateUsername(ctx context.Context, session authn.Session, id, username string) (users.User, error) {
	user, err := es.svc.UpdateUsername(ctx, session, id, username)
	if err != nil {
		return user, err
	}

	event := updateUsernameEvent{
		user,
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, updateUsernameStream, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateProfilePicture(ctx context.Context, session authn.Session, id string, usr users.UserReq) (users.User, error) {
	user, err := es.svc.UpdateProfilePicture(ctx, session, id, usr)
	if err != nil {
		return user, err
	}

	event := updateProfilePictureEvent{
		user,
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, updatePictureStream, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateEmail(ctx context.Context, session authn.Session, id, email string) (users.User, error) {
	user, err := es.svc.UpdateEmail(ctx, session, id, email)
	if err != nil {
		return user, err
	}

	return es.update(ctx, session, userUpdateEmail, UpdateEmailStream, user)
}

func (es *eventStore) update(ctx context.Context, session authn.Session, operation, stream string, user users.User) (users.User, error) {
	event := updateUserEvent{
		user, operation, session, middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, stream, event); err != nil {
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
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, viewStream, event); err != nil {
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
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, viewProfileStream, event); err != nil {
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
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, listStream, event); err != nil {
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
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, searchStream, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) Enable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.Enable(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.changeStatus(ctx, session, userEnable, enableStream, user)
}

func (es *eventStore) Disable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	user, err := es.svc.Disable(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.changeStatus(ctx, session, userDisable, disableStream, user)
}

func (es *eventStore) changeStatus(ctx context.Context, session authn.Session, operation, stream string, user users.User) (users.User, error) {
	event := changeUserStatusEvent{
		id:        user.ID,
		operation: operation,
		updatedAt: user.UpdatedAt,
		updatedBy: user.UpdatedBy,
		status:    user.Status.String(),
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, stream, event); err != nil {
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
		userID:    userID,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, identifyStream, event); err != nil {
		return userID, err
	}

	return userID, nil
}

func (es *eventStore) SendPasswordReset(ctx context.Context, email string) error {
	err := es.svc.SendPasswordReset(ctx, email)
	if err != nil {
		return err
	}

	event := sendPasswordResetEvent{
		email:     email,
		requestID: middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, sendPasswordResetStream, event)
}

func (es *eventStore) IssueToken(ctx context.Context, username, secret, description string) (*grpcTokenV1.Token, error) {
	token, err := es.svc.IssueToken(ctx, username, secret, description)
	if err != nil {
		return token, err
	}

	event := issueTokenEvent{
		username:  username,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, issueTokenStream, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) RefreshToken(ctx context.Context, session authn.Session, refreshToken string) (*grpcTokenV1.Token, error) {
	token, err := es.svc.RefreshToken(ctx, session, refreshToken)
	if err != nil {
		return token, err
	}

	event := refreshTokenEvent{
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, refreshTokenStream, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) RevokeRefreshToken(ctx context.Context, session authn.Session, tokenID string) error {
	err := es.svc.RevokeRefreshToken(ctx, session, tokenID)
	if err != nil {
		return err
	}

	event := revokeRefreshTokenEvent{
		requestID: middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, revokeRefreshTokenStream, event)
}

func (es *eventStore) ListActiveRefreshTokens(ctx context.Context, session authn.Session) (*grpcTokenV1.ListUserRefreshTokensRes, error) {
	return es.svc.ListActiveRefreshTokens(ctx, session)
}

func (es *eventStore) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	if err := es.svc.ResetSecret(ctx, session, secret); err != nil {
		return err
	}

	event := resetSecretEvent{
		requestID: middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, resetSecretStream, event)
}

func (es *eventStore) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	token, err := es.svc.OAuthCallback(ctx, user)
	if err != nil {
		return token, err
	}

	event := oauthCallbackEvent{
		userID:    user.ID,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, oauthStream, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.Delete(ctx, session, id); err != nil {
		return err
	}

	event := deleteUserEvent{
		id:        id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, deleteStream, event)
}

func (es *eventStore) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	if err := es.svc.OAuthAddUserPolicy(ctx, user); err != nil {
		return err
	}

	event := addUserPolicyEvent{
		id:        user.ID,
		role:      user.Role.String(),
		requestID: middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, addPolicyStream, event)
}
