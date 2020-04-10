// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"time"

	"github.com/mainflux/mainflux/errors"
)

const (
	loginDuration    = 10 * time.Hour
	recoveryDuration = 5 * time.Minute
	issuerName       = "mainflux.authn"
)

var (
	// ErrUnauthorizedAccess represents unauthorized access.
	ErrUnauthorizedAccess = errors.New("unauthorized access")

	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid owner or ID).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrNotFound indicates a non-existing entity request.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")

	errIssueUser = errors.New("failed to issue new user key")
	errIssueTmp  = errors.New("failed to issue new temporary key")
	errRevoke    = errors.New("failed to remove key")
	errRetrieve  = errors.New("failed to retrieve key data")
	errIdentify  = errors.New("failed to validate token")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Issue issues a new Key.
	Issue(context.Context, string, Key) (Key, error)

	// Revoke removes the Key with the provided id that is
	// issued by the user identified by the provided key.
	Revoke(context.Context, string, string) error

	// Retrieve retrieves data for the Key identified by the provided
	// ID, that is issued by the user identified by the provided key.
	Retrieve(context.Context, string, string) (Key, error)

	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(context.Context, string) (string, error)
}

var _ Service = (*service)(nil)

type service struct {
	keys      KeyRepository
	idp       IdentityProvider
	tokenizer Tokenizer
}

// New instantiates the auth service implementation.
func New(keys KeyRepository, idp IdentityProvider, tokenizer Tokenizer) Service {
	return &service{
		tokenizer: tokenizer,
		keys:      keys,
		idp:       idp,
	}
}

func (svc service) Issue(ctx context.Context, issuer string, key Key) (Key, error) {
	if key.IssuedAt.IsZero() {
		return Key{}, ErrInvalidKeyIssuedAt
	}
	switch key.Type {
	case APIKey:
		return svc.userKey(ctx, issuer, key)
	case RecoveryKey:
		return svc.tmpKey(issuer, recoveryDuration, key)
	default:
		return svc.tmpKey(issuer, loginDuration, key)
	}
}

func (svc service) Revoke(ctx context.Context, issuer, id string) error {
	email, err := svc.login(issuer)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, email, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) Retrieve(ctx context.Context, issuer, id string) (Key, error) {
	email, err := svc.login(issuer)
	if err != nil {
		return Key{}, errors.Wrap(errRetrieve, err)
	}

	return svc.keys.Retrieve(ctx, email, id)
}

func (svc service) Identify(ctx context.Context, token string) (string, error) {
	c, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", errors.Wrap(errIdentify, err)
	}

	switch c.Type {
	case APIKey:
		k, err := svc.keys.Retrieve(ctx, c.Issuer, c.ID)
		if err != nil {
			return "", err
		}
		// Auto revoke expired key.
		if k.Expired() {
			svc.keys.Remove(ctx, c.Issuer, c.ID)
			return "", ErrKeyExpired
		}
		return c.Issuer, nil
	case RecoveryKey, UserKey:
		if c.Issuer != issuerName {
			return "", ErrUnauthorizedAccess
		}
		return c.Secret, nil
	default:
		return "", ErrUnauthorizedAccess
	}
}

func (svc service) tmpKey(issuer string, duration time.Duration, key Key) (Key, error) {
	key.Secret = issuer
	key.Issuer = issuerName
	key.ExpiresAt = key.IssuedAt.Add(duration)
	val, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, errors.Wrap(errIssueTmp, err)
	}

	key.Secret = val
	return key, nil
}

func (svc service) userKey(ctx context.Context, issuer string, key Key) (Key, error) {
	email, err := svc.login(issuer)
	if err != nil {
		return Key{}, errors.Wrap(errIssueUser, err)
	}
	key.Issuer = email

	id, err := svc.idp.ID()
	if err != nil {
		return Key{}, errors.Wrap(errIssueUser, err)
	}
	key.ID = id

	value, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, errors.Wrap(errIssueUser, err)
	}
	key.Secret = value

	if _, err := svc.keys.Save(ctx, key); err != nil {
		return Key{}, errors.Wrap(errIssueUser, err)
	}

	return key, nil
}

func (svc service) login(token string) (string, error) {
	c, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", err
	}
	// Only user key token is valid for login.
	if c.Type != UserKey {
		return "", ErrUnauthorizedAccess
	}

	if c.Secret == "" {
		return "", ErrUnauthorizedAccess
	}
	return c.Secret, nil
}
