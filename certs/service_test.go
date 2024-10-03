// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/certs/mocks"
	"github.com/absmach/magistrala/certs/pki"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	invalid   = "invalid"
	email     = "user@example.com"
	token     = "token"
	thingsNum = 1
	thingKey  = "thingKey"
	thingID   = "1"
	ttl       = "1h"
	certNum   = 10
	validID   = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

func newService(_ *testing.T) (certs.Service, *mocks.Repository, *mocks.Agent, *authmocks.AuthClient, *sdkmocks.SDK) {
	repo := new(mocks.Repository)
	agent := new(mocks.Agent)
	auth := new(authmocks.AuthClient)
	sdk := new(sdkmocks.SDK)

	return certs.New(auth, repo, sdk, agent), repo, agent, auth, sdk
}

var cert = certs.Cert{
	OwnerID: validID,
	ThingID: thingID,
	Serial:  "",
	Expire:  time.Time{},
}

func TestIssueCert(t *testing.T) {
	svc, repo, agent, auth, sdk := newService(t)
	cases := []struct {
		token        string
		desc         string
		thingID      string
		ttl          string
		key          string
		pki          pki.Cert
		identifyRes  *magistrala.IdentityRes
		identifyErr  error
		thingErr     errors.SDKError
		issueCertErr error
		repoErr      error
		err          error
	}{
		{
			desc:    "issue new cert",
			token:   token,
			thingID: thingID,
			ttl:     ttl,
			pki: pki.Cert{
				ClientCert:     "",
				IssuingCA:      "",
				CAChain:        []string{},
				ClientKey:      "",
				PrivateKeyType: "",
				Serial:         "",
				Expire:         0,
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:    "issue new cert for non existing thing id",
			token:   token,
			thingID: "2",
			ttl:     ttl,
			pki: pki.Cert{
				ClientCert:     "",
				IssuingCA:      "",
				CAChain:        []string{},
				ClientKey:      "",
				PrivateKeyType: "",
				Serial:         "",
				Expire:         0,
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			thingErr:    errors.NewSDKError(errors.ErrMalformedEntity),
			err:         certs.ErrFailedCertCreation,
		},
		{
			desc:    "issue new cert for invalid token",
			token:   invalid,
			thingID: thingID,
			ttl:     ttl,
			pki: pki.Cert{
				ClientCert:     "",
				IssuingCA:      "",
				CAChain:        []string{},
				ClientKey:      "",
				PrivateKeyType: "",
				Serial:         "",
				Expire:         0,
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		sdkCall := sdk.On("Thing", tc.thingID, tc.token).Return(mgsdk.Thing{ID: tc.thingID, Credentials: mgsdk.Credentials{Secret: thingKey}}, tc.thingErr)
		agentCall := agent.On("IssueCert", thingKey, tc.ttl).Return(tc.pki, tc.issueCertErr)
		repoCall := repo.On("Save", context.Background(), mock.Anything).Return("", tc.repoErr)

		c, err := svc.IssueCert(context.Background(), tc.token, tc.thingID, tc.ttl)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		cert, _ := certs.ReadCert([]byte(c.ClientCert))
		if cert != nil {
			assert.True(t, strings.Contains(cert.Subject.CommonName, thingKey), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, thingKey, cert.Subject.CommonName))
		}
		authCall.Unset()
		sdkCall.Unset()
		agentCall.Unset()
		repoCall.Unset()
	}
}

func TestRevokeCert(t *testing.T) {
	svc, repo, _, auth, sdk := newService(t)
	cases := []struct {
		token       string
		desc        string
		thingID     string
		page        certs.Page
		identifyRes *magistrala.IdentityRes
		identifyErr error
		authErr     error
		thingErr    errors.SDKError
		repoErr     error
		err         error
	}{
		{
			desc:        "revoke cert",
			token:       token,
			thingID:     thingID,
			page:        certs.Page{Limit: 10000, Offset: 0, Total: 1, Certs: []certs.Cert{cert}},
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:        "revoke cert for invalid token",
			token:       invalid,
			thingID:     thingID,
			page:        certs.Page{},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "revoke cert for invalid thing id",
			token:       token,
			thingID:     "2",
			page:        certs.Page{},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			thingErr:    errors.NewSDKError(certs.ErrFailedCertCreation),
			err:         certs.ErrFailedCertRevocation,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, tc.authErr)
		sdkCall := sdk.On("Thing", tc.thingID, tc.token).Return(mgsdk.Thing{ID: tc.thingID, Credentials: mgsdk.Credentials{Secret: thingKey}}, tc.thingErr)
		repoCall := repo.On("RetrieveByThing", context.Background(), validID, tc.thingID, tc.page.Offset, tc.page.Limit).Return(certs.Page{}, tc.repoErr)

		_, err := svc.RevokeCert(context.Background(), tc.token, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		authCall1.Unset()
		sdkCall.Unset()
		repoCall.Unset()
	}
}

func TestListCerts(t *testing.T) {
	svc, repo, agent, auth, _ := newService(t)
	var mycerts []certs.Cert
	for i := 0; i < certNum; i++ {
		c := certs.Cert{
			OwnerID: validID,
			ThingID: thingID,
			Serial:  fmt.Sprintf("%d", i),
			Expire:  time.Now().Add(time.Hour),
		}
		mycerts = append(mycerts, c)
	}

	for i := 0; i < certNum; i++ {
		agent.On("Read", fmt.Sprintf("%d", i)).Return(pki.Cert{}, nil)
	}

	cases := []struct {
		token       string
		desc        string
		thingID     string
		page        certs.Page
		cert        certs.Cert
		identifyRes *magistrala.IdentityRes
		identifyErr error
		repoErr     error
		err         error
	}{
		{
			desc:    "list all certs with valid token",
			token:   token,
			thingID: thingID,
			page:    certs.Page{Limit: certNum, Offset: 0, Total: certNum, Certs: mycerts},
			cert: certs.Cert{
				OwnerID: validID,
				ThingID: thingID,
				Serial:  "0",
				Expire:  time.Now().Add(time.Hour),
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:    "list all certs with invalid token",
			token:   invalid,
			thingID: thingID,
			page:    certs.Page{},
			cert: certs.Cert{
				OwnerID: validID,
				ThingID: thingID,
				Serial:  fmt.Sprintf("%d", certNum-1),
				Expire:  time.Now().Add(time.Hour),
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "list half certs with valid token",
			token:   token,
			thingID: thingID,
			page:    certs.Page{Limit: certNum, Offset: certNum / 2, Total: certNum / 2, Certs: mycerts[certNum/2:]},
			cert: certs.Cert{
				OwnerID: validID,
				ThingID: thingID,
				Serial:  fmt.Sprintf("%d", certNum/2),
				Expire:  time.Now().Add(time.Hour),
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:    "list last cert with valid token",
			token:   token,
			thingID: thingID,
			page:    certs.Page{Limit: certNum, Offset: certNum - 1, Total: 1, Certs: []certs.Cert{mycerts[certNum-1]}},
			cert: certs.Cert{
				OwnerID: validID,
				ThingID: thingID,
				Serial:  fmt.Sprintf("%d", certNum-1),
				Expire:  time.Now().Add(time.Hour),
			},
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		repoCall := repo.On("RetrieveByThing", context.Background(), validID, thingID, tc.page.Offset, tc.page.Limit).Return(tc.page, tc.repoErr)

		page, err := svc.ListCerts(context.Background(), tc.token, tc.thingID, tc.page.Offset, tc.page.Limit)
		size := uint64(len(page.Certs))
		assert.Equal(t, tc.page.Total, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Total, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestListSerials(t *testing.T) {
	svc, repo, _, auth, _ := newService(t)

	var issuedCerts []certs.Cert
	for i := 0; i < certNum; i++ {
		crt := certs.Cert{
			OwnerID: cert.OwnerID,
			ThingID: cert.ThingID,
			Serial:  cert.Serial,
			Expire:  cert.Expire,
		}
		issuedCerts = append(issuedCerts, crt)
	}

	cases := []struct {
		token       string
		desc        string
		thingID     string
		offset      uint64
		limit       uint64
		certs       []certs.Cert
		identifyRes *magistrala.IdentityRes
		identifyErr error
		repoErr     error
		err         error
	}{
		{
			desc:        "list all certs with valid token",
			token:       token,
			thingID:     thingID,
			offset:      0,
			limit:       certNum,
			certs:       issuedCerts,
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:        "list all certs with invalid token",
			token:       invalid,
			thingID:     thingID,
			offset:      0,
			limit:       certNum,
			certs:       nil,
			identifyRes: &magistrala.IdentityRes{Id: validID},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "list half certs with valid token",
			token:       token,
			thingID:     thingID,
			offset:      certNum / 2,
			limit:       certNum,
			certs:       issuedCerts[certNum/2:],
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:        "list last cert with valid token",
			token:       token,
			thingID:     thingID,
			offset:      certNum - 1,
			limit:       certNum,
			certs:       []certs.Cert{issuedCerts[certNum-1]},
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		repoCall := repo.On("RetrieveByThing", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(certs.Page{Limit: tc.limit, Offset: tc.offset, Total: certNum, Certs: tc.certs}, tc.repoErr)

		page, err := svc.ListSerials(context.Background(), tc.token, tc.thingID, tc.offset, tc.limit)
		assert.Equal(t, tc.certs, page.Certs, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.certs, page.Certs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestViewCert(t *testing.T) {
	svc, repo, agent, auth, sdk := newService(t)

	authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	sdkCall := sdk.On("Thing", thingID, token).Return(mgsdk.Thing{ID: thingID, Credentials: mgsdk.Credentials{Secret: thingKey}}, nil)
	agentCall := agent.On("IssueCert", thingKey, ttl).Return(pki.Cert{}, nil)
	repoCall := repo.On("Save", context.Background(), mock.Anything).Return("", nil)

	ic, err := svc.IssueCert(context.Background(), token, thingID, ttl)
	require.Nil(t, err, fmt.Sprintf("unexpected cert creation error: %s\n", err))
	authCall.Unset()
	sdkCall.Unset()
	agentCall.Unset()
	repoCall.Unset()

	cert := certs.Cert{
		ThingID:    thingID,
		ClientCert: ic.ClientCert,
		Serial:     ic.Serial,
		Expire:     ic.Expire,
	}

	cases := []struct {
		token       string
		desc        string
		serialID    string
		cert        certs.Cert
		identifyRes *magistrala.IdentityRes
		identifyErr error
		repoErr     error
		agentErr    error
		err         error
	}{
		{
			desc:        "list cert with valid token and serial",
			token:       token,
			serialID:    cert.Serial,
			cert:        cert,
			identifyRes: &magistrala.IdentityRes{Id: validID},
		},
		{
			desc:        "list cert with invalid token",
			token:       invalid,
			serialID:    cert.Serial,
			cert:        certs.Cert{},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "list cert with invalid serial",
			token:       token,
			serialID:    invalid,
			cert:        certs.Cert{},
			identifyRes: &magistrala.IdentityRes{Id: validID},
			repoErr:     repoerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		repoCall := repo.On("RetrieveBySerial", context.Background(), validID, tc.serialID).Return(tc.cert, tc.repoErr)
		agentCall := agent.On("Read", tc.serialID).Return(pki.Cert{}, tc.agentErr)

		cert, err := svc.ViewCert(context.Background(), tc.token, tc.serialID)
		assert.Equal(t, tc.cert, cert, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.cert, cert))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
		agentCall.Unset()
	}
}
