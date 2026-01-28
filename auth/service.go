// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/google/uuid"
)

const (
	recoveryDuration   = 5 * time.Minute
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
	errRoleAuth  = errors.New("failed to authorize user role")

	errMalformedPAT        = errors.New("malformed personal access token")
	errFailedToParseUUID   = errors.New("failed to parse string to UUID")
	errInvalidLenFor2UUIDs = errors.New("invalid input length for 2 UUID, excepted 32 byte")
	errRevokedPAT          = errors.NewServiceError("revoked pat")
	errCreatePAT           = errors.NewServiceError("failed to create PAT")
	errUpdatePAT           = errors.NewServiceError("failed to update PAT")
	errRetrievePAT         = errors.NewServiceError("failed to retrieve PAT")
	errDeletePAT           = errors.NewServiceError("failed to delete PAT")
	errInvalidScope        = errors.New("invalid scope")
)

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed to `relation` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, pr policies.Policy, patAuthz *PATAuthz) error
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

	// RetrieveJWKS retrieves public keys to validate issued tokens.
	RetrieveJWKS() []PublicKeyInfo
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
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
}

// New instantiates the auth service implementation.
func New(keys KeyRepository, pats PATSRepository, cache Cache, hasher Hasher, idp supermq.IDProvider, tokenizer Tokenizer, policyEvaluator policies.Evaluator, policyService policies.Service, loginDuration, refreshDuration, invitationDuration time.Duration) Service {
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
		return svc.tmpKey(ctx, recoveryDuration, key)
	case InvitationKey:
		return svc.invitationKey(ctx, key)
	default:
		return svc.accessKey(ctx, key)
	}
}

func (svc service) Revoke(ctx context.Context, token, id string) error {
	issuerID, _, err := svc.authenticate(ctx, token)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, issuerID, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) RetrieveKey(ctx context.Context, token, id string) (Key, error) {
	issuerID, _, err := svc.authenticate(ctx, token)
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
	key, err := svc.tokenizer.Parse(ctx, token)
	if errors.Contains(err, ErrExpiry) {
		err = svc.keys.Remove(ctx, key.Issuer, key.ID)
		return Key{}, errors.Wrap(svcerr.ErrAuthentication, errors.Wrap(ErrKeyExpired, err))
	}
	if err != nil {
		return Key{}, errors.Wrap(svcerr.ErrAuthentication, errors.Wrap(errIdentify, err))
	}

	switch key.Type {
	case PersonalAccessToken:
		res, err := svc.IdentifyPAT(ctx, token)
		if err != nil {
			return Key{}, err
		}
		return Key{ID: res.ID, Type: PersonalAccessToken, Subject: res.User, Role: res.Role}, nil
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

func (svc service) RetrieveJWKS() []PublicKeyInfo {
	keys, err := svc.tokenizer.RetrieveJWKS()
	if err != nil {
		return nil
	}
	return keys
}

func (svc service) Authorize(ctx context.Context, pr policies.Policy, patAuthz *PATAuthz) error {
	if patAuthz != nil {
		if err := svc.AuthorizePAT(ctx, patAuthz.UserID, patAuthz.PatID, patAuthz.EntityType, patAuthz.Domain, patAuthz.Operation, patAuthz.EntityID); err != nil {
			return err
		}
	}

	if err := svc.PolicyValidation(pr); err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	if err := svc.checkPolicy(ctx, pr); err != nil {
		return err
	}

	return nil
}

func (svc service) checkPolicy(ctx context.Context, pr policies.Policy) error {
	if err := svc.evaluator.CheckPolicy(ctx, pr); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (svc service) PolicyValidation(pr policies.Policy) error {
	if pr.ObjectType == policies.PlatformType && pr.Object != policies.SuperMQObject {
		return errPlatform
	}
	return nil
}

func (svc service) tmpKey(ctx context.Context, duration time.Duration, key Key) (Token, error) {
	key.ExpiresAt = time.Now().UTC().Add(duration)
	if err := svc.checkUserRole(ctx, key); err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}
	value, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: value}, nil
}

func (svc service) accessKey(ctx context.Context, key Key) (Token, error) {
	var err error
	key.Type = AccessKey
	key.ExpiresAt = time.Now().UTC().Add(svc.loginDuration)

	if err := svc.checkUserRole(ctx, key); err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}

	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	key.ExpiresAt = time.Now().UTC().Add(svc.refreshDuration)
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
	key.ExpiresAt = time.Now().UTC().Add(svc.invitationDuration)

	if err := svc.checkUserRole(ctx, key); err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: access}, nil
}

func (svc service) refreshKey(ctx context.Context, token string, key Key) (Token, error) {
	k, err := svc.tokenizer.Parse(ctx, token)
	if err != nil {
		return Token{}, errors.Wrap(errRetrieve, err)
	}
	if k.Type != RefreshKey {
		return Token{}, errIssueUser
	}
	key.ID = k.ID
	key.Type = AccessKey
	key.Subject = k.Subject

	if err := svc.checkUserRole(ctx, key); err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}
	key.Role = k.Role

	key.ExpiresAt = time.Now().UTC().Add(svc.loginDuration)
	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	key.ExpiresAt = time.Now().UTC().Add(svc.refreshDuration)
	key.Type = RefreshKey
	refresh, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Token{}, errors.Wrap(errIssueTmp, err)
	}

	return Token{AccessToken: access, RefreshToken: refresh}, nil
}

