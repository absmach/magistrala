// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
)

const (
	recoveryDuration = 5 * time.Minute
	defLimit         = 100
)

var (
	// ErrExpiry indicates that the token is expired.
	ErrExpiry = errors.New("token is expired")

	errIssueUser          = errors.New("failed to issue new login key")
	errIssueTmp           = errors.New("failed to issue new temporary key")
	errRevoke             = errors.New("failed to remove key")
	errRetrieve           = errors.New("failed to retrieve key data")
	errIdentify           = errors.New("failed to validate token")
	errPlatform           = errors.New("invalid platform id")
	errCreateDomainPolicy = errors.New("failed to create domain policy")
	errAddPolicies        = errors.New("failed to add policies")
	errRemovePolicies     = errors.New("failed to remove the policies")
	errRollbackPolicy     = errors.New("failed to rollback policy")
	errRemoveLocalPolicy  = errors.New("failed to remove from local policy copy")
	errRemovePolicyEngine = errors.New("failed to remove from policy engine")
)

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
//
//go:generate mockery --name Authz --output=./mocks --filename authz.go --quiet --note "Copyright (c) Abstract Machines"
type Authz interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed to `relation` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, pr policies.Policy) error
}

// Authn specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Authn interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, token string, key Key) (Token, error)

	// Revoke removes the Key with the provided id that is
	// issued by the user identified by the provided key.
	Revoke(ctx context.Context, token, id string) error

	// RetrieveKey retrieves data for the Key identified by the provided
	// ID, that is issued by the user identified by the provided key.
	RetrieveKey(ctx context.Context, token, id string) (Key, error)

	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(ctx context.Context, token string) (Key, error)
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.

//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	Authn
	Authz
	Domains
}

var _ Service = (*service)(nil)

type service struct {
	keys               KeyRepository
	domains            DomainsRepository
	idProvider         magistrala.IDProvider
	evaluator          policies.Evaluator
	policysvc          policies.Service
	tokenizer          Tokenizer
	loginDuration      time.Duration
	refreshDuration    time.Duration
	invitationDuration time.Duration
}

// New instantiates the auth service implementation.
func New(keys KeyRepository, domains DomainsRepository, idp magistrala.IDProvider, tokenizer Tokenizer, policyEvaluator policies.Evaluator, policyService policies.Service, loginDuration, refreshDuration, invitationDuration time.Duration) Service {
	return &service{
		tokenizer:          tokenizer,
		domains:            domains,
		keys:               keys,
		idProvider:         idp,
		evaluator:          policyEvaluator,
		policysvc:          policyService,
		loginDuration:      loginDuration,
		refreshDuration:    refreshDuration,
		invitationDuration: invitationDuration,
	}
}

func (svc service) Issue(ctx context.Context, token string, key Key) (Token, error) {
	key.IssuedAt = time.Now().UTC()
	switch key.Type {
	case APIKey:
		return svc.userKey(ctx, token, key)
	case RefreshKey:
		return svc.refreshKey(ctx, token, key)
	case RecoveryKey:
		return svc.tmpKey(recoveryDuration, key)
	case InvitationKey:
		return svc.invitationKey(ctx, key)
	default:
		return svc.accessKey(ctx, key)
	}
}

