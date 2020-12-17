// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	loginDuration    = 10 * time.Hour
	recoveryDuration = 5 * time.Minute
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
// Token is a string value of the actual Key and is used to authenticate
// an AuthN service request.
type Service interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, token string, key Key) (Key, string, error)

	// Revoke removes the Key with the provided id that is
	// issued by the user identified by the provided key.
	Revoke(ctx context.Context, token, id string) error

	// Retrieve retrieves data for the Key identified by the provided
	// ID, that is issued by the user identified by the provided key.
	Retrieve(ctx context.Context, token, id string) (Key, error)

	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(ctx context.Context, token string) (Identity, error)
}

var _ Service = (*service)(nil)

type service struct {
	keys         KeyRepository
	uuidProvider mainflux.IDProvider
	tokenizer    Tokenizer
}

// New instantiates the auth service implementation.
func New(keys KeyRepository, up mainflux.IDProvider, tokenizer Tokenizer) Service {
	return &service{
		tokenizer:    tokenizer,
		keys:         keys,
		uuidProvider: up,
	}
}

func (svc service) Issue(ctx context.Context, token string, key Key) (Key, string, error) {
	if key.IssuedAt.IsZero() {
		return Key{}, "", ErrInvalidKeyIssuedAt
	}
	switch key.Type {
	case APIKey:
		return svc.userKey(ctx, token, key)
	case RecoveryKey:
		return svc.tmpKey(recoveryDuration, key)
	default:
		return svc.tmpKey(loginDuration, key)
	}
}

func (svc service) Revoke(ctx context.Context, token, id string) error {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, issuerID, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) Retrieve(ctx context.Context, token, id string) (Key, error) {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return Key{}, errors.Wrap(errRetrieve, err)
	}

	return svc.keys.Retrieve(ctx, issuerID, id)
}

func (svc service) Identify(ctx context.Context, token string) (Identity, error) {
	key, err := svc.tokenizer.Parse(token)
	if err == ErrAPIKeyExpired {
		err = svc.keys.Remove(ctx, key.IssuerID, key.ID)
		return Identity{}, errors.Wrap(ErrAPIKeyExpired, err)
	}
	if err != nil {
		return Identity{}, errors.Wrap(errIdentify, err)
	}

	switch key.Type {
	case APIKey, RecoveryKey, UserKey:
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	default:
		return Identity{}, ErrUnauthorizedAccess
	}
}

func (svc service) tmpKey(duration time.Duration, key Key) (Key, string, error) {
	key.ExpiresAt = key.IssuedAt.Add(duration)
	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueTmp, err)
	}

	return key, secret, nil
}

func (svc service) userKey(ctx context.Context, token string, key Key) (Key, string, error) {
	id, sub, err := svc.login(token)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	key.IssuerID = id
	if key.Subject == "" {
		key.Subject = sub
	}

	keyID, err := svc.uuidProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}
	key.ID = keyID

	if _, err := svc.keys.Save(ctx, key); err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return key, secret, nil
}

func (svc service) login(token string) (string, string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", "", err
	}
	// Only user key token is valid for login.
	if key.Type != UserKey || key.IssuerID == "" {
		return "", "", ErrUnauthorizedAccess
	}

	return key.IssuerID, key.Subject, nil
}