func (svc service) checkUserRole(ctx context.Context, key Key) (err error) {
	switch key.Role {
	case AdminRole:
		if err = svc.Authorize(ctx, policies.Policy{
			Subject:     key.Subject,
			SubjectType: policies.UserType,
			Permission:  policies.AdminPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		}, nil); err != nil {
			return errRoleAuth
		}
		return nil
	case UserRole:
		if err = svc.Authorize(ctx, policies.Policy{
			Subject:     key.Subject,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		}, nil); err != nil {
			return errRoleAuth
		}
		return nil
	default:
		return nil
	}
}

func (svc service) getUserRole(ctx context.Context, userID string) (role Role) {
	rl := UserRole
	if err := svc.Authorize(ctx, policies.Policy{
		Subject:     userID,
		SubjectType: policies.UserType,
		Permission:  policies.AdminPermission,
		Object:      policies.SuperMQObject,
		ObjectType:  policies.PlatformType,
	}, nil); err == nil {
		rl = AdminRole
	}

	return rl
}

func (svc service) userKey(ctx context.Context, token string, key Key) (Token, error) {
	id, sub, err := svc.authenticate(ctx, token)
	if err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
	}

	key.Issuer = id
	if key.Subject == "" {
		key.Subject = sub
	}
	if err := svc.checkUserRole(ctx, key); err != nil {
		return Token{}, errors.Wrap(errIssueUser, err)
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

func (svc service) authenticate(ctx context.Context, token string) (string, string, error) {
	key, err := svc.tokenizer.Parse(ctx, token)
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
	secret, hash, err := svc.generateSecretAndHash(key.Subject, id)
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	now := time.Now().UTC()
	pat := PAT{
		ID:          id,
		User:        key.Subject,
		Name:        name,
		Description: description,
		Secret:      hash,
		IssuedAt:    now,
		ExpiresAt:   now.Add(duration),
		Status:      ActiveStatus,
		Revoked:     false,
	}

	if err := pat.Validate(); err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrCreateEntity, err)
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
	pat, err := svc.pats.UpdateName(ctx, key.Subject, patID, name)
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
	pat, err := svc.pats.UpdateDescription(ctx, key.Subject, patID, description)
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
	pat, err := svc.pats.Retrieve(ctx, key.Subject, patID)
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
	patsPage, err := svc.pats.RetrieveAll(ctx, key.Subject, pm)
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
	if err := svc.pats.Remove(ctx, key.Subject, patID); err != nil {
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
	secret, hash, err := svc.generateSecretAndHash(key.Subject, patID)
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	pat, err := svc.pats.UpdateTokenHash(ctx, key.Subject, patID, hash, time.Now().UTC().Add(duration))
	if err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	if err := svc.pats.Reactivate(ctx, key.Subject, patID); err != nil {
		return PAT{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	pat.Secret = secret
	pat.Status = ActiveStatus
	pat.RevokedAt = time.Time{}
	return pat, nil
}

func (svc service) RevokePATSecret(ctx context.Context, token, patID string) error {
	key, err := svc.authnAuthzUserPAT(ctx, token, patID)
	if err != nil {
		return err
	}

	if err := svc.pats.Revoke(ctx, key.Subject, patID); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (svc service) RemoveAllPAT(ctx context.Context, token string) error {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.pats.RemoveAllPAT(ctx, key.Subject); err != nil {
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

	err = svc.pats.AddScope(ctx, key.Subject, scopes)
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

	err = svc.pats.RemoveScope(ctx, key.Subject, scopesIDs...)
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
	role := svc.getUserRole(ctx, userID.String())
	pat := PAT{ID: patID.String(), User: userID.String(), Role: role}
	return pat, nil
}

func (svc service) AuthorizePAT(ctx context.Context, userID, patID string, entityType EntityType, domainID string, operation string, entityID string) error {
	if err := svc.pats.CheckScope(ctx, userID, patID, entityType, domainID, operation, entityID); err != nil {
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

	randomPart, err := generateRandomString(100)
	if err != nil {
		return "", "", err
	}

	secret := patPrefix + patSecretSeparator + encode(uID, pID) + patSecretSeparator + randomPart
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

func generateRandomString(n int) (string, error) {
	letterRunes := []rune(randStr)
	b := make([]rune, n)
	randBytes := make([]byte, n)

	if _, err := rand.Read(randBytes); err != nil {
		return "", errors.Wrap(errors.New("failed to generate random string"), err)
	}

	for i := range b {
		b[i] = letterRunes[int(randBytes[i])%len(letterRunes)]
	}
	return string(b), nil
}

func (svc service) authnAuthzUserPAT(ctx context.Context, token, patID string) (Key, error) {
	key, err := svc.Identify(ctx, token)
	if err != nil {
		return Key{}, err
	}

	_, err = svc.pats.Retrieve(ctx, key.Subject, patID)
	if err != nil {
		if errors.Contains(err, repoerr.ErrNotFound) {
			return Key{}, svcerr.ErrNotFound
		}
		return Key{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return key, nil
}