func (svc service) Revoke(ctx context.Context, token, id string) error {
	issuerID, _, err := svc.authenticate(token)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, issuerID, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) RetrieveKey(ctx context.Context, token, id string) (Key, error) {
	issuerID, _, err := svc.authenticate(token)
	if err != nil {
		return Key{}, errors.Wrap(errRetrieve, err)
	}

	key, err := svc.keys.Retrieve(ctx, issuerID, id)
	if err != nil {
		return Key{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return key, nil
}

func (svc service) Identify(ctx context.Context, token string) (Key, error) {
	key, err := svc.tokenizer.Parse(token)
	if errors.Contains(err, ErrExpiry) {
		err = svc.keys.Remove(ctx, key.Issuer, key.ID)
		return Key{}, errors.Wrap(svcerr.ErrAuthentication, errors.Wrap(ErrKeyExpired, err))
	}
	if err != nil {
		return Key{}, errors.Wrap(svcerr.ErrAuthentication, errors.Wrap(errIdentify, err))
	}

	switch key.Type {
	case RecoveryKey, AccessKey, InvitationKey, RefreshKey:
		return key, nil
	case APIKey:
		_, err := svc.keys.Retrieve(ctx, key.Issuer, key.ID)
		if err != nil {
			return Key{}, svcerr.ErrAuthentication
		}
		return key, nil
	default:
		return Key{}, svcerr.ErrAuthentication
	}
}

func (svc service) Authorize(ctx context.Context, pr policies.Policy) error {
	if err := svc.PolicyValidation(pr); err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	if pr.SubjectKind == policies.TokenKind {
		key, err := svc.Identify(ctx, pr.Subject)
		if err != nil {
			return errors.Wrap(svcerr.ErrAuthentication, err)
		}
		if key.Subject == "" {
			if pr.ObjectType == policies.GroupType || pr.ObjectType == policies.ThingType || pr.ObjectType == policies.DomainType {
				return svcerr.ErrDomainAuthorization
			}
			return svcerr.ErrAuthentication
		}
		pr.Subject = key.Subject
		pr.Domain = key.Domain
	}
	if err := svc.checkPolicy(ctx, pr); err != nil {
		return err
	}
	return nil
}

func (svc service) checkPolicy(ctx context.Context, pr policies.Policy) error {
	// Domain status is required for if user sent authorization request on things, channels, groups and domains
	if pr.SubjectType == policies.UserType && (pr.ObjectType == policies.GroupType || pr.ObjectType == policies.ThingType || pr.ObjectType == policies.DomainType) {
		domainID := pr.Domain
		if domainID == "" {
			if pr.ObjectType != policies.DomainType {
				return svcerr.ErrDomainAuthorization
			}
			domainID = pr.Object
		}
		if err := svc.checkDomain(ctx, pr.SubjectType, pr.Subject, domainID); err != nil {
			return err
		}
	}
	if err := svc.evaluator.CheckPolicy(ctx, pr); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (svc service) checkDomain(ctx context.Context, subjectType, subject, domainID string) error {
	if err := svc.evaluator.CheckPolicy(ctx, policies.Policy{
		Subject:     subject,
		SubjectType: subjectType,
		Permission:  policies.MembershipPermission,
		Object:      domainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return svcerr.ErrDomainAuthorization
	}

	d, err := svc.domains.RetrieveByID(ctx, domainID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	switch d.Status {
	case EnabledStatus:
	case DisabledStatus:
		if err := svc.evaluator.CheckPolicy(ctx, policies.Policy{
			Subject:     subject,
			SubjectType: subjectType,
			Permission:  policies.AdminPermission,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		}); err != nil {
			return svcerr.ErrDomainAuthorization
		}
	case FreezeStatus:
		if err := svc.evaluator.CheckPolicy(ctx, policies.Policy{
			Subject:     subject,
			SubjectType: subjectType,
			Permission:  policies.AdminPermission,
			Object:      policies.MagistralaObject,
			ObjectType:  policies.PlatformType,
		}); err != nil {
			return svcerr.ErrDomainAuthorization
		}
	default:
		return svcerr.ErrDomainAuthorization
	}

	return nil
}

func (svc service) PolicyValidation(pr policies.Policy) error {
	if pr.ObjectType == policies.PlatformType && pr.Object != policies.MagistralaObject {
		return errPlatform
	}
	return nil
}

func (svc service) tmpKey(duration time.Duration, key Key) (Token, error) {
	key.ExpiresAt = time.Now().Add(duration)
	value, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: value}, nil
}

func (svc service) accessKey(ctx context.Context, key Key) (Token, error) {
	var err error
	key.Type = AccessKey
	key.ExpiresAt = time.Now().Add(svc.loginDuration)

	key.Subject, err = svc.checkUserDomain(ctx, key)
	if err != nil {
		return Token{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	key.ExpiresAt = time.Now().Add(svc.refreshDuration)
	key.Type = RefreshKey
	refresh, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: access, RefreshToken: refresh}, nil
}

func (svc service) invitationKey(ctx context.Context, key Key) (Token, error) {
	var err error
	key.Type = InvitationKey
	key.ExpiresAt = time.Now().Add(svc.invitationDuration)

	key.Subject, err = svc.checkUserDomain(ctx, key)
	if err != nil {
		return Token{}, err
	}

	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: access}, nil
}

func (svc service) refreshKey(ctx context.Context, token string, key Key) (Token, error) {
	k, err := svc.tokenizer.Parse(token)
	if err != nil {
		return Token{}, errors.Wrap(errRetrieve, err)
	}
	if k.Type != RefreshKey {
		return Token{}, errIssueUser
	}
	key.ID = k.ID
	if key.Domain == "" {
		key.Domain = k.Domain
	}
	key.User = k.User
	key.Type = AccessKey

	key.Subject, err = svc.checkUserDomain(ctx, key)
	if err != nil {
		return Token{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	key.ExpiresAt = time.Now().Add(svc.loginDuration)
	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	key.ExpiresAt = time.Now().Add(svc.refreshDuration)
	key.Type = RefreshKey
	refresh, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: access, RefreshToken: refresh}, nil
}

func (svc service) checkUserDomain(ctx context.Context, key Key) (subject string, err error) {
	if key.Domain != "" {
		// Check user is platform admin.
		if err = svc.Authorize(ctx, policies.Policy{
			Subject:     key.User,
			SubjectType: policies.UserType,
			Permission:  policies.AdminPermission,
			Object:      policies.MagistralaObject,
			ObjectType:  policies.PlatformType,
		}); err == nil {
			return key.User, nil
		}
		// Check user is domain member.
		domainUserSubject := EncodeDomainUserID(key.Domain, key.User)
		if err = svc.Authorize(ctx, policies.Policy{
			Subject:     domainUserSubject,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      key.Domain,
			ObjectType:  policies.DomainType,
		}); err != nil {
			return "", err
		}
		return domainUserSubject, nil
	}
	return "", nil
}

func (svc service) userKey(ctx context.Context, token string, key Key) (Token, error) {
	id, sub, err := svc.authenticate(token)
	if err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}

	key.Issuer = id
	if key.Subject == "" {
		key.Subject = sub
	}

	keyID, err := svc.idProvider.ID()
	if err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}
	key.ID = keyID

	if _, err := svc.keys.Save(ctx, key); err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}

	tkn, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}

	return Token{AccessToken: tkn}, nil
}

func (svc service) authenticate(token string) (string, string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", "", errors.Wrap(svcerr.ErrAuthentication, err)
	}
	// Only login key token is valid for login.
	if key.Type != AccessKey || key.Issuer == "" {
		return "", "", svcerr.ErrAuthentication
	}

	return key.Issuer, key.Subject, nil
}

