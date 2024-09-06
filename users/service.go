// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcclient "github.com/absmach/magistrala/auth/api/grpc"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/absmach/magistrala/users/postgres"
	"golang.org/x/sync/errgroup"
)

var (
	errIssueToken            = errors.New("failed to issue token")
	errFailedPermissionsList = errors.New("failed to list permissions")
	errRecoveryToken         = errors.New("failed to generate password recovery token")
	errLoginDisableUser      = errors.New("failed to login in disabled user")
)

type service struct {
	clients      postgres.Repository
	idProvider   magistrala.IDProvider
	auth         grpcclient.AuthServiceClient
	policy       policy.PolicyClient
	hasher       Hasher
	email        Emailer
	selfRegister bool
}

// NewService returns a new Users service implementation.
func NewService(crepo postgres.Repository, authClient grpcclient.AuthServiceClient, policyClient policy.PolicyClient, emailer Emailer, hasher Hasher, idp magistrala.IDProvider, selfRegister bool) Service {
	return service{
		clients:      crepo,
		auth:         authClient,
		policy:       policyClient,
		hasher:       hasher,
		email:        emailer,
		idProvider:   idp,
		selfRegister: selfRegister,
	}
}

func (svc service) RegisterClient(ctx context.Context, token string, cli mgclients.Client) (rc mgclients.Client, err error) {
	if !svc.selfRegister {
		userID, err := svc.Identify(ctx, token)
		if err != nil {
			return mgclients.Client{}, err
		}
		if err := svc.checkSuperAdmin(ctx, userID); err != nil {
			return mgclients.Client{}, err
		}
	}

	clientID, err := svc.idProvider.ID()
	if err != nil {
		return mgclients.Client{}, err
	}

	if cli.Credentials.Secret != "" {
		hash, err := svc.hasher.Hash(cli.Credentials.Secret)
		if err != nil {
			return mgclients.Client{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
		}
		cli.Credentials.Secret = hash
	}

	if cli.Status != mgclients.DisabledStatus && cli.Status != mgclients.EnabledStatus {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrInvalidStatus)
	}
	if cli.Role != mgclients.UserRole && cli.Role != mgclients.AdminRole {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrInvalidRole)
	}
	cli.ID = clientID
	cli.CreatedAt = time.Now()

	if err := svc.addClientPolicy(ctx, cli.ID, cli.Role); err != nil {
		return mgclients.Client{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addClientPolicyRollback(ctx, cli.ID, cli.Role); errRollback != nil {
				err = errors.Wrap(errors.Wrap(errors.ErrRollbackTx, errRollback), err)
			}
		}
	}()
	client, err := svc.clients.Save(ctx, cli)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return client, nil
}

func (svc service) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	dbUser, err := svc.clients.RetrieveByIdentity(ctx, identity)
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if err := svc.hasher.Compare(secret, dbUser.Credentials.Secret); err != nil {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrLogin, err)
	}

	var d string
	if domainID != "" {
		d = domainID
	}

	token, err := svc.auth.Issue(ctx, &magistrala.IssueReq{UserId: dbUser.ID, DomainId: &d, Type: uint32(auth.AccessKey)})
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(errIssueToken, err)
	}

	return token, err
}

func (svc service) RefreshToken(ctx context.Context, refreshToken, domainID string) (*magistrala.Token, error) {
	var d string
	if domainID != "" {
		d = domainID
	}

	tokenUserID, err := svc.Identify(ctx, refreshToken)
	if err != nil {
		return &magistrala.Token{}, err
	}

	dbUser, err := svc.clients.RetrieveByID(ctx, tokenUserID)
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if dbUser.Status == mgclients.DisabledStatus {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, errLoginDisableUser)
	}

	return svc.auth.Refresh(ctx, &magistrala.RefreshReq{RefreshToken: refreshToken, DomainId: &d})
}

func (svc service) ViewClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if tokenUserID != id {
		if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
			return mgclients.Client{Name: client.Name, ID: client.ID}, nil
		}
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
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	client.Credentials.Secret = ""

	return client, nil
}

func (svc service) ListClients(ctx context.Context, token string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	userID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}
	if err := svc.checkSuperAdmin(ctx, userID); err != nil {
		return mgclients.ClientsPage{}, err
	}

	pm.Role = mgclients.AllRole
	pg, err := svc.clients.RetrieveAll(ctx, pm)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return pg, err
}

func (svc service) SearchUsers(ctx context.Context, token string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	_, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}

	page := mgclients.Page{
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Name:   pm.Name,
		Id:     pm.Id,
		Role:   mgclients.UserRole,
	}

	cp, err := svc.clients.SearchClients(ctx, page)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return cp, nil
}

