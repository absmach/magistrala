// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"regexp"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/users/postgres"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrRecoveryToken indicates error in generating password recovery token.
	ErrRecoveryToken = errors.New("failed to generate password recovery token")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")

	// ErrFailedPolicyUpdate indicates a failure to update user policy.
	ErrFailedPolicyUpdate = errors.New("failed to update user policy")

	// ErrFailedOwnerUpdate indicates a failure to update user policy.
	ErrFailedOwnerUpdate = errors.New("failed to update user owner")

	// ErrAddPolicies indictaed a failre to add policies.
	errAddPolicies = errors.New("failed to add policies")

	// ErrIssueToken indicates a failure to issue token.
	ErrIssueToken = errors.New("failed to issue token")

	// ErrAddPolicies indictaed a failre to add policies.
	errDeletePolicies = errors.New("failed to delete policies")
)

type service struct {
	clients      postgres.Repository
	idProvider   magistrala.IDProvider
	auth         magistrala.AuthServiceClient
	hasher       Hasher
	email        Emailer
	passRegex    *regexp.Regexp
	selfRegister bool
}

// NewService returns a new Users service implementation.
func NewService(crepo postgres.Repository, authClient magistrala.AuthServiceClient, emailer Emailer, hasher Hasher, idp magistrala.IDProvider, pr *regexp.Regexp, selfRegister bool) Service {
	return service{
		clients:      crepo,
		auth:         authClient,
		hasher:       hasher,
		email:        emailer,
		idProvider:   idp,
		passRegex:    pr,
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
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUniqueID, err)
	}

	if cli.Credentials.Secret == "" {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrMalformedEntity, repoerr.ErrMissingSecret)
	}
	hash, err := svc.hasher.Hash(cli.Credentials.Secret)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
	}
	cli.Credentials.Secret = hash
	if cli.Status != mgclients.DisabledStatus && cli.Status != mgclients.EnabledStatus {
		return mgclients.Client{}, svcerr.ErrInvalidStatus
	}
	if cli.Role != mgclients.UserRole && cli.Role != mgclients.AdminRole {
		return mgclients.Client{}, svcerr.ErrInvalidRole
	}
	cli.ID = clientID
	cli.CreatedAt = time.Now()

	if err := svc.addClientPolicy(ctx, cli.ID, cli.Role); err != nil {
		return mgclients.Client{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addClientPolicyRollback(ctx, cli.ID, cli.Role); errRollback != nil {
				err = errors.Wrap(err, errors.Wrap(repoerr.ErrRollbackTx, errRollback))
			}
		}
	}()
	client, err := svc.clients.Save(ctx, cli)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	return client, nil
}

func (svc service) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	dbUser, err := svc.clients.RetrieveByIdentity(ctx, identity)
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(repoerr.ErrNotFound, err)
	}
	if err := svc.hasher.Compare(secret, dbUser.Credentials.Secret); err != nil {
		return &magistrala.Token{}, errors.Wrap(errors.ErrLogin, err)
	}

	var d string
	if domainID != "" {
		d = domainID
	}
	return svc.auth.Issue(ctx, &magistrala.IssueReq{UserId: dbUser.ID, DomainId: &d, Type: 0})
}

func (svc service) RefreshToken(ctx context.Context, refreshToken, domainID string) (*magistrala.Token, error) {
	var d string
	if domainID != "" {
		d = domainID
	}
	return svc.auth.Refresh(ctx, &magistrala.RefreshReq{RefreshToken: refreshToken, DomainId: &d})
}

func (svc service) ViewClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}

	if tokenUserID != id {
		if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
			return mgclients.Client{}, err
		}
	}

	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
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
		return mgclients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
	}
	client.Credentials.Secret = ""

	return client, nil
}