// Switch the relative permission for the relation.
func SwitchToPermission(relation string) string {
	switch relation {
	case policies.AdministratorRelation:
		return policies.AdminPermission
	case policies.EditorRelation:
		return policies.EditPermission
	case policies.ContributorRelation:
		return policies.ViewPermission
	case policies.MemberRelation:
		return policies.MembershipPermission
	case policies.GuestRelation:
		return policies.ViewPermission
	default:
		return relation
	}
}

func (svc service) CreateDomain(ctx context.Context, token string, d Domain) (do Domain, err error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	d.CreatedBy = key.User

	domainID, err := svc.idProvider.ID()
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	d.ID = domainID

	if d.Status != DisabledStatus && d.Status != EnabledStatus {
		return Domain{}, svcerr.ErrInvalidStatus
	}

	d.CreatedAt = time.Now()

	if err := svc.createDomainPolicy(ctx, key.User, domainID, policies.AdministratorRelation); err != nil {
		return Domain{}, errors.Wrap(errCreateDomainPolicy, err)
	}
	defer func() {
		if err != nil {
			if errRollBack := svc.createDomainPolicyRollback(ctx, key.User, domainID, policies.AdministratorRelation); errRollBack != nil {
				err = errors.Wrap(err, errors.Wrap(errRollbackPolicy, errRollBack))
			}
		}
	}()
	dom, err := svc.domains.Save(ctx, d)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return dom, nil
}

func (svc service) RetrieveDomain(ctx context.Context, token, id string) (Domain, error) {
	res, err := svc.Identify(ctx, token)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	domain, err := svc.domains.RetrieveByID(ctx, id)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if err = svc.Authorize(ctx, policies.Policy{
		Subject:     EncodeDomainUserID(id, res.User),
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return Domain{ID: domain.ID, Name: domain.Name, Alias: domain.Alias}, nil
	}
	return domain, nil
}

func (svc service) RetrieveDomainPermissions(ctx context.Context, token, id string) (policies.Permissions, error) {
	res, err := svc.Identify(ctx, token)
	if err != nil {
		return []string{}, err
	}
	domainUserSubject := EncodeDomainUserID(id, res.User)
	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     domainUserSubject,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return []string{}, err
	}

	lp, err := svc.policysvc.ListPermissions(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     domainUserSubject,
		Object:      id,
		ObjectType:  policies.DomainType,
	}, []string{policies.AdminPermission, policies.EditPermission, policies.ViewPermission, policies.MembershipPermission, policies.CreatePermission})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return lp, nil
}

