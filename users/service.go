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
	users      Repository
	idProvider magistrala.IDProvider
	policies   policies.Service
	hasher     Hasher
	email      Emailer
}

// NewService returns a new Users service implementation.
func NewService(token magistrala.TokenServiceClient, urepo Repository, policyService policies.Service, emailer Emailer, hasher Hasher, idp magistrala.IDProvider) Service {
	return service{
		token:      token,
		users:      urepo,
		policies:   policyService,
		hasher:     hasher,
		email:      emailer,
		idProvider: idp,
	}
}

func (svc service) RegisterUser(ctx context.Context, session authn.Session, u User, selfRegister bool) (uc User, err error) {
	if !selfRegister {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
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

func (svc service) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	dbUser, err := svc.users.RetrieveByIdentity(ctx, identity)
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

	dbUser, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if dbUser.Status == DisabledStatus {
		return &magistrala.Token{}, errors.Wrap(svcerr.ErrAuthentication, errLoginDisableUser)
	}

	return svc.token.Refresh(ctx, &magistrala.RefreshReq{RefreshToken: refreshToken, DomainId: &d})
}

func (svc service) ViewUser(ctx context.Context, session authn.Session, id string) (User, error) {
	user, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{
				FirstName:   user.FirstName,
				LastName:    user.LastName,
				ID:          user.ID,
				Credentials: Credentials{UserName: user.Credentials.UserName},
			}, nil
		}
	}

	user.Credentials.Secret = ""

	return user, nil
}

func (svc service) ViewProfile(ctx context.Context, session authn.Session) (User, error) {
	user, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	user.Credentials.Secret = ""

	return user, nil
}

func (svc service) ViewUserByUserName(ctx context.Context, session authn.Session, userName string) (User, error) {
	_, err := svc.Identify(ctx, session)
	if err != nil {
		return User{}, err
	}

	user, err := svc.users.RetrieveByUserName(ctx, userName)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return user, nil
}

