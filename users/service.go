// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/postgres"
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
	idProvider mainflux.IDProvider
	auth       mainflux.AuthServiceClient
	hasher     Hasher
	email      Emailer
	passRegex  *regexp.Regexp
}

// NewService returns a new Users service implementation.
func NewService(crepo postgres.Repository, auth mainflux.AuthServiceClient, emailer Emailer, hasher Hasher, idp mainflux.IDProvider, pr *regexp.Regexp) Service {
	return service{
		clients:    crepo,
		auth:       auth,
		hasher:     hasher,
		email:      emailer,
		idProvider: idp,
		passRegex:  pr,
	}
}

func (svc service) RegisterClient(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	// We don't check the error currently since we can register client with empty token
	ownerID, _ := svc.Identify(ctx, token)

	clientID, err := svc.idProvider.ID()
	if err != nil {
		return mfclients.Client{}, err
	}
	if cli.Owner == "" && ownerID != "" {
		cli.Owner = ownerID
	}
	if cli.Credentials.Secret == "" {
		return mfclients.Client{}, apiutil.ErrMissingSecret
	}
	hash, err := svc.hasher.Hash(cli.Credentials.Secret)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	cli.Credentials.Secret = hash
	if cli.Status != mfclients.DisabledStatus && cli.Status != mfclients.EnabledStatus {
		return mfclients.Client{}, apiutil.ErrInvalidStatus
	}
	if cli.Role != mfclients.UserRole && cli.Role != mfclients.AdminRole {
		return mfclients.Client{}, apiutil.ErrInvalidRole
	}
	cli.ID = clientID
	cli.CreatedAt = time.Now()

	client, err := svc.clients.Save(ctx, cli)
	if err != nil {
		return mfclients.Client{}, err
	}

	if err := svc.addOwnerPolicy(ctx, ownerID, client.ID); err != nil {
		return mfclients.Client{}, err
	}

	return client, nil
}

func (svc service) IssueToken(ctx context.Context, identity, secret string) (*mainflux.Token, error) {
	dbUser, err := svc.clients.RetrieveByIdentity(ctx, identity)
	if err != nil {
		return &mainflux.Token{}, err
	}
	if err := svc.hasher.Compare(secret, dbUser.Credentials.Secret); err != nil {
		return &mainflux.Token{}, errors.Wrap(errors.ErrLogin, err)
	}

	return svc.auth.Issue(ctx, &mainflux.IssueReq{Id: dbUser.ID, Type: 0})
}

func (svc service) RefreshToken(ctx context.Context, refreshToken string) (*mainflux.Token, error) {
	return svc.auth.Refresh(ctx, &mainflux.RefreshReq{Value: refreshToken})
}

func (svc service) ViewClient(ctx context.Context, token string, id string) (mfclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}

	if tokenUserID != id {
		if err := svc.isOwner(ctx, id, tokenUserID); err != nil {
			return mfclients.Client{}, err
		}
	}

	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mfclients.Client{}, err
	}
	client.Credentials.Secret = ""

	return client, nil
}

func (svc service) ViewProfile(ctx context.Context, token string) (mfclients.Client, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mfclients.Client{}, err
	}
	client.Credentials.Secret = ""

	return client, nil
}

func (svc service) ListClients(ctx context.Context, token string, pm mfclients.Page) (mfclients.ClientsPage, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.ClientsPage{}, err
	}
	pm.Owner = id
	clients, err := svc.clients.RetrieveAll(ctx, pm)
	if err != nil {
		return mfclients.ClientsPage{}, err
	}
	return clients, nil
}

func (svc service) UpdateClient(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.isOwner(ctx, cli.ID, tokenUserID); err != nil {
			return mfclients.Client{}, err
		}
	}

	client := mfclients.Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	return svc.clients.Update(ctx, client)
}

func (svc service) UpdateClientTags(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.isOwner(ctx, cli.ID, tokenUserID); err != nil {
			return mfclients.Client{}, err
		}
	}

	client := mfclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	return svc.clients.UpdateTags(ctx, client)
}

