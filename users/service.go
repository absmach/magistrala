// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/users/postgres"
)

const (
	ownerRelation = "owner"

	userKind   = "users"
	tokenKind  = "token"
	thingsKind = "things"
	groupsKind = "groups"

	userType  = "user"
	groupType = "group"
	thingType = "thing"
)

var (
	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")
)

type service struct {
	clients    postgres.Repository
	idProvider magistrala.IDProvider
	auth       magistrala.AuthServiceClient
	hasher     Hasher
	email      Emailer
	passRegex  *regexp.Regexp
}

// NewService returns a new Users service implementation.
func NewService(crepo postgres.Repository, auth magistrala.AuthServiceClient, emailer Emailer, hasher Hasher, idp magistrala.IDProvider, pr *regexp.Regexp) Service {
	return service{
		clients:    crepo,
		auth:       auth,
		hasher:     hasher,
		email:      emailer,
		idProvider: idp,
		passRegex:  pr,
	}
}

func (svc service) RegisterClient(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	// We don't check the error currently since we can register client with empty token
	ownerID, _ := svc.Identify(ctx, token)

	clientID, err := svc.idProvider.ID()
	if err != nil {
		return mgclients.Client{}, err
	}
	if cli.Owner == "" && ownerID != "" {
		cli.Owner = ownerID
	}
	if cli.Credentials.Secret == "" {
		return mgclients.Client{}, apiutil.ErrMissingSecret
	}
	hash, err := svc.hasher.Hash(cli.Credentials.Secret)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	cli.Credentials.Secret = hash
	if cli.Status != mgclients.DisabledStatus && cli.Status != mgclients.EnabledStatus {
		return mgclients.Client{}, apiutil.ErrInvalidStatus
	}
	if cli.Role != mgclients.UserRole && cli.Role != mgclients.AdminRole {
		return mgclients.Client{}, apiutil.ErrInvalidRole
	}
	cli.ID = clientID
	cli.CreatedAt = time.Now()

	client, err := svc.clients.Save(ctx, cli)
	if err != nil {
		return mgclients.Client{}, err
	}

	if err := svc.addOwnerPolicy(ctx, ownerID, client.ID); err != nil {
		return mgclients.Client{}, err
	}

	return client, nil
}

func (svc service) IssueToken(ctx context.Context, identity, secret string) (*magistrala.Token, error) {
	dbUser, err := svc.clients.RetrieveByIdentity(ctx, identity)
	if err != nil {
		return &magistrala.Token{}, err
	}
	if err := svc.hasher.Compare(secret, dbUser.Credentials.Secret); err != nil {
		return &magistrala.Token{}, errors.Wrap(errors.ErrLogin, err)
	}

	return svc.auth.Issue(ctx, &magistrala.IssueReq{Id: dbUser.ID, Type: 0})
}

func (svc service) RefreshToken(ctx context.Context, refreshToken string) (*magistrala.Token, error) {
	return svc.auth.Refresh(ctx, &magistrala.RefreshReq{Value: refreshToken})
}

func (svc service) ViewClient(ctx context.Context, token string, id string) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != id {
		if err := svc.isOwner(ctx, id, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}

	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, err
	}
	client.Credentials.Secret = ""

	return client, nil
}

func (svc service) ViewProfile(ctx context.Context, token string) (mgclients.Client, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, err
	}
	client.Credentials.Secret = ""

	return client, nil
}

func (svc service) ListClients(ctx context.Context, token string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}
	pm.Owner = id
	clients, err := svc.clients.RetrieveAll(ctx, pm)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}
	return clients, nil
}

func (svc service) UpdateClient(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.isOwner(ctx, cli.ID, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	return svc.clients.Update(ctx, client)
}

func (svc service) UpdateClientTags(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.isOwner(ctx, cli.ID, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	return svc.clients.UpdateTags(ctx, client)
}

func (svc service) UpdateClientIdentity(ctx context.Context, token, clientID, identity string) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != clientID {
		if err := svc.isOwner(ctx, clientID, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}

	cli := mgclients.Client{
		ID: clientID,
		Credentials: mgclients.Credentials{
			Identity: identity,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}
	return svc.clients.UpdateIdentity(ctx, cli)
}

func (svc service) GenerateResetToken(ctx context.Context, email, host string) error {
	client, err := svc.clients.RetrieveByIdentity(ctx, email)
	if err != nil || client.Credentials.Identity == "" {
		return errors.ErrNotFound
	}
	issueReq := &magistrala.IssueReq{
		Id:   client.ID,
		Type: uint32(auth.RecoveryKey),
	}
	token, err := svc.auth.Issue(ctx, issueReq)
	if err != nil {
		return errors.Wrap(ErrRecoveryToken, err)
	}

	return svc.SendPasswordReset(ctx, host, email, client.Name, token.AccessToken)
}

func (svc service) ResetSecret(ctx context.Context, resetToken, secret string) error {
	id, err := svc.Identify(ctx, resetToken)
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}
	c, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return err
	}
	if c.Credentials.Identity == "" {
		return errors.ErrNotFound
	}
	if !svc.passRegex.MatchString(secret) {
		return ErrPasswordFormat
	}
	secret, err = svc.hasher.Hash(secret)
	if err != nil {
		return err
	}
	c = mgclients.Client{
		Credentials: mgclients.Credentials{
			Identity: c.Credentials.Identity,
			Secret:   secret,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: id,
	}
	if _, err := svc.clients.UpdateSecret(ctx, c); err != nil {
		return err
	}
	return nil
}

func (svc service) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (mgclients.Client, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}
	if !svc.passRegex.MatchString(newSecret) {
		return mgclients.Client{}, ErrPasswordFormat
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, err
	}
	if _, err := svc.IssueToken(ctx, dbClient.Credentials.Identity, oldSecret); err != nil {
		return mgclients.Client{}, err
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return mgclients.Client{}, err
	}
	dbClient.Credentials.Secret = newSecret
	dbClient.UpdatedAt = time.Now()
	dbClient.UpdatedBy = id

	return svc.clients.UpdateSecret(ctx, dbClient)
}

func (svc service) SendPasswordReset(_ context.Context, host, email, user, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, user, token)
}