func (svc service) UpdateDomain(ctx context.Context, token, id string, d DomainReq) (Domain, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return Domain{}, err
	}
	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     EncodeDomainUserID(id, key.User),
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  policies.EditPermission,
	}); err != nil {
		return Domain{}, err
	}

	dom, err := svc.domains.Update(ctx, id, key.User, d)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

func (svc service) ChangeDomainStatus(ctx context.Context, token, id string, d DomainReq) (Domain, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     EncodeDomainUserID(id, key.User),
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  policies.AdminPermission,
	}); err != nil {
		return Domain{}, err
	}

	dom, err := svc.domains.Update(ctx, id, key.User, d)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

func (svc service) ListDomains(ctx context.Context, token string, p Page) (DomainsPage, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return DomainsPage{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	p.SubjectID = key.User
	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     key.User,
		SubjectType: policies.UserType,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}); err == nil {
		p.SubjectID = ""
	}
	dp, err := svc.domains.ListDomains(ctx, p)
	if err != nil {
		return DomainsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if p.SubjectID == "" {
		for i := range dp.Domains {
			dp.Domains[i].Permission = policies.AdministratorRelation
		}
	}
	return dp, nil
}

func (svc service) AssignUsers(ctx context.Context, token, id string, userIds []string, relation string) error {
	res, err := svc.Identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     res.User,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  policies.SharePermission,
	}); err != nil {
		return err
	}

	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     res.User,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  SwitchToPermission(relation),
	}); err != nil {
		return err
	}

	for _, userID := range userIds {
		if err := svc.Authorize(ctx, policies.Policy{
			Subject:     userID,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      policies.MagistralaObject,
			ObjectType:  policies.PlatformType,
		}); err != nil {
			return errors.Wrap(svcerr.ErrMalformedEntity, fmt.Errorf("invalid user id : %s ", userID))
		}
	}

	return svc.addDomainPolicies(ctx, id, relation, userIds...)
}

