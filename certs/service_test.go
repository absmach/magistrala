// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/certs/mocks"
	mgcrt "github.com/absmach/magistrala/certs/pki/amcerts"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func newService(_ *testing.T) (certs.Service, *mocks.Agent, *authnmocks.Authentication, *sdkmocks.SDK) {
	agent := new(mocks.Agent)
	authn := new(authnmocks.Authentication)
	sdk := new(sdkmocks.SDK)

	return certs.New(authn, sdk, agent), agent, authn, sdk
}

var cert = mgcrt.Cert{
	ThingID:      thingID,
	SerialNumber: "Serial",
	ExpiryTime:   time.Now().Add(time.Duration(1000)),
	Revoked:      false,
}

func TestIssueCert(t *testing.T) {
	svc, agent, auth, sdk := newService(t)
	cases := []struct {
		token           string
		desc            string
		thingID         string
		ttl             string
		ipAddr          []string
		key             string
		cert            mgcrt.Cert
		authenticateRes mgauthn.Session
		authenticateErr error
		thingErr        errors.SDKError
		issueCertErr    error
		err             error
	}{
		{
			desc:            "issue new cert",
			token:           token,
			thingID:         thingID,
			ttl:             ttl,
			ipAddr:          []string{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			cert:            cert,
		},
		{
			desc:            "issue new for failed pki",
			token:           token,
			thingID:         thingID,
			ttl:             ttl,
			ipAddr:          []string{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			thingErr:        nil,
			issueCertErr:    certs.ErrFailedCertCreation,
			err:             certs.ErrFailedCertCreation,
		},
		{
			desc:            "issue new cert for non existing thing id",
			token:           token,
			thingID:         "2",
			ttl:             ttl,
			ipAddr:          []string{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			thingErr:        errors.NewSDKError(errors.ErrMalformedEntity),
			err:             certs.ErrFailedCertCreation,
		},
		{
			desc:            "issue new cert for invalid token",
			token:           invalid,
			thingID:         thingID,
			ttl:             ttl,
			ipAddr:          []string{},
			authenticateRes: mgauthn.Session{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", context.Background(), tc.token).Return(tc.authenticateRes, tc.authenticateErr)
			sdkCall := sdk.On("Thing", tc.thingID, tc.token).Return(mgsdk.Thing{ID: tc.thingID, Credentials: mgsdk.Credentials{Secret: thingKey}}, tc.thingErr)
			agentCall := agent.On("Issue", thingID, tc.ttl, tc.ipAddr).Return(tc.cert, tc.issueCertErr)

			resp, err := svc.IssueCert(context.Background(), tc.token, tc.thingID, tc.ttl)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.cert.SerialNumber, resp.SerialNumber, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.cert.SerialNumber, resp.SerialNumber))
			authCall.Unset()
			sdkCall.Unset()
			agentCall.Unset()
		})
	}
}

func TestRevokeCert(t *testing.T) {
	svc, agent, auth, sdk := newService(t)
	cases := []struct {
		token           string
		desc            string
		thingID         string
		page            mgcrt.CertPage
		authenticateRes mgauthn.Session
		authenticateErr error
		authErr         error
		thingErr        errors.SDKError
		revokeErr       error
		listErr         error
		err             error
	}{
		{
			desc:            "revoke cert",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{Limit: 10000, Offset: 0, Total: 1, Certificates: []mgcrt.Cert{cert}},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
		{
			desc:            "revoke cert for failed pki revoke",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{Limit: 10000, Offset: 0, Total: 1, Certificates: []mgcrt.Cert{cert}},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			revokeErr:       certs.ErrFailedCertRevocation,
			err:             certs.ErrFailedCertRevocation,
		},
		{
			desc:            "revoke cert for invalid token",
			token:           invalid,
			thingID:         thingID,
			page:            mgcrt.CertPage{},
			authenticateRes: mgauthn.Session{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:            "revoke cert for invalid thing id",
			token:           token,
			thingID:         "2",
			page:            mgcrt.CertPage{},
			authenticateRes: mgauthn.Session{},
			thingErr:        errors.NewSDKError(certs.ErrFailedCertCreation),
			err:             certs.ErrFailedCertRevocation,
		},
		{
			desc:            "revoke cert with failed to list certs",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			listErr:         certs.ErrFailedCertRevocation,
			err:             certs.ErrFailedCertRevocation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", context.Background(), tc.token).Return(tc.authenticateRes, tc.authenticateErr)
			sdkCall := sdk.On("Thing", tc.thingID, tc.token).Return(mgsdk.Thing{ID: tc.thingID, Credentials: mgsdk.Credentials{Secret: thingKey}}, tc.thingErr)
			agentCall := agent.On("Revoke", mock.Anything).Return(tc.revokeErr)
			agentCall1 := agent.On("ListCerts", mock.Anything).Return(tc.page, tc.listErr)
			_, err := svc.RevokeCert(context.Background(), tc.token, tc.thingID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			authCall.Unset()
			sdkCall.Unset()
			agentCall.Unset()
			agentCall1.Unset()
		})
	}
}

func TestListCerts(t *testing.T) {
	svc, agent, auth, _ := newService(t)
	var mycerts []mgcrt.Cert
	for i := 0; i < certNum; i++ {
		c := mgcrt.Cert{
			ThingID:      thingID,
			SerialNumber: fmt.Sprintf("%d", i),
			ExpiryTime:   time.Now().Add(time.Hour),
		}
		mycerts = append(mycerts, c)
	}

	cases := []struct {
		token           string
		desc            string
		thingID         string
		page            mgcrt.CertPage
		authenticateRes mgauthn.Session
		authenticateErr error
		listErr         error
		err             error
	}{
		{
			desc:            "list all certs with valid token",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{Limit: certNum, Offset: 0, Total: certNum, Certificates: mycerts},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
		{
			desc:            "list all certs with invalid token",
			token:           invalid,
			thingID:         thingID,
			page:            mgcrt.CertPage{},
			authenticateRes: mgauthn.Session{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:            "list all certs with failed pki",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			listErr:         svcerr.ErrViewEntity,
			err:             svcerr.ErrViewEntity,
		},
		{
			desc:            "list half certs with valid token",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{Limit: certNum, Offset: certNum / 2, Total: certNum / 2, Certificates: mycerts[certNum/2:]},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
		{
			desc:            "list last cert with valid token",
			token:           token,
			thingID:         thingID,
			page:            mgcrt.CertPage{Limit: certNum, Offset: certNum - 1, Total: 1, Certificates: []mgcrt.Cert{mycerts[certNum-1]}},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", context.Background(), tc.token).Return(tc.authenticateRes, tc.authenticateErr)
			agentCall := agent.On("ListCerts", mock.Anything).Return(tc.page, tc.listErr)

			page, err := svc.ListCerts(context.Background(), tc.token, tc.thingID, certs.PageMetadata{Offset: tc.page.Offset, Limit: tc.page.Limit})
			size := uint64(len(page.Certificates))
			assert.Equal(t, tc.page.Total, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Total, size))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			authCall.Unset()
			agentCall.Unset()
		})
	}
}

func TestListSerials(t *testing.T) {
	svc, agent, auth, _ := newService(t)
	revoke := "false"

	var issuedCerts []mgcrt.Cert
	for i := 0; i < certNum; i++ {
		crt := mgcrt.Cert{
			ThingID:      cert.ThingID,
			SerialNumber: cert.SerialNumber,
			ExpiryTime:   cert.ExpiryTime,
			Revoked:      false,
		}
		issuedCerts = append(issuedCerts, crt)
	}

	cases := []struct {
		token           string
		desc            string
		thingID         string
		revoke          string
		offset          uint64
		limit           uint64
		certs           []mgcrt.Cert
		authenticateRes mgauthn.Session
		authenticateErr error
		listErr         error
		err             error
	}{
		{
			desc:            "list all certs with valid token",
			token:           token,
			thingID:         thingID,
			revoke:          revoke,
			offset:          0,
			limit:           certNum,
			certs:           issuedCerts,
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
		{
			desc:            "list all certs with invalid token",
			token:           invalid,
			thingID:         thingID,
			revoke:          revoke,
			offset:          0,
			limit:           certNum,
			certs:           nil,
			authenticateRes: mgauthn.Session{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:            "list all certs with failed pki",
			token:           token,
			thingID:         thingID,
			revoke:          revoke,
			offset:          0,
			limit:           certNum,
			certs:           nil,
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			listErr:         svcerr.ErrViewEntity,
			err:             svcerr.ErrViewEntity,
		},
		{
			desc:            "list half certs with valid token",
			token:           token,
			thingID:         thingID,
			revoke:          revoke,
			offset:          certNum / 2,
			limit:           certNum,
			certs:           issuedCerts[certNum/2:],
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
		{
			desc:            "list last cert with valid token",
			token:           token,
			thingID:         thingID,
			revoke:          revoke,
			offset:          certNum - 1,
			limit:           certNum,
			certs:           []mgcrt.Cert{issuedCerts[certNum-1]},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", context.Background(), tc.token).Return(tc.authenticateRes, tc.authenticateErr)
			agentCall := agent.On("ListCerts", mock.Anything).Return(mgcrt.CertPage{Certificates: tc.certs}, tc.listErr)
			page, err := svc.ListSerials(context.Background(), tc.token, tc.thingID, certs.PageMetadata{Revoked: tc.revoke, Offset: tc.offset, Limit: tc.limit})
			assert.Equal(t, len(tc.certs), len(page.Certificates), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.certs, page.Certificates))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			authCall.Unset()
			agentCall.Unset()
		})
	}
}

func TestViewCert(t *testing.T) {
	svc, agent, auth, _ := newService(t)

	cases := []struct {
		token           string
		desc            string
		serialID        string
		cert            mgcrt.Cert
		authenticateRes mgauthn.Session
		authenticateErr error
		repoErr         error
		agentErr        error
		err             error
	}{
		{
			desc:            "view cert with valid token and serial",
			token:           token,
			serialID:        cert.SerialNumber,
			cert:            cert,
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
		},
		{
			desc:            "list cert with invalid token",
			token:           invalid,
			serialID:        cert.SerialNumber,
			cert:            mgcrt.Cert{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:            "list cert with invalid serial",
			token:           token,
			serialID:        invalid,
			cert:            mgcrt.Cert{},
			authenticateRes: mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			agentErr:        svcerr.ErrNotFound,
			err:             svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", context.Background(), tc.token).Return(tc.authenticateRes, tc.authenticateErr)
			agentCall := agent.On("View", tc.serialID).Return(tc.cert, tc.agentErr)

			res, err := svc.ViewCert(context.Background(), tc.token, tc.serialID)
			assert.Equal(t, tc.cert.SerialNumber, res.SerialNumber, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.cert.SerialNumber, res.SerialNumber))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			authCall.Unset()
			agentCall.Unset()
		})
	}
}