func (svc service) ListClients(ctx context.Context, token string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	userID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}
	if err := svc.checkSuperAdmin(ctx, userID); err == nil {
		pg, err := svc.clients.RetrieveAll(ctx, pm)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
		return pg, err
	}

	p := mgclients.Page{
		Status:   mgclients.EnabledStatus,
		Offset:   pm.Offset,
		Limit:    pm.Limit,
		Name:     pm.Name,
		Identity: pm.Identity,
		Role:     mgclients.UserRole,
	}
	pg, err := svc.clients.RetrieveAll(ctx, p)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return pg, nil
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
	if err != nil || client.Credentials.Identity == "" {
		return errors.ErrNotFound
	}
	issueReq := &magistrala.IssueReq{
		UserId: client.ID,
		Type:   uint32(auth.RecoveryKey),
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
		return err
	}
	c, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(repoerr.ErrNotFound, err)
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
		return errors.Wrap(svcerr.ErrAuthorization, err)
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
		return mgclients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
	}
	if _, err := svc.IssueToken(ctx, dbClient.Credentials.Identity, oldSecret, ""); err != nil {
		return mgclients.Client{}, errors.Wrap(ErrIssueToken, err)
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
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

	if err := svc.updateClientPolicy(ctx, cli.ID, cli.Role); err != nil {
		return mgclients.Client{}, errors.Wrap(ErrFailedPolicyUpdate, err)
	}
	client, err = svc.clients.UpdateRole(ctx, client)
	if err != nil {
		// If failed to update role in DB, then revert back to platform admin policy in spicedb
		if errRollback := svc.updateClientPolicy(ctx, cli.ID, mgclients.UserRole); errRollback != nil {
			return mgclients.Client{}, errors.Wrap(err, errors.Wrap(repoerr.ErrRollbackTx, errRollback))
		}
		return mgclients.Client{}, errors.Wrap(ErrFailedOwnerUpdate, err)
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
		return mgclients.Client{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	return client, nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	tokenUserID, err := svc.Identify(ctx, token)
	if err != nil {
		return mgclients.Client{}, err
	}
	if err := svc.checkSuperAdmin(ctx, tokenUserID); err != nil {
		return mgclients.Client{}, err
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, mgclients.ErrStatusAlreadyAssigned
	}
	client.UpdatedBy = tokenUserID

	client, err = svc.clients.ChangeStatus(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, err
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
	duids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
		SubjectType: auth.UserType,
		Permission:  pm.Permission,
		Object:      objectID,
		ObjectType:  objectType,
	})
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(repoerr.ErrNotFound, err)
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
		return mgclients.MembersPage{}, errors.Wrap(repoerr.ErrNotFound, err)
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
		return err
	}
	client.Permissions = permissions
	return nil
}

func (svc service) listObjectUserPermission(ctx context.Context, userID, objectType, objectID string) ([]string, error) {
	lp, err := svc.auth.ListPermissions(ctx, &magistrala.ListPermissionsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Object:      objectID,
		ObjectType:  objectType,
	})
	if err != nil {
		return []string{}, err
	}
	return lp.GetPermissions(), nil
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
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return res.GetId(), nil
}

func (svc service) Identify(ctx context.Context, token string) (string, error) {
	user, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return user.GetUserId(), nil
}

func (svc service) addClientPolicy(ctx context.Context, userID string, role mgclients.Role) error {
	var policies magistrala.AddPoliciesReq

	policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Relation:    auth.MemberRelation,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})

	if role == mgclients.AdminRole {
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
	}
	resp, err := svc.auth.AddPolicies(ctx, &policies)
	if err != nil {
		return err
	}
	if !resp.Authorized {
		return errors.ErrAuthorization
	}
	return nil
}

func (svc service) addClientPolicyRollback(ctx context.Context, userID string, role mgclients.Role) error {
	var policies magistrala.DeletePoliciesReq

	policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Relation:    auth.MemberRelation,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})

	if role == mgclients.AdminRole {
		policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
	}
	resp, err := svc.auth.DeletePolicies(ctx, &policies)
	if err != nil {
		return err
	}
	if !resp.Deleted {
		return errors.ErrAuthorization
	}
	return nil
}

func (svc service) updateClientPolicy(ctx context.Context, userID string, role mgclients.Role) error {
	switch role {
	case mgclients.AdminRole:
		resp, err := svc.auth.AddPolicy(ctx, &magistrala.AddPolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
		if err != nil {
			return errors.Wrap(errAddPolicies, err)
		}
		if !resp.Authorized {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return nil
	case mgclients.UserRole:
		fallthrough
	default:
		resp, err := svc.auth.DeletePolicy(ctx, &magistrala.DeletePolicyReq{
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectType:  auth.PlatformType,
			Object:      auth.MagistralaObject,
		})
		if err != nil {
			return errors.Wrap(errDeletePolicies, err)
		}
		if !resp.Deleted {
			return errors.Wrap(errDeletePolicies, err)
		}
		return nil
	}
}