func (svc service) UpdateClient(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
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

	client, err = svc.clients.Update(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientTags(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != cli.ID {
		if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}
	client, err = svc.clients.UpdateTags(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return client, nil
}

func (svc service) UpdateClientIdentity(ctx context.Context, token, clientID, identity string) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != clientID {
		if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
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
	cli, err = svc.clients.UpdateIdentity(ctx, cli)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return cli, nil
}

func (svc service) GenerateResetToken(ctx context.Context, email, host string) error {
	client, err := svc.clients.RetrieveByIdentity(ctx, email)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	issueReq := &magistrala.IssueReq{
		UserId: client.ID,
		Type:   uint32(auth.RecoveryKey),
	}
	token, err := svc.auth.Issue(ctx, issueReq)
	if err != nil {
		return errors.Wrap(errRecoveryToken, err)
	}

	return svc.SendPasswordReset(ctx, host, email, client.Name, token.AccessToken)
}

func (svc service) ResetSecret(ctx context.Context, resetToken, secret string) error {
	id, err := svc.Identify(ctx, resetToken)
	if err != nil {
		return err
	}
	c, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	secret, err = svc.hasher.Hash(secret)
	if err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	c = mgclients.Client{
		ID: c.ID,
		Credentials: mgclients.Credentials{
			Identity: c.Credentials.Identity,
			Secret:   secret,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: id,
	}
	if _, err := svc.clients.UpdateSecret(ctx, c); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (svc service) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (mgclients.Client, error) {
	id, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if _, err := svc.IssueToken(ctx, dbClient.Credentials.Identity, oldSecret, ""); err != nil {
		return mgclients.Client{}, err
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	dbClient.Credentials.Secret = newSecret
	dbClient.UpdatedAt = time.Now()
	dbClient.UpdatedBy = id

	dbClient, err = svc.clients.UpdateSecret(ctx, dbClient)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return dbClient, nil
}

func (svc service) SendPasswordReset(_ context.Context, host, email, user, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, user, token)
}

func (svc service) UpdateClientRole(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
		return mgclients.Client{}, err
	}
	client := mgclients.Client{
		ID:        cli.ID,
		Role:      cli.Role,
		UpdatedAt: time.Now(),
		UpdatedBy: tokenUserID,
	}

	if _, err := svc.authorize(ctx, auth.UserType, auth.UsersKind, client.ID, auth.MembershipPermission, auth.PlatformType, auth.MagistralaObject); err != nil {
		return mgclients.Client{}, err
	}

	if err := svc.updateClientPolicy(ctx, cli.ID, cli.Role); err != nil {
		return mgclients.Client{}, err
	}

	client, err = svc.clients.UpdateRole(ctx, client)
	if err != nil {
		// If failed to update role in DB, then revert back to platform admin policy in spicedb
		if errRollback := svc.updateClientPolicy(ctx, cli.ID, mgclients.UserRole); errRollback != nil {
			return mgclients.Client{}, errors.Wrap(errRollback, err)
		}
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
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
		return mgclients.Client{}, err
	}

	return client, nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}
	if tokenUserID != client.ID {
		if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, errors.ErrStatusAlreadyAssigned
	}
	client.UpdatedBy = tokenUserID

	client, err = svc.clients.ChangeStatus(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) DeleteClient(ctx context.Context, token, id string) error {
	client := mgclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    mgclients.DeletedStatus,
	}

	if _, err := svc.changeClientStatus(ctx, token, client); err != nil {
		return err
	}

	return nil
}

func (svc service) ListMembers(ctx context.Context, token, objectKind, objectID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return mgclients.MembersPage{}, err
	}
	var objectType string
	var authzPerm string
	switch objectKind {
	case auth.ThingsKind:
		objectType = auth.ThingType
		authzPerm = pm.Permission
	case auth.DomainsKind:
		objectType = auth.DomainType
		authzPerm = auth.SwitchToPermission(pm.Permission)
	case auth.GroupsKind:
		fallthrough
	default:
		objectType = auth.GroupType
		authzPerm = auth.SwitchToPermission(pm.Permission)
	}

	if _, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, authzPerm, objectType, objectID); err != nil {
		return mgclients.MembersPage{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	duids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
		SubjectType: auth.UserType,
		Permission:  pm.Permission,
		Object:      objectID,
		ObjectType:  objectType,
	})
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}
	if len(duids.Policies) == 0 {
		return mgclients.MembersPage{
			Page: mgclients.Page{Total: 0, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}

	var userIDs []string

	for _, domainUserID := range duids.Policies {
		_, userID := auth.DecodeDomainUserID(domainUserID)
		userIDs = append(userIDs, userID)
	}
	pm.IDs = userIDs

	cp, err := svc.clients.RetrieveAll(ctx, pm)
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	for i, c := range cp.Clients {
		cp.Clients[i] = mgclients.Client{
			ID:        c.ID,
			Name:      c.Name,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Status:    c.Status,
		}
	}

	if pm.ListPerms && len(cp.Clients) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range cp.Clients {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrieveObjectUsersPermissions(ctx, res.GetDomainId(), objectType, objectID, &cp.Clients[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return mgclients.MembersPage{}, err
		}
	}

	return mgclients.MembersPage{
		Page:    cp.Page,
		Members: cp.Clients,
	}, nil
}

func (svc service) retrieveObjectUsersPermissions(ctx context.Context, domainID, objectType, objectID string, client *mgclients.Client) error {
	userID := auth.EncodeDomainUserID(domainID, client.ID)
	permissions, err := svc.listObjectUserPermission(ctx, userID, objectType, objectID)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	client.Permissions = permissions
	return nil
}

func (svc service) listObjectUserPermission(ctx context.Context, userID, objectType, objectID string) ([]string, error) {
	permissions, err := svc.policy.ListPermissions(ctx, policy.PolicyReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Object:      objectID,
		ObjectType:  objectType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(errFailedPermissionsList, err)
	}
	return permissions, nil
}

func (svc *service) checkSuperAdmin(ctx context.Context, adminID string) error {
	if _, err := svc.authorize(ctx, auth.UserType, auth.UsersKind, adminID, auth.AdminPermission, auth.PlatformType, auth.MagistralaObject); err != nil {
		if err := svc.clients.CheckSuperAdmin(ctx, adminID); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return nil
}

func (svc service) identify(ctx context.Context, token string) (*magistrala.IdentityRes, error) {
	res, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return &magistrala.IdentityRes{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return res, nil
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
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	if !res.GetAuthorized() {
		return "", svcerr.ErrAuthorization
	}
	return res.GetId(), nil
}

func (svc service) OAuthCallback(ctx context.Context, client mgclients.Client) (*magistrala.Token, error) {
	rclient, err := svc.clients.RetrieveByIdentity(ctx, client.Credentials.Identity)
	if err != nil {
		switch errors.Contains(err, repoerr.ErrNotFound) {
		case true:
			rclient, err = svc.RegisterClient(ctx, "", client)
			if err != nil {
				return &magistrala.Token{}, err
			}
		default:
			return &magistrala.Token{}, err
		}
	}

	if _, err = svc.authorize(ctx, auth.UserType, auth.UsersKind, rclient.ID, auth.MembershipPermission, auth.PlatformType, auth.MagistralaObject); err != nil {
		if err := svc.addClientPolicy(ctx, rclient.ID, rclient.Role); err != nil {
			return &magistrala.Token{}, err
		}
	}

	claims := &magistrala.IssueReq{
		UserId: rclient.ID,
		Type:   uint32(auth.AccessKey),
	}

	return svc.auth.Issue(ctx, claims)
}

func (svc service) Identify(ctx context.Context, token string) (string, error) {
	user, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return user.GetUserId(), nil
}

func (svc service) addClientPolicy(ctx context.Context, userID string, role mgclients.Role) error {
	policies := []policy.PolicyReq{}

	policies = append(policies, policy.PolicyReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Relation:    auth.MemberRelation,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})

	if role == mgclients.AdminRole {
		policies = append(policies, policy.PolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
	}
	err := svc.policy.AddPolicies(ctx, policies)
	if err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return nil
}

func (svc service) addClientPolicyRollback(ctx context.Context, userID string, role mgclients.Role) error {
	policies := []policy.PolicyReq{}

	policies = append(policies, policy.PolicyReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Relation:    auth.MemberRelation,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})

	if role == mgclients.AdminRole {
		policies = append(policies, policy.PolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
	}
	err := svc.policy.DeletePolicies(ctx, policies)
	if err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	return nil
}

func (svc service) updateClientPolicy(ctx context.Context, userID string, role mgclients.Role) error {
	switch role {
	case mgclients.AdminRole:
		err := svc.policy.AddPolicy(ctx, policy.PolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
		if err != nil {
			return errors.Wrap(svcerr.ErrAddPolicies, err)
		}

		return nil
	case mgclients.UserRole:
		fallthrough
	default:
		err := svc.policy.DeletePolicyFilter(ctx, policy.PolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
		if err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}

		return nil
	}
}
