// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"context"
	"testing"
	"time"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/domains/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	policiesMocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/sid"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	secret          = "secret"
	email           = "test@example.com"
	id              = "testID"
	groupName       = "mgx"
	description     = "Description"
	memberRelation  = "member"
	authoritiesObj  = "authorities"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	ErrExpiry       = errors.New("session is expired")
	errAddPolicies  = errors.New("failed to add policies")
	errRollbackRepo = errors.New("failed to rollback repo")
	inValid         = "invalid"
	valid           = "valid"
	domain          = domains.Domain{
		ID:         validID,
		Name:       groupName,
		Tags:       []string{"tag1", "tag2"},
		Alias:      "test",
		Permission: policies.AdminPermission,
		CreatedBy:  validID,
		UpdatedBy:  validID,
	}
	userID       = testsutil.GenerateUUID(&testing.T{})
	validSession = authn.Session{UserID: userID}
)

var (
	drepo  *mocks.Repository
	policy *policiesMocks.Service
)

func newService() domains.Service {
	drepo = new(mocks.Repository)
	idProvider := uuid.NewMock()
	sidProvider := sid.NewMock()
	policy = new(policiesMocks.Service)
	ds, _ := domains.New(drepo, policy, idProvider, sidProvider)
	return ds
}

func TestCreateDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc              string
		d                 domains.Domain
		session           authn.Session
		userID            string
		addPoliciesErr    error
		addRolesErr       error
		saveDomainErr     error
		deleteDomainErr   error
		deletePoliciesErr error
		err               error
	}{
		{
			desc: "create domain successfully",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.EnabledStatus,
			},
			session: validSession,
			err:     nil,
		},
		{
			desc: "create domain with invalid status",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.AllStatus,
			},
			session: validSession,
			err:     svcerr.ErrInvalidStatus,
		},
		{
			desc: "create domain with failed to save domain",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.EnabledStatus,
			},
			session:       validSession,
			saveDomainErr: svcerr.ErrCreateEntity,
			err:           svcerr.ErrCreateEntity,
		},
		{
			desc: "create domain with failed to add policies",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.EnabledStatus,
			},
			session:        validSession,
			addPoliciesErr: errAddPolicies,
			err:            errAddPolicies,
		},
		{
			desc: "create domain with failed to add policies and failed rollback",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.EnabledStatus,
			},
			session:         validSession,
			addPoliciesErr:  errAddPolicies,
			deleteDomainErr: svcerr.ErrRemoveEntity,
			err:             errRollbackRepo,
		},
		{
			desc: "create domain with failed to add roles",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.EnabledStatus,
			},
			session:     validSession,
			addRolesErr: errors.ErrMalformedEntity,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with failed to add roles and failed rollback",
			d: domains.Domain{
				Name:   groupName,
				Status: domains.EnabledStatus,
			},
			session:         validSession,
			addRolesErr:     errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errRollbackRepo,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("Save", mock.Anything, mock.Anything).Return(tc.d, tc.saveDomainErr)
			repoCall1 := drepo.On("Delete", mock.Anything, mock.Anything).Return(tc.deleteDomainErr)
			repoCall2 := drepo.On("AddRoles", mock.Anything, mock.Anything).Return([]roles.Role{}, tc.addRolesErr)
			policyCall := policy.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesErr)
			policyCall1 := policy.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
			_, err := svc.CreateDomain(context.Background(), tc.session, tc.d)
			assert.True(t, errors.Contains(err, tc.err))
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			policyCall.Unset()
			policyCall1.Unset()
		})
	}
}

