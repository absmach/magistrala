// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
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

func (es *eventStore) RegisterClient(ctx context.Context, session auth.Session, user mgclients.Client, selfRegister bool) (mgclients.Client, error) {
	user, err := es.svc.RegisterClient(ctx, session, user, selfRegister)
	if err != nil {
		return user, err
	}

	event := createClientEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) UpdateClient(ctx context.Context, session auth.Session, user mgclients.Client) (mgclients.Client, error) {
	user, err := es.svc.UpdateClient(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "", user)
}

func (es *eventStore) UpdateClientRole(ctx context.Context, session auth.Session, user mgclients.Client) (mgclients.Client, error) {
	user, err := es.svc.UpdateClientRole(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "role", user)
}

func (es *eventStore) UpdateClientTags(ctx context.Context, session auth.Session, user mgclients.Client) (mgclients.Client, error) {
	user, err := es.svc.UpdateClientTags(ctx, session, user)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "tags", user)
}

func (es *eventStore) UpdateClientSecret(ctx context.Context, session auth.Session, oldSecret, newSecret string) (mgclients.Client, error) {
	user, err := es.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "secret", user)
}

func (es *eventStore) UpdateClientIdentity(ctx context.Context, session auth.Session, id, identity string) (mgclients.Client, error) {
	user, err := es.svc.UpdateClientIdentity(ctx, session, id, identity)
	if err != nil {
		return user, err
	}

	return es.update(ctx, "identity", user)
}

func (es *eventStore) update(ctx context.Context, operation string, user mgclients.Client) (mgclients.Client, error) {
	event := updateClientEvent{
		user, operation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) ViewClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	user, err := es.svc.ViewClient(ctx, session, id)
	if err != nil {
		return user, err
	}

	event := viewClientEvent{
		user,
	}

	if err := es.Publish(ctx, event); err != nil {
		return user, err
	}

	return user, nil
}

func (es *eventStore) ViewProfile(ctx context.Context, session auth.Session) (mgclients.Client, error) {
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

func (es *eventStore) ListClients(ctx context.Context, session auth.Session, pm mgclients.Page) (mgclients.ClientsPage, error) {
	cp, err := es.svc.ListClients(ctx, session, pm)
	if err != nil {
		return cp, err
	}
	event := listClientEvent{
		pm,
	}

	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) SearchUsers(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	cp, err := es.svc.SearchUsers(ctx, pm)
	if err != nil {
		return cp, err
	}
	event := searchClientEvent{
		pm,
	}

	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) ListMembers(ctx context.Context, session auth.Session, objectKind, objectID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	mp, err := es.svc.ListMembers(ctx, session, objectKind, objectID, pm)
	if err != nil {
		return mp, err
	}
	event := listClientByGroupEvent{
		pm, objectKind, objectID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

func (es *eventStore) EnableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	user, err := es.svc.EnableClient(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.delete(ctx, user)
}

func (es *eventStore) DisableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	user, err := es.svc.DisableClient(ctx, session, id)
	if err != nil {
		return user, err
	}

	return es.delete(ctx, user)
}

func (es *eventStore) delete(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	event := removeClientEvent{
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

func (es *eventStore) Identify(ctx context.Context, session auth.Session) (string, error) {
	userID, err := es.svc.Identify(ctx, session)
	if err != nil {
		return userID, err
	}

	event := identifyClientEvent{
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

func (es *eventStore) RefreshToken(ctx context.Context, session auth.Session, refreshToken, domainID string) (*magistrala.Token, error) {
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

func (es *eventStore) ResetSecret(ctx context.Context, session auth.Session, secret string) error {
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

func (es *eventStore) OAuthCallback(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	token, err := es.svc.OAuthCallback(ctx, client)
	if err != nil {
		return token, err
	}

	event := oauthCallbackEvent{
		clientID: client.ID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return token, err
	}

	return token, nil
}

func (es *eventStore) DeleteClient(ctx context.Context, session auth.Session, id string) error {
	if err := es.svc.DeleteClient(ctx, session, id); err != nil {
		return err
	}

	event := deleteClientEvent{
		id: id,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) OAuthAddClientPolicy(ctx context.Context, client mgclients.Client) error {
	if err := es.svc.OAuthAddClientPolicy(ctx, client); err != nil {
		return err
	}

	event := addClientPolicyEvent{
		id:   client.ID,
		role: client.Role.String(),
	}

	return es.Publish(ctx, event)
}
