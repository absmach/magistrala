// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package authn_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/authn/jwt"
	"github.com/mainflux/mainflux/authn/mocks"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	secret = "secret"
	email  = "test@example.com"
	id     = "testID"
)

func newService() authn.Service {
	repo := mocks.NewKeyRepository()
	uuidProvider := uuid.NewMock()
	t := jwt.New(secret)
	return authn.New(repo, uuidProvider, t)
}

func TestIssue(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		key   authn.Key
		token string
		err   error
	}{
		{
			desc: "issue user key",
			key: authn.Key{
				Type:     authn.UserKey,
				IssuedAt: time.Now(),
			},
			token: secret,
			err:   nil,
		},
		{
			desc: "issue user key with no time",
			key: authn.Key{
				Type: authn.UserKey,
			},
			token: secret,
			err:   authn.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue API key",
			key: authn.Key{
				Type:     authn.APIKey,
				IssuedAt: time.Now(),
			},
			token: secret,
			err:   nil,
		},
		{
			desc: "issue API key unauthorized",
			key: authn.Key{
				Type:     authn.APIKey,
				IssuedAt: time.Now(),
			},
			token: "invalid",
			err:   authn.ErrUnauthorizedAccess,
		},
		{
			desc: "issue API key with no time",
			key: authn.Key{
				Type: authn.APIKey,
			},
			token: secret,
			err:   authn.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue recovery key",
			key: authn.Key{
				Type:     authn.RecoveryKey,
				IssuedAt: time.Now(),
			},
			token: "",
			err:   nil,
		},
		{
			desc: "issue recovery with no issue time",
			key: authn.Key{
				Type: authn.RecoveryKey,
			},
			token: secret,
			err:   authn.ErrInvalidKeyIssuedAt,
		},
	}

	for _, tc := range cases {
		_, _, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRevoke(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := authn.Key{
		Type:     authn.APIKey,
		IssuedAt: time.Now(),
		IssuerID: id,
		Subject:  email,
	}
	newKey, _, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "revoke user key",
			id:    newKey.ID,
			token: secret,
			err:   nil,
		},
		{
			desc:  "revoke non-existing user key",
			id:    newKey.ID,
			token: secret,
			err:   nil,
		},
		{
			desc:  "revoke unauthorized",
			id:    newKey.ID,
			token: "",
			err:   authn.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Revoke(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieve(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.UserKey, IssuedAt: time.Now(), Subject: email, IssuerID: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := authn.Key{
		ID:       "id",
		Type:     authn.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, userToken, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	apiKey, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	_, resetToken, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "retrieve user key",
			id:    apiKey.ID,
			token: userToken,
			err:   nil,
		},
		{
			desc:  "retrieve non-existing user key",
			id:    "invalid",
			token: userToken,
			err:   authn.ErrNotFound,
		},
		{
			desc:  "retrieve unauthorized",
			id:    apiKey.ID,
			token: "wrong",
			err:   authn.ErrUnauthorizedAccess,
		},
		{
			desc:  "retrieve with API token",
			id:    apiKey.ID,
			token: apiToken,
			err:   authn.ErrUnauthorizedAccess,
		},
		{
			desc:  "retrieve with reset token",
			id:    apiKey.ID,
			token: resetToken,
			err:   authn.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.Retrieve(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	_, loginSecret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, recoverySecret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	_, apiSecret, err := svc.Issue(context.Background(), loginSecret, authn.Key{Type: authn.APIKey, IssuerID: id, Subject: email, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	exp1 := time.Now().Add(-2 * time.Second)
	_, expSecret, err := svc.Issue(context.Background(), loginSecret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired user key expected to succeed: %s", err))

	_, invalidSecret, err := svc.Issue(context.Background(), loginSecret, authn.Key{Type: 22, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	cases := []struct {
		desc string
		key  string
		idt  authn.Identity
		err  error
	}{
		{
			desc: "identify login key",
			key:  loginSecret,
			idt:  authn.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify recovery key",
			key:  recoverySecret,
			idt:  authn.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify API key",
			key:  apiSecret,
			idt:  authn.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify expired API key",
			key:  expSecret,
			idt:  authn.Identity{},
			err:  authn.ErrKeyExpired,
		},
		{
			desc: "identify expired key",
			key:  invalidSecret,
			idt:  authn.Identity{},
			err:  authn.ErrUnauthorizedAccess,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			idt:  authn.Identity{},
			err:  authn.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.idt, idt))
	}
}