func TestRetrieveDomain(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc              string
		session           authn.Session
		domainID          string
		retrieveDomainRes domains.Domain
		retrieveDomainErr error
		err               error
	}{
		{
			desc:              "retrieve domain successfully",
			session:           validSession,
			domainID:          validID,
			retrieveDomainRes: domain,
			err:               nil,
		},
		{
			desc:              "retrieve domain with empty domain id",
			session:           validSession,
			domainID:          "",
			retrieveDomainErr: repoerr.ErrNotFound,
			err:               svcerr.ErrViewEntity,
		},
		{
			desc:              "retrieve non-existing domain",
			session:           validSession,
			domainID:          inValid,
			retrieveDomainErr: repoerr.ErrNotFound,
			err:               svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveByID", context.Background(), tc.domainID).Return(tc.retrieveDomainRes, tc.retrieveDomainErr)
			domain, err := svc.RetrieveDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.retrieveDomainRes, domain)
			repoCall.Unset()
		})
	}
}

func TestUpdateDomain(t *testing.T) {
	svc := newService()

	updatedDomain := domain
	updatedDomain.Name = valid
	updatedDomain.Alias = valid

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		updateReq domains.DomainReq
		updateRes domains.Domain
		updateErr error
		err       error
	}{
		{
			desc:     "update domain successfully",
			session:  validSession,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			updateRes: updatedDomain,
			err:       nil,
		},
		{
			desc:     "update domain with empty domainID",
			session:  validSession,
			domainID: "",
			updateReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			updateErr: repoerr.ErrNotFound,
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:     "update domain with failed to update",
			session:  validSession,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			updateErr: errors.ErrMalformedEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("Update", context.Background(), tc.domainID, tc.session.UserID, tc.updateReq).Return(tc.updateRes, tc.updateErr)
			domain, err := svc.UpdateDomain(context.Background(), tc.session, tc.domainID, tc.updateReq)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.updateRes, domain)
			repoCall.Unset()
		})
	}
}

func TestEnableDomain(t *testing.T) {
	svc := newService()

	enabledDomain := domain
	enabledDomain.Status = domains.EnabledStatus
	status := domains.EnabledStatus

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		enableRes domains.Domain
		enableErr error
		err       error
	}{
		{
			desc:      "enable domain successfully",
			session:   validSession,
			domainID:  domain.ID,
			enableRes: enabledDomain,
			err:       nil,
		},
		{
			desc:      "enable domain with empty domainID",
			session:   validSession,
			domainID:  "",
			enableErr: repoerr.ErrNotFound,
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:      "enable domain with failed to enable",
			session:   validSession,
			domainID:  domain.ID,
			enableErr: errors.ErrMalformedEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("Update", context.Background(), tc.domainID, tc.session.UserID, domains.DomainReq{Status: &status}).Return(tc.enableRes, tc.enableErr)
			domain, err := svc.EnableDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.enableRes, domain)
			repoCall.Unset()
		})
	}
}

func TestDisableDomain(t *testing.T) {
	svc := newService()

	disabledDomain := domain
	disabledDomain.Status = domains.DisabledStatus
	status := domains.DisabledStatus

	cases := []struct {
		desc       string
		session    authn.Session
		domainID   string
		disableRes domains.Domain
		disableErr error
		err        error
	}{
		{
			desc:       "disable domain successfully",
			session:    validSession,
			domainID:   domain.ID,
			disableRes: disabledDomain,
			err:        nil,
		},
		{
			desc:       "disable domain with empty domainID",
			session:    validSession,
			domainID:   "",
			disableErr: repoerr.ErrNotFound,
			err:        svcerr.ErrUpdateEntity,
		},
		{
			desc:       "disable domain with failed to disable",
			session:    validSession,
			domainID:   domain.ID,
			disableErr: errors.ErrMalformedEntity,
			err:        svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("Update", context.Background(), tc.domainID, tc.session.UserID, domains.DomainReq{Status: &status}).Return(tc.disableRes, tc.disableErr)
			domain, err := svc.DisableDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.disableRes, domain)
			repoCall.Unset()
		})
	}
}