func (svc service) ListUsers(ctx context.Context, session authn.Session, pm Page) (UsersPage, error) {
	if err := svc.checkSuperAdmin(ctx, session); err != nil {
		return UsersPage{}, err
	}

	pm.Role = AllRole
	pg, err := svc.users.RetrieveAll(ctx, pm)
	if err != nil {
		return UsersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return pg, err
}

func (svc service) SearchUsers(ctx context.Context, pm Page) (UsersPage, error) {
	page := Page{
		Offset:    pm.Offset,
		Limit:     pm.Limit,
		FirstName: pm.FirstName,
		LastName:  pm.LastName,
		UserName:  pm.UserName,
		Id:        pm.Id,
		Role:      UserRole,
	}

	cp, err := svc.users.SearchUsers(ctx, page)
	if err != nil {
		return UsersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return cp, nil
}

func (svc service) UpdateUser(ctx context.Context, session authn.Session, usr User) (User, error) {
	if session.UserID != usr.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	user := User{
		ID:             usr.ID,
		FirstName:      usr.FirstName,
		LastName:       usr.LastName,
		Metadata:       usr.Metadata,
		Tags:           usr.Tags,
		Role:           usr.Role,
		ProfilePicture: usr.ProfilePicture,
		UpdatedAt:      time.Now(),
		UpdatedBy:      session.UserID,
	}

	user, err := svc.users.Update(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) UpdateUserTags(ctx context.Context, session authn.Session, usr User) (User, error) {
	if session.UserID != usr.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	user := User{
		ID:        usr.ID,
		Tags:      usr.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	user, err := svc.users.Update(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return user, nil
}

func (svc service) UpdateProfilePicture(ctx context.Context, session authn.Session, usr User) (User, error) {
	if session.UserID != usr.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	user := User{
		ID:             usr.ID,
		ProfilePicture: usr.ProfilePicture,
		UpdatedAt:      time.Now(),
		UpdatedBy:      session.UserID,
	}

	user, err := svc.users.Update(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return user, nil
}

func (svc service) UpdateUserIdentity(ctx context.Context, session authn.Session, userID, identity string) (User, error) {
	if session.UserID != userID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	user := User{
		ID:        userID,
		Identity:  identity,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	user, err := svc.users.Update(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByIdentity(ctx, email)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	issueReq := &magistrala.IssueReq{
		UserId: user.ID,
		Type:   uint32(mgauth.RecoveryKey),
	}
	token, err := svc.token.Issue(ctx, issueReq)
	if err != nil {
		return errors.Wrap(errRecoveryToken, err)
	}

	return svc.SendPasswordReset(ctx, host, email, user.Credentials.UserName, token.AccessToken)
}

func (svc service) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	u, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	secret, err = svc.hasher.Hash(secret)
	if err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	u = User{
		ID:       u.ID,
		Identity: u.Identity,
		Credentials: Credentials{
			Secret: secret,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	if _, err := svc.users.UpdateSecret(ctx, u); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (svc service) UpdateUserSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (User, error) {
	dbUser, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if _, err := svc.IssueToken(ctx, dbUser.Identity, oldSecret, ""); err != nil {
		return User{}, err
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	dbUser.Credentials.Secret = newSecret
	dbUser.UpdatedAt = time.Now()
	dbUser.UpdatedBy = session.UserID

	dbUser, err = svc.users.UpdateSecret(ctx, dbUser)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return dbUser, nil
}

func (svc service) UpdateUserNames(ctx context.Context, session authn.Session, usr User) (User, error) {
	if session.UserID != usr.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	if usr.FirstName == "" || usr.LastName == "" {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrMissingNames)
	}

	usr.UpdatedAt = time.Now()
	usr.UpdatedBy = session.UserID

	updatedUser, err := svc.users.UpdateUserNames(ctx, usr)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return updatedUser, nil
}

func (svc service) SendPasswordReset(_ context.Context, host, email, user, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, user, token)
}

func (svc service) UpdateUserRole(ctx context.Context, session authn.Session, usr User) (User, error) {
	if err := svc.checkSuperAdmin(ctx, session); err != nil {
		return User{}, err
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

	client, err := svc.users.Update(ctx, user)
	if err != nil {
		// If failed to update role in DB, then revert back to platform admin policies in spicedb
		if errRollback := svc.updateUserPolicy(ctx, usr.ID, UserRole); errRollback != nil {
			return User{}, errors.Wrap(errRollback, err)
		}
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) EnableUser(ctx context.Context, session authn.Session, id string) (User, error) {
	client := User{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    EnabledStatus,
	}
	user, err := svc.changeUserStatus(ctx, session, client)
	if err != nil {
		return User{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return user, nil
}

func (svc service) DisableUser(ctx context.Context, session authn.Session, id string) (User, error) {
	user := User{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    DisabledStatus,
	}
	user, err := svc.changeUserStatus(ctx, session, user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (svc service) changeUserStatus(ctx context.Context, session authn.Session, user User) (User, error) {
	if session.UserID != user.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}
	dbClient, err := svc.users.RetrieveByID(ctx, user.ID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == user.Status {
		return User{}, errors.ErrStatusAlreadyAssigned
	}
	user.UpdatedBy = session.UserID

	user, err = svc.users.ChangeStatus(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) DeleteUser(ctx context.Context, session authn.Session, id string) error {
	client := User{
		ID:        id,
		UpdatedAt: time.Now(),
		Status:    DeletedStatus,
	}

	if _, err := svc.changeUserStatus(ctx, session, client); err != nil {
		return err
	}

	return nil
}

func (svc service) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm Page) (MembersPage, error) {
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
				return svc.retrieveObjectUsersPermissions(ctx, session.DomainID, objectType, objectID, &up.Users[iter])
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

func (svc service) retrieveObjectUsersPermissions(ctx context.Context, domainID, objectType, objectID string, client *User) error {
	userID := mgauth.EncodeDomainUserID(domainID, client.ID)
	permissions, err := svc.listObjectUserPermission(ctx, userID, objectType, objectID)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	client.Permissions = permissions
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
		if err := svc.users.CheckSuperAdmin(ctx, session.UserID); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
	}

	return nil
}

func (svc service) OAuthCallback(ctx context.Context, user User) (User, error) {
	ruser, err := svc.users.RetrieveByIdentity(ctx, user.Identity)
	if err != nil {
		switch errors.Contains(err, repoerr.ErrNotFound) {
		case true:
			ruser, err = svc.RegisterUser(ctx, authn.Session{}, user, true)
			if err != nil {
				return User{}, err
			}
		default:
			return User{}, err
		}
	}

	return User{
		ID:   ruser.ID,
		Role: ruser.Role,
	}, nil
}

func (svc service) OAuthAddUserPolicy(ctx context.Context, user User) error {
	return svc.addUserPolicy(ctx, user.ID, user.Role)
}

func (svc service) Identify(ctx context.Context, session authn.Session) (string, error) {
	return session.UserID, nil
}

func (svc service) addUserPolicy(ctx context.Context, userID string, role Role) error {
	policyList := []policies.Policy{}

	policyList = append(policyList, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.MemberRelation,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	})

	if role == AdminRole {
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

func (svc service) addUserPolicyRollback(ctx context.Context, userID string, role Role) error {
	policyList := []policies.Policy{}

	policyList = append(policyList, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.MemberRelation,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	})

	if role == AdminRole {
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
	case AdminRole:
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
