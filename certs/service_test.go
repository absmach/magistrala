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
	domain    = "domain"
	token     = "token"
	thingsNum = 1
	thingKey  = "thingKey"
	thingID   = "1"
	ttl       = "1h"
	certNum   = 10
	validID   = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

func newService(_ *testing.T) (certs.Service, *mocks.Agent, *sdkmocks.SDK) {
	agent := new(mocks.Agent)
	sdk := new(sdkmocks.SDK)

	return certs.New(sdk, agent), agent, sdk
}

var cert = mgcrt.Cert{
	ThingID:      thingID,
	SerialNumber: "Serial",
	ExpiryTime:   time.Now().Add(time.Duration(1000)),
	Revoked:      false,
}

func TestIssueCert(t *testing.T) {
	svc, agent, sdk := newService(t)
	cases := []struct {
		domainID     string
		token        string
		desc         string
		thingID      string
		ttl          string
		ipAddr       []string
		key          string
		cert         mgcrt.Cert
		thingErr     errors.SDKError
		issueCertErr error
		err          error
	}{
		{
			desc:     "issue new cert",
			domainID: domain,
			token:    token,
			thingID:  thingID,
			ttl:      ttl,
			ipAddr:   []string{},
			cert:     cert,
		},
		{
			desc:         "issue new for failed pki",
			domainID:     domain,
			token:        token,
			thingID:      thingID,
			ttl:          ttl,
			ipAddr:       []string{},
			thingErr:     nil,
			issueCertErr: certs.ErrFailedCertCreation,
			err:          certs.ErrFailedCertCreation,
		},
		{
			desc:     "issue new cert for non existing thing id",
			domainID: domain,
			token:    token,
			thingID:  "2",
			ttl:      ttl,
			ipAddr:   []string{},
			thingErr: errors.NewSDKError(errors.ErrMalformedEntity),
			err:      certs.ErrFailedCertCreation,
		},
		{
			desc:     "issue new cert for invalid token",
			domainID: domain,
			token:    invalid,
			thingID:  thingID,
			ttl:      ttl,
			ipAddr:   []string{},
			thingErr: errors.NewSDKError(svcerr.ErrAuthentication),
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdk.On("Thing", tc.thingID, tc.domainID, tc.token).Return(mgsdk.Thing{ID: tc.thingID, Credentials: mgsdk.ClientCredentials{Secret: thingKey}}, tc.thingErr)
			agentCall := agent.On("Issue", thingID, tc.ttl, tc.ipAddr).Return(tc.cert, tc.issueCertErr)
			resp, err := svc.IssueCert(context.Background(), tc.domainID, tc.token, tc.thingID, tc.ttl)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.cert.SerialNumber, resp.SerialNumber, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.cert.SerialNumber, resp.SerialNumber))
			sdkCall.Unset()
			agentCall.Unset()
		})
	}
}

func TestRevokeCert(t *testing.T) {
	svc, agent, sdk := newService(t)
	cases := []struct {
		domainID  string
		token     string
		desc      string
		thingID   string
		page      mgcrt.CertPage
		authErr   error
		thingErr  errors.SDKError
		revokeErr error
		listErr   error
		err       error
	}{
		{
			desc:     "revoke cert",
			domainID: domain,
			token:    token,
			thingID:  thingID,
			page:     mgcrt.CertPage{Limit: 10000, Offset: 0, Total: 1, Certificates: []mgcrt.Cert{cert}},
		},
		{
			desc:      "revoke cert for failed pki revoke",
			domainID:  domain,
			token:     token,
			thingID:   thingID,
			page:      mgcrt.CertPage{Limit: 10000, Offset: 0, Total: 1, Certificates: []mgcrt.Cert{cert}},
			revokeErr: certs.ErrFailedCertRevocation,
			err:       certs.ErrFailedCertRevocation,
		},
		{
			desc:     "revoke cert for invalid thing id",
			domainID: domain,
			token:    token,
			thingID:  "2",
			page:     mgcrt.CertPage{},
			thingErr: errors.NewSDKError(certs.ErrFailedCertCreation),
			err:      certs.ErrFailedCertRevocation,
		},
		{
			desc:     "revoke cert with failed to list certs",
			domainID: domain,
			token:    token,
			thingID:  thingID,
			page:     mgcrt.CertPage{},
			listErr:  certs.ErrFailedCertRevocation,
			err:      certs.ErrFailedCertRevocation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdk.On("Thing", tc.thingID, tc.domainID, tc.token).Return(mgsdk.Thing{ID: tc.thingID, Credentials: mgsdk.ClientCredentials{Secret: thingKey}}, tc.thingErr)
			agentCall := agent.On("Revoke", mock.Anything).Return(tc.revokeErr)
			agentCall1 := agent.On("ListCerts", mock.Anything).Return(tc.page, tc.listErr)
			_, err := svc.RevokeCert(context.Background(), tc.domainID, tc.token, tc.thingID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			sdkCall.Unset()
			agentCall.Unset()
			agentCall1.Unset()
		})
	}
}