func (svc service) UpdateClientIdentity(ctx context.Context, token, clientID, identity string) (mfclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}

	if tokenUserID != clientID {
		if err := svc.isOwner(ctx, clientID, tokenUserID); err != nil {
			return mfclients.Client{}, err
		}
	}

	cli := mfclients.Client{
		ID: clientID,
		Credentials: mfclients.Credentials{
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
	issueReq := &mainflux.IssueReq{
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
	c = mfclients.Client{
		Credentials: mfclients.Credentials{
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

func (svc service) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (mfclients.Client, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}
	if !svc.passRegex.MatchString(newSecret) {
		return mfclients.Client{}, ErrPasswordFormat
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mfclients.Client{}, err
	}
	if _, err := svc.IssueToken(ctx, dbClient.Credentials.Identity, oldSecret); err != nil {
		return mfclients.Client{}, err
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return mfclients.Client{}, err
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

func (svc service) UpdateClientOwner(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.isOwner(ctx, cli.ID, tokenUserID); err != nil {
			return mfclients.Client{}, err
		}
	}
	client := mfclients.Client{
		ID:        cli.ID,
		Owner:     cli.Owner,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	if err := svc.updateOwnerPolicy(ctx, tokenUserID, cli.Owner, cli.ID); err != nil {
		return mfclients.Client{}, err
	}
	return svc.clients.UpdateOwner(ctx, client)
}

func (svc service) EnableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	client := mfclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    mfclients.EnabledStatus,
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(mfclients.ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) DisableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	client := mfclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    mfclients.DisabledStatus,
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(mfclients.ErrDisableClient, err)
	}

	return client, nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mfclients.Client) (mfclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mfclients.Client{}, err
	}
	if tokenUserID != client.ID {
		if err := svc.isOwner(ctx, client.ID, tokenUserID); err != nil {
			return mfclients.Client{}, err
		}
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mfclients.Client{}, err
	}
	if dbClient.Status == client.Status {
		return mfclients.Client{}, mfclients.ErrStatusAlreadyAssigned
	}
	client.UpdatedBy = tokenUserID
	return svc.clients.ChangeStatus(ctx, client)
}

func (svc service) ListMembers(ctx context.Context, token, objectKind string, objectID string, pm mfclients.Page) (mfclients.MembersPage, error) {
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
		return mfclients.MembersPage{}, err
	}
	uids, err := svc.auth.ListAllSubjects(ctx, &mainflux.ListSubjectsReq{
		SubjectType: userType,
		Permission:  pm.Permission,
		Object:      objectID,
		ObjectType:  objectType,
	})
	if err != nil {
		return mfclients.MembersPage{}, err
	}
	if len(uids.Policies) == 0 {
		return mfclients.MembersPage{
			Page: mfclients.Page{Total: 0, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}

	pm.IDs = uids.Policies

	cp, err := svc.clients.RetrieveAll(ctx, pm)
	if err != nil {
		return mfclients.MembersPage{}, err
	}

	return mfclients.MembersPage{
		Page:    cp.Page,
		Members: cp.Clients,
	}, nil
}

func (svc *service) isOwner(ctx context.Context, clientID, ownerID string) error {
	_, err := svc.authorize(ctx, userType, userKind, ownerID, ownerRelation, userType, clientID)
	return err
}

func (svc *service) authorize(ctx context.Context, subjType, subjKind, subj, perm, objType, obj string) (string, error) {
	req := &mainflux.AuthorizeReq{
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
	user, err := svc.auth.Identify(ctx, &mainflux.IdentityReq{Token: token})
	if err != nil {
		return "", err
	}
	return user.GetId(), nil
}

func (svc service) updateOwnerPolicy(ctx context.Context, previousOwnerID, ownerID, userID string) error {
	if previousOwnerID != "" {
		if _, err := svc.auth.DeletePolicy(ctx, &mainflux.DeletePolicyReq{
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
		if _, err := svc.auth.AddPolicy(ctx, &mainflux.AddPolicyReq{
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
