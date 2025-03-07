// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/base64"
	"math/rand"
	"strings"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/google/uuid"
)

const (
	recoveryDuration   = 5 * time.Minute
	defLimit           = 100
	randStr            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&&*|+-="
	patPrefix          = "pat"
	patSecretSeparator = "_"
)

var (
	// ErrExpiry indicates that the token is expired.
	ErrExpiry = errors.New("token is expired")

	errIssueUser = errors.New("failed to issue new login key")
	errIssueTmp  = errors.New("failed to issue new temporary key")
	errRevoke    = errors.New("failed to remove key")
	errRetrieve  = errors.New("failed to retrieve key data")
	errIdentify  = errors.New("failed to validate token")
	errPlatform  = errors.New("invalid platform id")

	errMalformedPAT        = errors.New("malformed personal access token")
	errFailedToParseUUID   = errors.New("failed to parse string to UUID")
	errInvalidLenFor2UUIDs = errors.New("invalid input length for 2 UUID, excepted 32 byte")
	errRevokedPAT          = errors.New("revoked pat")
	errCreatePAT           = errors.New("failed to create PAT")
	errUpdatePAT           = errors.New("failed to update PAT")
	errRetrievePAT         = errors.New("failed to retrieve PAT")
	errDeletePAT           = errors.New("failed to delete PAT")
	errInvalidScope        = errors.New("invalid scope")
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
	PATS
}

var _ Service = (*service)(nil)

type service struct {
	keys               KeyRepository
	pats               PATSRepository
	cache              Cache
	hasher             Hasher
	idProvider         supermq.IDProvider
	evaluator          policies.Evaluator
	policysvc          policies.Service
	tokenizer          Tokenizer
	loginDuration      time.Duration
	refreshDuration    time.Duration
	invitationDuration time.Duration
	callback           CallBack
}

// New instantiates the auth service implementation.
func New(keys KeyRepository, pats PATSRepository, cache Cache, hasher Hasher, idp supermq.IDProvider, tokenizer Tokenizer, policyEvaluator policies.Evaluator, policyService policies.Service, loginDuration, refreshDuration, invitationDuration time.Duration, callback CallBack) Service {
	return &service{
		tokenizer:          tokenizer,
		keys:               keys,
		pats:               pats,
		cache:              cache,
		hasher:             hasher,
		idProvider:         idp,
		evaluator:          policyEvaluator,
		policysvc:          policyService,
		loginDuration:      loginDuration,
		refreshDuration:    refreshDuration,
		invitationDuration: invitationDuration,
		callback:           callback,
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
			if pr.ObjectType == policies.GroupType || pr.ObjectType == policies.ClientType || pr.ObjectType == policies.DomainType {
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

	if err := svc.callback.Authorize(ctx, pr); err != nil {
		return err
	}

	return nil
}

func (svc service) checkPolicy(ctx context.Context, pr policies.Policy) error {
	// Domain status is required for if user sent authorization request on clients, channels, groups and domains
	if pr.SubjectType == policies.UserType && (pr.ObjectType == policies.GroupType || pr.ObjectType == policies.ClientType || pr.ObjectType == policies.DomainType) {
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

	return nil
}

func (svc service) PolicyValidation(pr policies.Policy) error {
	if pr.ObjectType == policies.PlatformType && pr.Object != policies.SuperMQObject {
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
			Object:      policies.SuperMQObject,
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

func (svc service) CreatePAT(ctx context.Context, token, name, description string, duration time.Duration) (PAT, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return PAT{}, err
	}

	id, err := svc.idProvider.ID()
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	secret, hash, err := svc.generateSecretAndHash(key.User, id)
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	now := time.Now()
	pat := PAT{
		ID:          id,
		User:        key.User,
		Name:        name,
		Description: description,
		Secret:      hash,
		IssuedAt:    now,
		ExpiresAt:   now.Add(duration),
	}

	if err := svc.pats.Save(ctx, pat); err != nil {
		return PAT{}, errors.Wrap(errCreatePAT, err)
	}
	pat.Secret = secret

	return pat, nil
}

func (svc service) UpdatePATName(ctx context.Context, token, patID, name string) (PAT, error) {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return PAT{}, err
	}
	pat, err := svc.pats.UpdateName(ctx, key.User, patID, name)
	if err != nil {
		return PAT{}, errors.Wrap(errUpdatePAT, err)
	}
	return pat, nil
}

func (svc service) UpdatePATDescription(ctx context.Context, token, patID, description string) (PAT, error) {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return PAT{}, err
	}
	pat, err := svc.pats.UpdateDescription(ctx, key.User, patID, description)
	if err != nil {
		return PAT{}, errors.Wrap(errUpdatePAT, err)
	}
	return pat, nil
}

func (svc service) RetrievePAT(ctx context.Context, token, patID string) (PAT, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return PAT{}, err
	}
	pat, err := svc.pats.Retrieve(ctx, key.User, patID)
	if err != nil {
		return PAT{}, errors.Wrap(errRetrievePAT, err)
	}
	return pat, nil
}

func (svc service) ListPATS(ctx context.Context, token string, pm PATSPageMeta) (PATSPage, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return PATSPage{}, err
	}
	patsPage, err := svc.pats.RetrieveAll(ctx, key.User, pm)
	if err != nil {
		return PATSPage{}, errors.Wrap(errRetrievePAT, err)
	}
	return patsPage, nil
}

