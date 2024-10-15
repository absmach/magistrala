// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/authn"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
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
	token      magistrala.TokenServiceClient
	clients    postgres.Repository
	idProvider magistrala.IDProvider
	policies   policies.Service
	hasher     Hasher
	email      Emailer
}

// NewService returns a new Users service implementation.
func NewService(token magistrala.TokenServiceClient, crepo postgres.Repository, policyService policies.Service, emailer Emailer, hasher Hasher, idp magistrala.IDProvider) Service {
	return service{
		token:      token,
		clients:    crepo,
		policies:   policyService,
		hasher:     hasher,
		email:      emailer,
		idProvider: idp,
	}
}

func (svc service) RegisterClient(ctx context.Context, session authn.Session, cli mgclients.Client, selfRegister bool) (rc mgclients.Client, err error) {
	if !selfRegister {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return mgclients.Client{}, err
		}
	}

	userID, err := svc.idProvider.ID()
	if err != nil {
		return User{}, err
	}

	if u.Credentials.Secret != "" {
		hash, err := svc.hasher.Hash(u.Credentials.Secret)
		if err != nil {
			return User{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
		}
		u.Credentials.Secret = hash
	}

	if u.Credentials.UserName == "" {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}

	if u.Status != DisabledStatus && u.Status != EnabledStatus {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrInvalidStatus)
	}

	if u.Role != UserRole && u.Role != AdminRole {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrInvalidRole)
	}
	u.ID = userID
	u.CreatedAt = time.Now()

	if err := svc.addUserPolicy(ctx, u.ID, u.Role); err != nil {
		return User{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addUserPolicyRollback(ctx, u.ID, u.Role); errRollback != nil {
				err = errors.Wrap(errors.Wrap(errors.ErrRollbackTx, errRollback), err)
			}
		}
	}()
	user, err := svc.users.Save(ctx, u)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return user, nil
}

func (svc service) IssueToken(ctx context.Context, id, secret, domainID string) (*magistrala.Token, error) {
	dbUser, err := svc.users.RetrieveByID(ctx, id)
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

	token, err := svc.token.Issue(ctx, &magistrala.IssueReq{UserId: dbUser.ID, DomainId: &d, Type: uint32(mgauth.AccessKey)})
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(errIssueToken, err)
	}

	return token, err
}

func (svc service) RefreshToken(ctx context.Context, session authn.Session, refreshToken, domainID string) (*magistrala.Token, error) {
	var d string
	if domainID != "" {
		d = domainID
	}

	dbUser, err := svc.clients.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if dbUser.Status == DisabledStatus {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, errLoginDisableUser)
	}

	return svc.token.Refresh(ctx, &magistrala.RefreshReq{RefreshToken: refreshToken, DomainId: &d})
}

func (svc service) ViewClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return mgclients.Client{Name: client.Name, ID: client.ID}, nil
		}
	}

	user.Credentials.Secret = ""

	return user, nil
}

func (svc service) ViewProfile(ctx context.Context, session authn.Session) (mgclients.Client, error) {
	client, err := svc.clients.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	user.Credentials.Secret = ""

	return user, nil
}

func (svc service) ListClients(ctx context.Context, session authn.Session, pm mgclients.Page) (mgclients.ClientsPage, error) {
	if err := svc.checkSuperAdmin(ctx, session); err != nil {
		return mgclients.ClientsPage{}, err
	}

	pm.Role = AllRole
	pg, err := svc.users.RetrieveAll(ctx, pm)
	if err != nil {
		return UsersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return pg, err
}

func (svc service) SearchUsers(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	page := mgclients.Page{
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Name:   pm.Name,
		Id:     pm.Id,
		Role:   UserRole,
	}

	cp, err := svc.users.SearchUsers(ctx, page)
	if err != nil {
		return UsersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return cp, nil
}

func (svc service) UpdateClient(ctx context.Context, session authn.Session, cli mgclients.Client) (mgclients.Client, error) {
	if session.UserID != cli.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return mgclients.Client{}, err
		}
	}

	user := User{
		ID:        usr.ID,
		Name:      usr.Name,
		Metadata:  usr.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}

	client, err := svc.clients.Update(ctx, client)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) UpdateClientTags(ctx context.Context, session authn.Session, cli mgclients.Client) (mgclients.Client, error) {
	if session.UserID != cli.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return mgclients.Client{}, err
		}
	}

	user := User{
		ID:        usr.ID,
		Tags:      usr.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.UpdateTags(ctx, client)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return user, nil
}

