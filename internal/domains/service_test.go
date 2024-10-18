// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	domainSvc "github.com/absmach/magistrala/internal/domains"
	"github.com/absmach/magistrala/pkg/domains"
	"github.com/absmach/magistrala/pkg/domains/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policiesMocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/sid"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	secret      = "secret"
	email       = "test@example.com"
	id          = "testID"
	groupName   = "mgx"
	description = "Description"

	memberRelation  = "member"
	authoritiesObj  = "authorities"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	errIssueUser          = errors.New("failed to issue new login key")
	errCreateDomainPolicy = errors.New("failed to create domain policy")
	errRetrieve           = errors.New("failed to retrieve key data")
	ErrExpiry             = errors.New("token is expired")
	errRollbackPolicy     = errors.New("failed to rollback policy")
	errAddPolicies        = errors.New("failed to add policies")
	errPlatform           = errors.New("invalid platform id")
	inValidToken          = "invalid"
	inValid               = "invalid"
	valid                 = "valid"
	domain                = domains.Domain{
		ID:         validID,
		Name:       groupName,
		Tags:       []string{"tag1", "tag2"},
		Alias:      "test",
		Permission: auth.AdminPermission,
		CreatedBy:  validID,
		UpdatedBy:  validID,
	}
	accessToken = "accessToken"
)

var (
	drepo      *mocks.DomainsRepository
	policyMock *policiesMocks.Service
)

func newService() domains.Service {
	drepo = new(mocks.DomainsRepository)
	idProvider := uuid.NewMock()
	sidProvider := sid.NewMock()
	policyMock = new(policiesMocks.Service)
	ds, _ := domainSvc.New(drepo, policyMock, idProvider, sidProvider)
	return ds
}

func TestCreateDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc              string
		d                 domains.Domain
		token             string
		userID            string
		addPolicyErr      error
		savePolicyErr     error
		saveDomainErr     error
		deleteDomainErr   error
		deletePoliciesErr error
		err               error
	}{
		{
			desc: "create domain successfully",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token: accessToken,
			err:   nil,
		},
		{
			desc: "create domain with invalid token",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token: inValidToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "create domain with invalid status",
			d: domains.Domain{
				Status: domains.AllStatus,
			},
			token: accessToken,
			err:   svcerr.ErrInvalidStatus,
		},
		{
			desc: "create domain with failed policy request",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:        accessToken,
			addPolicyErr: errors.ErrMalformedEntity,
			err:          errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with failed save policy request",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:         accessToken,
			savePolicyErr: errors.ErrMalformedEntity,
			err:           errCreateDomainPolicy,
		},
		{
			desc: "create domain with failed save domain request",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:         accessToken,
			saveDomainErr: errors.ErrMalformedEntity,
			err:           svcerr.ErrCreateEntity,
		},
		{
			desc: "create domain with rollback error",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:           accessToken,
			savePolicyErr:   errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with rollback error and failed to delete policies",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:             accessToken,
			savePolicyErr:     errors.ErrMalformedEntity,
			deleteDomainErr:   errors.ErrMalformedEntity,
			deletePoliciesErr: errors.ErrMalformedEntity,
			err:               errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with failed to create and failed rollback",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:             accessToken,
			saveDomainErr:     errors.ErrMalformedEntity,
			deletePoliciesErr: errors.ErrMalformedEntity,
			err:               errRollbackPolicy,
		},
		{
			desc: "create domain with failed to create and failed rollback",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			token:           accessToken,
			saveDomainErr:   errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("Save", mock.Anything, mock.Anything).Return(domains.Domain{}, tc.saveDomainErr)
		_, err := svc.CreateDomain(context.Background(), tc.token, tc.d)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRetrieveDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc           string
		token          string
		domainID       string
		domainRepoErr  error
		domainRepoErr1 error
		checkPolicyErr error
		err            error
	}{
		{
			desc:     "retrieve domain successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "retrieve domain with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:           "retrieve domain with empty domain id",
			token:          accessToken,
			domainID:       "",
			err:            svcerr.ErrViewEntity,
			domainRepoErr1: repoerr.ErrNotFound,
		},
		{
			desc:           "retrieve non-existing domain",
			token:          accessToken,
			domainID:       inValid,
			domainRepoErr:  repoerr.ErrNotFound,
			err:            svcerr.ErrViewEntity,
			domainRepoErr1: repoerr.ErrNotFound,
		},
		{
			desc:           "retrieve domain with failed to retrieve by id",
			token:          accessToken,
			domainID:       validID,
			domainRepoErr1: repoerr.ErrNotFound,
			err:            svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, groupName).Return(domains.Domain{}, tc.domainRepoErr)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, tc.domainID).Return(domains.Domain{}, tc.domainRepoErr1)
		_, err := svc.RetrieveDomain(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc            string
		token           string
		domainID        string
		domReq          domains.DomainReq
		checkPolicyErr  error
		retrieveByIDErr error
		updateErr       error
		err             error
	}{
		{
			desc:     "update domain successfully",
			token:    accessToken,
			domainID: validID,
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			err: nil,
		},
		{
			desc:     "update domain with invalid token",
			token:    inValidToken,
			domainID: validID,
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "update domain with empty domainID",
			token:    accessToken,
			domainID: "",
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			checkPolicyErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrDomainAuthorization,
		},
		{
			desc:     "update domain with failed to retrieve by id",
			token:    accessToken,
			domainID: validID,
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrNotFound,
		},
		{
			desc:     "update domain with failed to update",
			token:    accessToken,
			domainID: validID,
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			updateErr: errors.ErrMalformedEntity,
			err:       errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(domains.Domain{}, tc.retrieveByIDErr)
		repoCall2 := drepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(domains.Domain{}, tc.updateErr)
		_, err := svc.UpdateDomain(context.Background(), tc.token, tc.domainID, tc.domReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListDomains(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc            string
		token           string
		domainID        string
		authReq         domains.Page
		listDomainsRes  domains.DomainsPage
		retreiveByIDErr error
		checkPolicyErr  error
		listDomainErr   error
		err             error
	}{
		{
			desc:     "list domains successfully",
			token:    accessToken,
			domainID: validID,
			authReq: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			listDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
			},
			err: nil,
		},
		{
			desc:     "list domains with invalid token",
			token:    inValidToken,
			domainID: validID,
			authReq: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "list domains with repository error on list domains",
			token:    accessToken,
			domainID: validID,
			authReq: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			listDomainErr: errors.ErrMalformedEntity,
			err:           svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := drepo.On("ListDomains", mock.Anything, mock.Anything).Return(tc.listDomainsRes, tc.listDomainErr)
		_, err := svc.ListDomains(context.Background(), tc.token, domains.Page{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall1.Unset()
	}
}
