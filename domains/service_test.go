// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/domains/mocks"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
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
	ErrExpiry             = errors.New("session is expired")
	errRollbackPolicy     = errors.New("failed to rollback policy")
	errAddPolicies        = errors.New("failed to add policies")
	errPlatform           = errors.New("invalid platform id")
	inValid               = "invalid"
	valid                 = "valid"
	domain                = domains.Domain{
		ID:         validID,
		Name:       groupName,
		Tags:       []string{"tag1", "tag2"},
		Alias:      "test",
		Permission: policies.AdminPermission,
		CreatedBy:  validID,
		UpdatedBy:  validID,
	}
	validSession   = authn.Session{}
	inValidSession = authn.Session{}
)

var (
	drepo      *mocks.Repository
	policyMock *policiesMocks.Service
)

func newService() domains.Service {
	drepo = new(mocks.Repository)
	idProvider := uuid.NewMock()
	sidProvider := sid.NewMock()
	policyMock = new(policiesMocks.Service)
	ds, _ := domains.New(drepo, policyMock, idProvider, sidProvider)
	return ds
}

func TestCreateDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc              string
		d                 domains.Domain
		session           authn.Session
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
			session: validSession,
			err:     nil,
		},
		{
			desc: "create domain with invalid session",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session: inValidSession,
			err:     svcerr.ErrAuthentication,
		},
		{
			desc: "create domain with invalid status",
			d: domains.Domain{
				Status: domains.AllStatus,
			},
			session: validSession,
			err:     svcerr.ErrInvalidStatus,
		},
		{
			desc: "create domain with failed policy request",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session:      validSession,
			addPolicyErr: errors.ErrMalformedEntity,
			err:          errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with failed save policy request",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session:       validSession,
			savePolicyErr: errors.ErrMalformedEntity,
			err:           errCreateDomainPolicy,
		},
		{
			desc: "create domain with failed save domain request",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session:       validSession,
			saveDomainErr: errors.ErrMalformedEntity,
			err:           svcerr.ErrCreateEntity,
		},
		{
			desc: "create domain with rollback error",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session:         validSession,
			savePolicyErr:   errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with rollback error and failed to delete policies",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session:           validSession,
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
			session:           validSession,
			saveDomainErr:     errors.ErrMalformedEntity,
			deletePoliciesErr: errors.ErrMalformedEntity,
			err:               errRollbackPolicy,
		},
		{
			desc: "create domain with failed to create and failed rollback",
			d: domains.Domain{
				Status: domains.EnabledStatus,
			},
			session:         validSession,
			saveDomainErr:   errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("Save", mock.Anything, mock.Anything).Return(domains.Domain{}, tc.saveDomainErr)
		_, err := svc.CreateDomain(context.Background(), tc.session, tc.d)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRetrieveDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc           string
		session        authn.Session
		domainID       string
		domainRepoErr  error
		domainRepoErr1 error
		checkPolicyErr error
		err            error
	}{
		{
			desc:     "retrieve domain successfully",
			session:  validSession,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "retrieve domain with invalid session",
			session:  inValidSession,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:           "retrieve domain with empty domain id",
			session:        validSession,
			domainID:       "",
			err:            svcerr.ErrViewEntity,
			domainRepoErr1: repoerr.ErrNotFound,
		},
		{
			desc:           "retrieve non-existing domain",
			session:        validSession,
			domainID:       inValid,
			domainRepoErr:  repoerr.ErrNotFound,
			err:            svcerr.ErrViewEntity,
			domainRepoErr1: repoerr.ErrNotFound,
		},
		{
			desc:           "retrieve domain with failed to retrieve by id",
			session:        validSession,
			domainID:       validID,
			domainRepoErr1: repoerr.ErrNotFound,
			err:            svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, groupName).Return(domains.Domain{}, tc.domainRepoErr)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, tc.domainID).Return(domains.Domain{}, tc.domainRepoErr1)
		_, err := svc.RetrieveDomain(context.Background(), tc.session, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc            string
		session         authn.Session
		domainID        string
		domReq          domains.DomainReq
		checkPolicyErr  error
		retrieveByIDErr error
		updateErr       error
		err             error
	}{
		{
			desc:     "update domain successfully",
			session:  validSession,
			domainID: validID,
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			err: nil,
		},
		{
			desc:     "update domain with invalid session",
			session:  inValidSession,
			domainID: validID,
			domReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "update domain with empty domainID",
			session:  validSession,
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
			session:  validSession,
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
			session:  validSession,
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
		_, err := svc.UpdateDomain(context.Background(), tc.session, tc.domainID, tc.domReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListDomains(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc            string
		session         authn.Session
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
			session:  validSession,
			domainID: validID,
			authReq: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			listDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
			},
			err: nil,
		},
		{
			desc:     "list domains with invalid session",
			session:  inValidSession,
			domainID: validID,
			authReq: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "list domains with repository error on list domains",
			session:  validSession,
			domainID: validID,
			authReq: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			listDomainErr: errors.ErrMalformedEntity,
			err:           svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall1 := drepo.On("ListDomains", mock.Anything, mock.Anything).Return(tc.listDomainsRes, tc.listDomainErr)
		_, err := svc.ListDomains(context.Background(), tc.session, domains.Page{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall1.Unset()
	}
}