func (svc service) DeletePAT(ctx context.Context, token, patID string) error {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return err
	}
	if err := svc.pats.Remove(ctx, key.User, patID); err != nil {
		return errors.Wrap(errDeletePAT, err)
	}
	return nil
}

func (svc service) ResetPATSecret(ctx context.Context, token, patID string, duration time.Duration) (PAT, error) {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return PAT{}, err
	}

	// Generate new HashToken take place here
	secret, hash, err := svc.generateSecretAndHash(key.User, patID)
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	pat, err := svc.pats.UpdateTokenHash(ctx, key.User, patID, hash, time.Now().Add(duration))
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	if err := svc.pats.Reactivate(ctx, key.User, patID); err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	pat.Secret = secret
	pat.Revoked = false
	pat.RevokedAt = time.Time{}
	return pat, nil
}

func (svc service) RevokePATSecret(ctx context.Context, token, patID string) error {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return err
	}

	if err := svc.pats.Revoke(ctx, key.User, patID); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (svc service) RemoveAllPAT(ctx context.Context, token string) error {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.pats.RemoveAllPAT(ctx, key.User); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (svc service) AddScope(ctx context.Context, token, patID string, scopes []Scope) error {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return err
	}

	for i := range len(scopes) {
		scopes[i].ID, err = svc.idProvider.ID()
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		scopes[i].PatID = patID
	}

	err = svc.pats.AddScope(ctx, key.User, scopes)
	if err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return nil
}

func (svc service) RemoveScope(ctx context.Context, token, patID string, scopesIDs ...string) error {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return err
	}

	err = svc.pats.RemoveScope(ctx, key.User, scopesIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (svc service) ListScopes(ctx context.Context, token string, pm ScopesPageMeta) (ScopesPage, error) {
	_, err := svc.authnAuthzUserPAT(ctx, token, pm.PatID)
	if err != nil {
		return ScopesPage{}, err
	}
	patsPage, err := svc.pats.RetrieveScope(ctx, pm)
	if err != nil {
		return ScopesPage{}, errors.Wrap(errRetrievePAT, err)
	}

	return patsPage, nil
}

func (svc service) RemovePATAllScope(ctx context.Context, token, patID string) error {
	_, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return err
	}
	if err := svc.pats.RemoveAllScope(ctx, patID); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (svc service) IdentifyPAT(ctx context.Context, secret string) (PAT, error) {
	parts := strings.Split(secret, patSecretSeparator)
	if len(parts) != 3 && parts[0] != patPrefix {
		return PAT{}, errors.Wrap(svcerr.ErrAuthentication, errMalformedPAT)
	}
	userID, patID, err := decode(parts[1])
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrAuthentication, errMalformedPAT)
	}
	secretHash, revoked, expired, err := svc.pats.RetrieveSecretAndRevokeStatus(ctx, userID.String(), patID.String())
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if revoked {
		return PAT{}, errors.Wrap(svcerr.ErrAuthentication, errRevokedPAT)
	}
	if expired {
		return PAT{}, errors.Wrap(svcerr.ErrAuthentication, ErrExpiry)
	}
	if err := svc.hasher.Compare(secret, secretHash); err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return PAT{ID: patID.String(), User: userID.String()}, nil
}

func (svc service) AuthorizePAT(ctx context.Context, userID, patID string, entityType EntityType, optionalDomainID string, operation Operation, entityID string) error {
	if err := svc.pats.CheckScope(ctx, userID, patID, entityType, optionalDomainID, operation, entityID); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return nil
}

func (svc service) generateSecretAndHash(userID, patID string) (string, string, error) {
	uID, err := uuid.Parse(userID)
	if err != nil {
		return "", "", errors.Wrap(errFailedToParseUUID, err)
	}
	pID, err := uuid.Parse(patID)
	if err != nil {
		return "", "", errors.Wrap(errFailedToParseUUID, err)
	}

	secret := patPrefix + patSecretSeparator + encode(uID, pID) + patSecretSeparator + generateRandomString(100)
	secretHash, err := svc.hasher.Hash(secret)
	return secret, secretHash, err
}

func encode(userID, patID uuid.UUID) string {
	c := append(userID[:], patID[:]...)
	return base64.StdEncoding.EncodeToString(c)
}

func decode(encoded string) (uuid.UUID, uuid.UUID, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	if len(data) != 32 {
		return uuid.Nil, uuid.Nil, errInvalidLenFor2UUIDs
	}

	var userID, patID uuid.UUID
	copy(userID[:], data[:16])
	copy(patID[:], data[16:])

	return userID, patID, nil
}

func generateRandomString(n int) string {
	letterRunes := []rune(randStr)
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (svc service) authnAuthzUserPAT(ctx context.Context, token, patID string) (Key, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return Key{}, err
	}

	_, err = svc.pats.Retrieve(ctx, key.User, patID)
	if err != nil {
		return Key{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return key, nil
}
