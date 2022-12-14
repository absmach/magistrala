package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mainflux/mainflux/auth"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
)

func TestIssue(t *testing.T) {
	svc := newThingAuthService()
	ts := newThingsAuthServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		AuthURL:         ts.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: groupID, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc     string
		token    string
		duration time.Duration
		err      error
	}{
		{
			desc:     "issue login key with empty token",
			token:    "",
			duration: loginDuration,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:     "issue key with an invalid token",
			token:    invalidToken,
			duration: loginDuration,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:     "issue key with no duration",
			token:    token,
			duration: 0,
			err:      nil,
		},
		{
			desc:     "Issue a new key",
			token:    token,
			duration: loginDuration,
			err:      nil,
		},
	}
	for _, tc := range cases {
		_, err := mainfluxSDK.Issue(tc.token, tc.duration)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
func TestRevoke(t *testing.T) {
	svc := newThingAuthService()
	ts := newThingsAuthServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		AuthURL:         ts.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: groupID, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key, err := mainfluxSDK.Issue(token, loginDuration)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "revoke key with empty ID",
			token: token,
			id:    "",
			err:   createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:  "revoke key with invalid token",
			token: invalidToken,
			id:    key.ID,
			err:   createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:  "revoke key with empty token",
			token: "",
			id:    key.ID,
			err:   createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:  "revoke an existing key",
			token: token,
			id:    key.ID,
			err:   nil,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.Revoke(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestRetrieveKey(t *testing.T) {
	svc := newThingAuthService()
	ts := newThingsAuthServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		AuthURL:         ts.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	_, token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: groupID, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key, err := mainfluxSDK.Issue(token, loginDuration)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "retrieve a non-existing key",
			token: token,
			id:    wrongID,
			err:   createError(sdk.ErrFailedFetch, http.StatusNotFound),
		},
		{
			desc:  "retrieve key with empty ID",
			token: token,
			id:    "",
			err:   createError(sdk.ErrFailedFetch, http.StatusBadRequest),
		},
		{
			desc:  "retrieve key with invalid token",
			token: invalidToken,
			id:    key.ID,
			err:   createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
		{
			desc:  "retrieve key with empty token",
			token: "",
			id:    key.ID,
			err:   createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
		{
			desc:  "retrieve an existing key",
			token: token,
			id:    key.ID,
			err:   nil,
		},
	}
	for _, tc := range cases {
		_, err := mainfluxSDK.RetrieveKey(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