func (svc service) UpdateClientIdentity(ctx context.Context, session authn.Session, clientID, identity string) (mgclients.Client, error) {
	if session.UserID != clientID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return mgclients.Client{}, err
		}
	}

	usr := User{
		ID: userID,
		Credentials: Credentials{
			Identity: identity,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	cli, err := svc.clients.UpdateIdentity(ctx, cli)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return usr, nil
}

func (svc service) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByUserName(ctx, email)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	issueReq := &magistrala.IssueReq{
		UserId: client.ID,
		Type:   uint32(mgauth.RecoveryKey),
	}
	token, err := svc.token.Issue(ctx, issueReq)
	if err != nil {
		return errors.Wrap(errRecoveryToken, err)
	}

	return svc.SendPasswordReset(ctx, host, email, user.Credentials.UserName, token.AccessToken)
}

func (svc service) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	c, err := svc.clients.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	secret, err = svc.hasher.Hash(secret)
	if err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	u = User{
		ID: u.ID,
		Credentials: Credentials{
			UserName: u.Credentials.UserName,
			Secret:   secret,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	if _, err := svc.users.UpdateSecret(ctx, u); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (svc service) UpdateClientSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (mgclients.Client, error) {
	dbClient, err := svc.clients.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if _, err := svc.IssueToken(ctx, dbUser.ID, oldSecret, ""); err != nil {
		return User{}, err
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	dbClient.Credentials.Secret = newSecret
	dbClient.UpdatedAt = time.Now()
	dbClient.UpdatedBy = session.UserID

	dbUser, err = svc.users.UpdateSecret(ctx, dbUser)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return dbUser, nil
}

func (svc service) SendPasswordReset(_ context.Context, host, email, user, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, user, token)
}

func (svc service) UpdateClientRole(ctx context.Context, session authn.Session, cli mgclients.Client) (mgclients.Client, error) {
	if err := svc.checkSuperAdmin(ctx, session); err != nil {
		return mgclients.Client{}, err
	}
	user := User{
		ID:        usr.ID,
		Role:      usr.Role,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}

	if err := svc.updateUserPolicy(ctx, usr.ID, usr.Role); err != nil {
		return User{}, err
	}

	client, err := svc.clients.UpdateRole(ctx, client)
	if err != nil {
		// If failed to update role in DB, then revert back to platform admin policies in spicedb
		if errRollback := svc.updateClientPolicy(ctx, cli.ID, mgclients.UserRole); errRollback != nil {
			return mgclients.Client{}, errors.Wrap(errRollback, err)
		}
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) EnableClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    EnabledStatus,
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return User{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return user, nil
}

func (svc service) DisableClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    DisabledStatus,
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (svc service) changeClientStatus(ctx context.Context, session authn.Session, client mgclients.Client) (mgclients.Client, error) {
	if session.UserID != client.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return mgclients.Client{}, err
		}
	}
	dbUser, err := svc.users.RetrieveByID(ctx, user.ID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbUser.Status == user.Status {
		return User{}, errors.ErrStatusAlreadyAssigned
	}
	client.UpdatedBy = session.UserID

	user, err = svc.users.ChangeStatus(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) DeleteClient(ctx context.Context, session authn.Session, id string) error {
	client := mgclients.Client{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    DeletedStatus,
	}

	if _, err := svc.changeClientStatus(ctx, session, client); err != nil {
		return err
	}

	return nil
}

func (svc service) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	var objectType string
	switch objectKind {
	case policies.ThingsKind:
		objectType = policies.ThingType
	case policies.DomainsKind:
		objectType = policies.DomainType
	case policies.GroupsKind:
		fallthrough
	default:
		objectType = policies.GroupType
	}

	duids, err := svc.policies.ListAllSubjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Permission:  pm.Permission,
		Object:      objectID,
		ObjectType:  objectType,
	})
	if err != nil {
		return MembersPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}
	if len(duids.Policies) == 0 {
		return MembersPage{
			Page: Page{Total: 0, Offset: pm.Offset, Limit: pm.Limit},
		}, nil
	}

	var userIDs []string

	for _, domainUserID := range duids.Policies {
		_, userID := mgauth.DecodeDomainUserID(domainUserID)
		userIDs = append(userIDs, userID)
	}
	pm.IDs = userIDs

	up, err := svc.users.RetrieveAll(ctx, pm)
	if err != nil {
		return MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	for i, u := range up.Users {
		up.Users[i] = User{
			ID:        u.ID,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Credentials: Credentials{
				UserName: u.Credentials.UserName,
			},
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
			Status:    u.Status,
		}
	}

	if pm.ListPerms && len(up.Users) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range up.Users {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrieveObjectUsersPermissions(ctx, session.DomainID, objectType, objectID, &cp.Clients[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return MembersPage{}, err
		}
	}

	return MembersPage{
		Page:    up.Page,
		Members: up.Users,
	}, nil
}

func (svc service) retrieveObjectUsersPermissions(ctx context.Context, domainID, objectType, objectID string, client *mgclients.Client) error {
	userID := mgauth.EncodeDomainUserID(domainID, client.ID)
	permissions, err := svc.listObjectUserPermission(ctx, userID, objectType, objectID)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	user.Permissions = permissions
	return nil
}

func (svc service) listObjectUserPermission(ctx context.Context, userID, objectType, objectID string) ([]string, error) {
	permissions, err := svc.policies.ListPermissions(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      objectID,
		ObjectType:  objectType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(errFailedPermissionsList, err)
	}
	return permissions, nil
}

func (svc *service) checkSuperAdmin(ctx context.Context, session authn.Session) error {
	if !session.SuperAdmin {
		if err := svc.clients.CheckSuperAdmin(ctx, session.UserID); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
	}

	return nil
}

func (svc service) OAuthCallback(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	rclient, err := svc.clients.RetrieveByIdentity(ctx, client.Credentials.Identity)
	if err != nil {
		switch errors.Contains(err, repoerr.ErrNotFound) {
		case true:
			rclient, err = svc.RegisterClient(ctx, authn.Session{}, client, true)
			if err != nil {
				return mgclients.Client{}, err
			}
		default:
			return mgclients.Client{}, err
		}
	}

	return mgclients.Client{
		ID:   rclient.ID,
		Role: rclient.Role,
	}, nil
}

func (svc service) OAuthAddClientPolicy(ctx context.Context, client mgclients.Client) error {
	return svc.addClientPolicy(ctx, client.ID, client.Role)
}

func (svc service) Identify(ctx context.Context, session authn.Session) (string, error) {
	return session.UserID, nil
}

func (svc service) addClientPolicy(ctx context.Context, userID string, role mgclients.Role) error {
	policyList := []policies.Policy{}

	policyList = append(policyList, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.MemberRelation,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	})

	if role == mgclients.AdminRole {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.MagistralaObject,
		})
	}
	err := svc.policies.AddPolicies(ctx, policyList)
	if err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return nil
}

func (svc service) addClientPolicyRollback(ctx context.Context, userID string, role mgclients.Role) error {
	policyList := []policies.Policy{}

	policyList = append(policyList, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.MemberRelation,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	})

	if role == mgclients.AdminRole {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.MagistralaObject,
		})
	}
	err := svc.policies.DeletePolicies(ctx, policyList)
	if err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	return nil
}

func (svc service) updateUserPolicy(ctx context.Context, userID string, role Role) error {
	switch role {
	case mgclients.AdminRole:
		err := svc.policies.AddPolicy(ctx, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.MagistralaObject,
		})
		if err != nil {
			return errors.Wrap(svcerr.ErrAddPolicies, err)
		}

		return nil
	case UserRole:
		fallthrough
	default:
		err := svc.policies.DeletePolicyFilter(ctx, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.MagistralaObject,
		})
		if err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}

		return nil
	}
}