func TestFreezeDomain(t *testing.T) {
	svc := newService()

	freezeDomain := domain
	freezeDomain.Status = domains.FreezeStatus
	status := domains.FreezeStatus

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		freezeRes domains.Domain
		freezeErr error
		err       error
	}{
		{
			desc:      "freeze domain successfully",
			session:   validSession,
			domainID:  domain.ID,
			freezeRes: freezeDomain,
			err:       nil,
		},
		{
			desc:      "freeze domain with empty domainID",
			session:   validSession,
			domainID:  "",
			freezeErr: repoerr.ErrNotFound,
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:      "freeze domain with failed to freeze",
			session:   validSession,
			domainID:  domain.ID,
			freezeErr: errors.ErrMalformedEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("Update", context.Background(), tc.domainID, tc.session.UserID, domains.DomainReq{Status: &status}).Return(tc.freezeRes, tc.freezeErr)
			domain, err := svc.FreezeDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.freezeRes, domain)
			repoCall.Unset()
		})
	}
}

func TestListDomains(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc           string
		session        authn.Session
		domainID       string
		pageMeta       domains.Page
		listDomainsRes domains.DomainsPage
		listDomainErr  error
		err            error
	}{
		{
			desc:     "list domains successfully",
			session:  validSession,
			domainID: validID,
			pageMeta: domains.Page{
				SubjectID:  userID,
				Offset:     0,
				Limit:      10,
				Permission: policies.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			listDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
				Offset:  0,
				Limit:   10,
				Total:   1,
			},
			err: nil,
		},
		{
			desc:     "list domains as admin successfully",
			session:  authn.Session{UserID: validID, SuperAdmin: true},
			domainID: validID,
			pageMeta: domains.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.AdminPermission,
				Status:     domains.EnabledStatus,
			},
			listDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
				Offset:  0,
				Limit:   10,
				Total:   1,
			},
			err: nil,
		},
		{
			desc:     "list domains with repository error on list domains",
			session:  validSession,
			domainID: validID,
			pageMeta: domains.Page{
				SubjectID:  userID,
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
		t.Run(tc.desc, func(t *testing.T) {
			repoCall1 := drepo.On("ListDomains", context.Background(), tc.pageMeta).Return(tc.listDomainsRes, tc.listDomainErr)
			dp, err := svc.ListDomains(context.Background(), tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.listDomainsRes, dp)
			repoCall1.Unset()
		})
	}
}

func TestDeleteUserFromDomains(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc                string
		userID              string
		listUserDomainsRes  domains.DomainsPage
		listUserDomainsRes1 domains.DomainsPage
		listUserDomainsErr  error
		listUserDomainsErr1 error
		err                 error
	}{
		{
			desc:   "delete user from domains successfully",
			userID: id,
			listUserDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
				Offset:  0,
				Limit:   10,
				Total:   1,
			},
			err: nil,
		},
		{
			desc:               "delete user from domains with repository error on list domains",
			userID:             id,
			listUserDomainsErr: svcerr.ErrViewEntity,
			err:                svcerr.ErrViewEntity,
		},
		{
			desc:   "delete user from domains with domains greater than default limit",
			userID: id,
			listUserDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
				Offset:  0,
				Limit:   100,
				Total:   101,
			},
			listUserDomainsRes1: domains.DomainsPage{
				Domains: []domains.Domain{domain},
				Offset:  100,
				Limit:   100,
				Total:   101,
			},
			err: nil,
		},
		{
			desc:   "delete user from domains with domains greater than default limit with error",
			userID: id,
			listUserDomainsRes: domains.DomainsPage{
				Domains: []domains.Domain{domain},
				Offset:  0,
				Limit:   100,
				Total:   101,
			},
			listUserDomainsRes1: domains.DomainsPage{},
			listUserDomainsErr1: svcerr.ErrViewEntity,
			err:                 svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("ListDomains", context.Background(), domains.Page{SubjectID: tc.userID, Limit: 100}).Return(tc.listUserDomainsRes, tc.listUserDomainsErr)
			repoCall1 := drepo.On("ListDomains", context.Background(), domains.Page{SubjectID: tc.userID, Offset: 100, Limit: 100}).Return(tc.listUserDomainsRes1, tc.listUserDomainsErr1)
			err := svc.DeleteUserFromDomains(context.Background(), tc.userID)
			assert.True(t, errors.Contains(err, tc.err))
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}