func TestListCerts(t *testing.T) {
	svc, agent, _ := newService(t)
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
		desc    string
		thingID string
		page    mgcrt.CertPage
		listErr error
		err     error
	}{
		{
			desc:    "list all certs successfully",
			thingID: thingID,
			page:    mgcrt.CertPage{Limit: certNum, Offset: 0, Total: certNum, Certificates: mycerts},
		},
		{
			desc:    "list all certs with failed pki",
			thingID: thingID,
			page:    mgcrt.CertPage{},
			listErr: svcerr.ErrViewEntity,
			err:     svcerr.ErrViewEntity,
		},
		{
			desc:    "list half certs successfully",
			thingID: thingID,
			page:    mgcrt.CertPage{Limit: certNum, Offset: certNum / 2, Total: certNum / 2, Certificates: mycerts[certNum/2:]},
		},
		{
			desc:    "list last cert successfully",
			thingID: thingID,
			page:    mgcrt.CertPage{Limit: certNum, Offset: certNum - 1, Total: 1, Certificates: []mgcrt.Cert{mycerts[certNum-1]}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("ListCerts", mock.Anything).Return(tc.page, tc.listErr)
			page, err := svc.ListCerts(context.Background(), tc.thingID, certs.PageMetadata{Offset: tc.page.Offset, Limit: tc.page.Limit})
			size := uint64(len(page.Certificates))
			assert.Equal(t, tc.page.Total, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Total, size))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			agentCall.Unset()
		})
	}
}

func TestListSerials(t *testing.T) {
	svc, agent, _ := newService(t)
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
		desc    string
		thingID string
		revoke  string
		offset  uint64
		limit   uint64
		certs   []mgcrt.Cert
		listErr error
		err     error
	}{
		{
			desc:    "list all certs successfully",
			thingID: thingID,
			revoke:  revoke,
			offset:  0,
			limit:   certNum,
			certs:   issuedCerts,
		},
		{
			desc:    "list all certs with failed pki",
			thingID: thingID,
			revoke:  revoke,
			offset:  0,
			limit:   certNum,
			certs:   nil,
			listErr: svcerr.ErrViewEntity,
			err:     svcerr.ErrViewEntity,
		},
		{
			desc:    "list half certs successfully",
			thingID: thingID,
			revoke:  revoke,
			offset:  certNum / 2,
			limit:   certNum,
			certs:   issuedCerts[certNum/2:],
		},
		{
			desc:    "list last cert successfully",
			thingID: thingID,
			revoke:  revoke,
			offset:  certNum - 1,
			limit:   certNum,
			certs:   []mgcrt.Cert{issuedCerts[certNum-1]},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("ListCerts", mock.Anything).Return(mgcrt.CertPage{Certificates: tc.certs}, tc.listErr)
			page, err := svc.ListSerials(context.Background(), tc.thingID, certs.PageMetadata{Revoked: tc.revoke, Offset: tc.offset, Limit: tc.limit})
			assert.Equal(t, len(tc.certs), len(page.Certificates), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.certs, page.Certificates))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			agentCall.Unset()
		})
	}
}

func TestViewCert(t *testing.T) {
	svc, agent, _ := newService(t)

	cases := []struct {
		desc     string
		serialID string
		cert     mgcrt.Cert
		repoErr  error
		agentErr error
		err      error
	}{
		{
			desc:     "view cert with valid serial",
			serialID: cert.SerialNumber,
			cert:     cert,
		},
		{
			desc:     "list cert with invalid serial",
			serialID: invalid,
			cert:     mgcrt.Cert{},
			agentErr: svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("View", tc.serialID).Return(tc.cert, tc.agentErr)
			res, err := svc.ViewCert(context.Background(), tc.serialID)
			assert.Equal(t, tc.cert.SerialNumber, res.SerialNumber, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.cert.SerialNumber, res.SerialNumber))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			agentCall.Unset()
		})
	}
}