func (svc service) UpdateClientOwner(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.isOwner(ctx, cli.ID, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}
	client := mgclients.Client{
		ID:        cli.ID,
		Owner:     cli.Owner,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	if err := svc.updateOwnerPolicy(ctx, tokenUserID, cli.Owner, cli.ID); err != nil {
		return mgclients.Client{}, err
	}
	return svc.clients.UpdateOwner(ctx, client)
}

func (svc service) EnableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    mgclients.EnabledStatus,
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) DisableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    mgclients.DisabledStatus,
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	return client, nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}
	if tokenUserID != client.ID {
		if err := svc.isOwner(ctx, client.ID, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, err
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, mgclients.ErrStatusAlreadyAssigned
	}
	client.UpdatedBy = tokenUserID
	return svc.clients.ChangeStatus(ctx, client)
}

func (svc service) ListMembers(ctx context.Context, token, objectKind string, objectID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	var objectType string
	var authzPerm string
	switch objectKind {
	case thingsKind:
		objectType = thingType
		authzPerm = pm.Permission
	case groupsKind:
		fallthrough
	default:
		objectType = groupType
		authzPerm = auth.SwitchToPermission(pm.Permission)
	}

	if _, err := svc.authorize(ctx, userType, tokenKind, token, authzPerm, objectType, objectID); err != nil {
		return mgclients.MembersPage{}, err
	}
	uids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
		SubjectType: userType,
		Permission:  pm.Permission,
		Object:      objectID,
		ObjectType:  objectType,
	})
	if err != nil {
		return mgclients.MembersPage{}, err
	}
	if len(uids.Policies) == 0 {
		return mgclients.MembersPage{
			Page: mgclients.Page{Total: 0, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}

	pm.IDs = uids.Policies

	cp, err := svc.clients.RetrieveAll(ctx, pm)
	if err != nil {
		return mgclients.MembersPage{}, err
	}

	return mgclients.MembersPage{
		Page:    cp.Page,
		Members: cp.Clients,
	}, nil
}

func (svc *service) isOwner(ctx context.Context, clientID, ownerID string) error {
	_, err := svc.authorize(ctx, userType, userKind, ownerID, ownerRelation, userType, clientID)
	return err
}

func (svc *service) authorize(ctx context.Context, subjType, subjKind, subj, perm, objType, obj string) (string, error) {
	req := &magistrala.AuthorizeReq{
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	if !res.GetAuthorized() {
		return "", errors.ErrAuthorization
	}
	return res.GetId(), nil
}

func (svc service) Identify(ctx context.Context, token string) (string, error) {
	user, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return "", err
	}
	return user.GetId(), nil
}

func (svc service) updateOwnerPolicy(ctx context.Context, previousOwnerID, ownerID, userID string) error {
	if previousOwnerID != "" {
		if _, err := svc.auth.DeletePolicy(ctx, &magistrala.DeletePolicyReq{
			SubjectType: userType,
			Subject:     previousOwnerID,
			Relation:    ownerRelation,
			ObjectType:  userType,
			Object:      userID,
		}); err != nil {
			return err
		}
	}
	return svc.addOwnerPolicy(ctx, ownerID, userID)
}

func (svc service) addOwnerPolicy(ctx context.Context, ownerID, userID string) error {
	if ownerID != "" {
		if _, err := svc.auth.AddPolicy(ctx, &magistrala.AddPolicyReq{
			SubjectType: userType,
			Subject:     ownerID,
			Relation:    ownerRelation,
			ObjectType:  userType,
			Object:      userID,
		}); err != nil {
			return err
		}
	}
	return nil
}