func (svc service) UnassignUser(ctx context.Context, token, id, userID string) error {
	res, err := svc.Identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	pr := policies.Policy{
		Subject:     res.User,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
		Permission:  policies.SharePermission,
	}
	if err := svc.Authorize(ctx, pr); err != nil {
		return err
	}

	pr.Permission = policies.AdminPermission
	if err := svc.Authorize(ctx, pr); err != nil {
		pr.SubjectKind = policies.UsersKind
		// User is not admin.
		pr.Subject = userID
		if err := svc.Authorize(ctx, pr); err == nil {
			// Non admin attempts to remove admin.
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
	}

	if err := svc.policysvc.DeletePolicyFilter(ctx, policies.Policy{
		Subject:     EncodeDomainUserID(id, userID),
		SubjectType: policies.UserType,
	}); err != nil {
		return errors.Wrap(errRemovePolicies, err)
	}

	pc := Policy{
		SubjectType: policies.UserType,
		SubjectID:   userID,
		ObjectType:  policies.DomainType,
		ObjectID:    id,
	}

	if err := svc.domains.DeletePolicies(ctx, pc); err != nil {
		return errors.Wrap(errRemovePolicies, err)
	}

	return nil
}

// IMPROVEMENT NOTE: Take decision: Only Patform admin or both Patform and domain admins can see others users domain.
func (svc service) ListUserDomains(ctx context.Context, token, userID string, p Page) (DomainsPage, error) {
	res, err := svc.Identify(ctx, token)
	if err != nil {
		return DomainsPage{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     res.User,
		SubjectType: policies.UserType,
		Permission:  policies.AdminPermission,
		Object:      policies.MagistralaObject,
		ObjectType:  policies.PlatformType,
	}); err != nil {
		return DomainsPage{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if userID != "" && res.User != userID {
		p.SubjectID = userID
	} else {
		p.SubjectID = res.User
	}
	dp, err := svc.domains.ListDomains(ctx, p)
	if err != nil {
		return DomainsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return dp, nil
}

func (svc service) addDomainPolicies(ctx context.Context, domainID, relation string, userIDs ...string) (err error) {
	var prs []policies.Policy
	var pcs []Policy

	for _, userID := range userIDs {
		prs = append(prs, policies.Policy{
			Subject:     EncodeDomainUserID(domainID, userID),
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Relation:    relation,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		})
		pcs = append(pcs, Policy{
			SubjectType: policies.UserType,
			SubjectID:   userID,
			Relation:    relation,
			ObjectType:  policies.DomainType,
			ObjectID:    domainID,
		})
	}
	if err := svc.policysvc.AddPolicies(ctx, prs); err != nil {
		return errors.Wrap(errAddPolicies, err)
	}
	defer func() {
		if err != nil {
			if errDel := svc.policysvc.DeletePolicies(ctx, prs); errDel != nil {
				err = errors.Wrap(err, errors.Wrap(errRollbackPolicy, errDel))
			}
		}
	}()

	if err = svc.domains.SavePolicies(ctx, pcs...); err != nil {
		return errors.Wrap(errAddPolicies, err)
	}
	return nil
}

func (svc service) createDomainPolicy(ctx context.Context, userID, domainID, relation string) (err error) {
	prs := []policies.Policy{
		{
			Subject:     EncodeDomainUserID(domainID, userID),
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Relation:    relation,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		},
		{
			Subject:     policies.MagistralaObject,
			SubjectType: policies.PlatformType,
			Relation:    policies.PlatformRelation,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		},
	}
	if err := svc.policysvc.AddPolicies(ctx, prs); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if errDel := svc.policysvc.DeletePolicies(ctx, prs); errDel != nil {
				err = errors.Wrap(err, errors.Wrap(errRollbackPolicy, errDel))
			}
		}
	}()
	err = svc.domains.SavePolicies(ctx, Policy{
		SubjectType: policies.UserType,
		SubjectID:   userID,
		Relation:    relation,
		ObjectType:  policies.DomainType,
		ObjectID:    domainID,
	})
	if err != nil {
		return errors.Wrap(errCreateDomainPolicy, err)
	}
	return err
}

func (svc service) createDomainPolicyRollback(ctx context.Context, userID, domainID, relation string) error {
	var err error
	prs := []policies.Policy{
		{
			Subject:     EncodeDomainUserID(domainID, userID),
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Relation:    relation,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		},
		{
			Subject:     policies.MagistralaObject,
			SubjectType: policies.PlatformType,
			Relation:    policies.PlatformRelation,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		},
	}
	if errPolicy := svc.policysvc.DeletePolicies(ctx, prs); errPolicy != nil {
		err = errors.Wrap(errRemovePolicyEngine, errPolicy)
	}
	errPolicyCopy := svc.domains.DeletePolicies(ctx, Policy{
		SubjectType: policies.UserType,
		SubjectID:   userID,
		Relation:    relation,
		ObjectType:  policies.DomainType,
		ObjectID:    domainID,
	})
	if errPolicyCopy != nil {
		err = errors.Wrap(err, errors.Wrap(errRemoveLocalPolicy, errPolicyCopy))
	}
	return err
}

func EncodeDomainUserID(domainID, userID string) string {
	if domainID == "" || userID == "" {
		return ""
	}
	return domainID + "_" + userID
}

func DecodeDomainUserID(domainUserID string) (string, string) {
	if domainUserID == "" {
		return domainUserID, domainUserID
	}
	duid := strings.Split(domainUserID, "_")

	switch {
	case len(duid) == 2:
		return duid[0], duid[1]
	case len(duid) == 1:
		return duid[0], ""
	case len(duid) == 0 || len(duid) > 2:
		fallthrough
	default:
		return "", ""
	}
}

func (svc service) DeleteUserFromDomains(ctx context.Context, id string) (err error) {
	domainsPage, err := svc.domains.ListDomains(ctx, Page{SubjectID: id, Limit: defLimit})
	if err != nil {
		return err
	}

	if domainsPage.Total > defLimit {
		for i := defLimit; i < int(domainsPage.Total); i += defLimit {
			page := Page{SubjectID: id, Offset: uint64(i), Limit: defLimit}
			dp, err := svc.domains.ListDomains(ctx, page)
			if err != nil {
				return err
			}
			domainsPage.Domains = append(domainsPage.Domains, dp.Domains...)
		}
	}

	for _, domain := range domainsPage.Domains {
		req := policies.Policy{
			Subject:     EncodeDomainUserID(domain.ID, id),
			SubjectType: policies.UserType,
		}
		if err := svc.policysvc.DeletePolicyFilter(ctx, req); err != nil {
			return err
		}
	}

	if err := svc.domains.DeleteUserPolicies(ctx, id); err != nil {
		return err
	}

	return nil
}
