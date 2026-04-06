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
	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policiesMocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/sid"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	groupName = "smqx"
	validID   = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	ErrExpiry      = errors.New("session is expired")
	errAddPolicies = errors.New("failed to add policies")
	inValid        = "invalid"
	valid          = "valid"
	domain         = domains.Domain{
		ID:        validID,
		Name:      groupName,
		Tags:      []string{"tag1", "tag2"},
		Route:     "test",
		RoleID:    "test_role_id",
		CreatedBy: validID,
		UpdatedBy: validID,
	}
	validRoles = []roles.MemberRoleActions{
		{
			RoleID:     "domain_role_id",
			RoleName:   "domain_role_name",
			Actions:    []string{"read", "delete"},
			AccessType: "direct",
		},
	}
	domainWithRoles = domains.Domain{
		ID:        validID,
		Name:      groupName,
		Tags:      []string{"tag1", "tag2"},
		Route:     "test",
		RoleID:    "test_role_id",
		CreatedBy: validID,
		UpdatedBy: validID,
		Roles:     validRoles,
	}
	userID          = testsutil.GenerateUUID(&testing.T{})
	validSession    = authn.Session{UserID: userID}
	validInvitation = domains.Invitation{
		InviteeUserID: testsutil.GenerateUUID(&testing.T{}),
		DomainID:      testsutil.GenerateUUID(&testing.T{}),
		RoleID:        testsutil.GenerateUUID(&testing.T{}),
	}
)

var (
	drepo  *mocks.Repository
	dcache *mocks.Cache
	policy *policiesMocks.Service
)

func newService() domains.Service {
	drepo = new(mocks.Repository)
	dcache = new(mocks.Cache)
	idProvider := uuid.NewMock()
	sidProvider := sid.NewMock()
	policy = new(policiesMocks.Service)
	availableActions := []roles.Action{}
	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		groups.BuiltInRoleAdmin: availableActions,
	}
	ds, _ := domains.New(drepo, dcache, policy, idProvider, sidProvider, availableActions, builtInRoles)
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
			desc: "create domain with custom id",
			d: domains.Domain{
				ID:     validID,
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
			err:             svcerr.ErrRemoveEntity,
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
			err:             errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("SaveDomain", mock.Anything, mock.Anything).Return(tc.d, tc.saveDomainErr)
			repoCall1 := drepo.On("DeleteDomain", mock.Anything, mock.Anything).Return(tc.deleteDomainErr)
			repoCall2 := drepo.On("AddRoles", mock.Anything, mock.Anything).Return([]roles.RoleProvision{}, tc.addRolesErr)
			policyCall := policy.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesErr)
			policyCall1 := policy.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
			_, _, err := svc.CreateDomain(context.Background(), tc.session, tc.d)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
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

	superAdminSession := validSession
	superAdminSession.SuperAdmin = true

	cases := []struct {
		desc              string
		session           authn.Session
		domainID          string
		withRoles         bool
		retrieveDomainRes domains.Domain
		retrieveDomainErr error
		err               error
	}{
		{
			desc:              "retrieve domain successfully as super admin",
			session:           superAdminSession,
			withRoles:         false,
			domainID:          validID,
			retrieveDomainRes: domain,
			err:               nil,
		},
		{
			desc:              "retrieve domain successfully as non super admin",
			session:           validSession,
			withRoles:         false,
			domainID:          validID,
			retrieveDomainRes: domain,
			err:               nil,
		},
		{
			desc:              "retrieve domain successfully as non super admin with roles",
			session:           validSession,
			withRoles:         true,
			domainID:          validID,
			retrieveDomainRes: domainWithRoles,
			err:               nil,
		},
		{
			desc:              "retrieve domain with empty domain id",
			session:           validSession,
			withRoles:         false,
			domainID:          "",
			retrieveDomainErr: repoerr.ErrNotFound,
			err:               svcerr.ErrViewEntity,
		},
		{
			desc:              "retrieve non-existing domain",
			session:           validSession,
			withRoles:         false,
			domainID:          inValid,
			retrieveDomainErr: repoerr.ErrNotFound,
			err:               svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveDomainByID", context.Background(), tc.domainID).Return(tc.retrieveDomainRes, tc.retrieveDomainErr)
			repoCall1 := drepo.On("RetrieveDomainByIDWithRoles", context.Background(), tc.domainID, tc.session.UserID).Return(tc.retrieveDomainRes, tc.retrieveDomainErr)
			domain, err := svc.RetrieveDomain(context.Background(), tc.session, tc.domainID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err))

			switch tc.withRoles {
			case true:
				assert.Equal(t, domain.Roles, validRoles, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, validRoles, domain.Roles))
				ok := drepo.AssertCalled(t, "RetrieveDomainByIDWithRoles", context.Background(), tc.domainID, tc.session.UserID)
				assert.True(t, ok, fmt.Sprintf("RetrieveDomainByIDWithRoles was not called on %s", tc.desc))
			default:
				assert.Empty(t, domain.Roles)
				ok := drepo.AssertCalled(t, "RetrieveDomainByID", context.Background(), tc.domainID)
				assert.True(t, ok, fmt.Sprintf("RetrieveDomainByID was not called on %s", tc.desc))
			}

			assert.Equal(t, tc.retrieveDomainRes, domain)
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestUpdateDomain(t *testing.T) {
	svc := newService()

	updatedDomain := domain
	updatedDomain.Name = valid
	updatedDomain.Route = valid

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
				Name: &valid,
			},
			updateRes: updatedDomain,
			err:       nil,
		},
		{
			desc:     "update domain with empty domainID",
			session:  validSession,
			domainID: "",
			updateReq: domains.DomainReq{
				Name: &valid,
			},
			updateErr: repoerr.ErrNotFound,
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:     "update domain with failed to update",
			session:  validSession,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name: &valid,
			},
			updateErr: errors.ErrMalformedEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("UpdateDomain", context.Background(), tc.domainID, mock.Anything).Return(tc.updateRes, tc.updateErr)
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

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		enableRes domains.Domain
		enableErr error
		cacheErr  error
		resp      domains.Domain
		err       error
	}{
		{
			desc:      "enable domain successfully",
			session:   validSession,
			domainID:  domain.ID,
			enableRes: enabledDomain,
			resp:      enabledDomain,
			err:       nil,
		},
		{
			desc:      "enable domain with empty domainID",
			session:   validSession,
			domainID:  "",
			enableErr: repoerr.ErrNotFound,
			resp:      domains.Domain{},
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:      "enable domain with failed to enable",
			session:   validSession,
			domainID:  domain.ID,
			enableErr: errors.ErrMalformedEntity,
			resp:      domains.Domain{},
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:      "enable domain with failed to remove cache",
			session:   validSession,
			domainID:  domain.ID,
			enableRes: enabledDomain,
			cacheErr:  errors.ErrMalformedEntity,
			resp:      enabledDomain,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("UpdateDomain", context.Background(), tc.domainID, mock.Anything).Return(tc.enableRes, tc.enableErr)
			cacheCall := dcache.On("RemoveStatus", context.Background(), tc.domainID).Return(tc.cacheErr)
			domain, err := svc.EnableDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.resp, domain)
			repoCall.Unset()
			cacheCall.Unset()
		})
	}
}

func TestDisableDomain(t *testing.T) {
	svc := newService()

	disabledDomain := domain
	disabledDomain.Status = domains.DisabledStatus

	cases := []struct {
		desc       string
		session    authn.Session
		domainID   string
		disableRes domains.Domain
		disableErr error
		cacheErr   error
		resp       domains.Domain
		err        error
	}{
		{
			desc:       "disable domain successfully",
			session:    validSession,
			domainID:   domain.ID,
			disableRes: disabledDomain,
			resp:       disabledDomain,
			err:        nil,
		},
		{
			desc:       "disable domain with empty domainID",
			session:    validSession,
			domainID:   "",
			disableErr: repoerr.ErrNotFound,
			resp:       domains.Domain{},
			err:        svcerr.ErrUpdateEntity,
		},
		{
			desc:       "disable domain with failed to disable",
			session:    validSession,
			domainID:   domain.ID,
			disableErr: errors.ErrMalformedEntity,
			resp:       domains.Domain{},
			err:        svcerr.ErrUpdateEntity,
		},
		{
			desc:       "disable domain with failed to remove cache",
			session:    validSession,
			domainID:   domain.ID,
			disableRes: disabledDomain,
			cacheErr:   errors.ErrMalformedEntity,
			resp:       disabledDomain,
			err:        svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("UpdateDomain", context.Background(), tc.domainID, mock.Anything).Return(tc.disableRes, tc.disableErr)
			cacheCall := dcache.On("RemoveStatus", context.Background(), tc.domainID).Return(tc.cacheErr)
			domain, err := svc.DisableDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.disableRes, domain)
			repoCall.Unset()
			cacheCall.Unset()
		})
	}
}

func TestFreezeDomain(t *testing.T) {
	svc := newService()

	freezeDomain := domain
	freezeDomain.Status = domains.FreezeStatus

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		freezeRes domains.Domain
		freezeErr error
		cacheErr  error
		resp      domains.Domain
		err       error
	}{
		{
			desc:      "freeze domain successfully",
			session:   validSession,
			domainID:  domain.ID,
			freezeRes: freezeDomain,
			resp:      freezeDomain,
			err:       nil,
		},
		{
			desc:      "freeze domain with empty domainID",
			session:   validSession,
			domainID:  "",
			freezeErr: repoerr.ErrNotFound,
			resp:      domains.Domain{},
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:      "freeze domain with failed to freeze",
			session:   validSession,
			domainID:  domain.ID,
			freezeErr: errors.ErrMalformedEntity,
			resp:      domains.Domain{},
			err:       svcerr.ErrUpdateEntity,
		},
		{
			desc:      "freeze domain with failed to remove cache",
			session:   validSession,
			domainID:  domain.ID,
			freezeRes: freezeDomain,
			cacheErr:  errors.ErrMalformedEntity,
			resp:      freezeDomain,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("UpdateDomain", context.Background(), tc.domainID, mock.Anything).Return(tc.freezeRes, tc.freezeErr)
			cacheCall := dcache.On("RemoveStatus", context.Background(), tc.domainID).Return(tc.cacheErr)
			domain, err := svc.FreezeDomain(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.freezeRes, domain)
			repoCall.Unset()
			cacheCall.Unset()
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
				UserID: userID,
				Offset: 0,
				Limit:  10,
				Status: domains.EnabledStatus,
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
				Offset: 0,
				Limit:  10,
				Status: domains.EnabledStatus,
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
				UserID: userID,
				Offset: 0,
				Limit:  10,
				Status: domains.EnabledStatus,
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

func TestSendInvitation(t *testing.T) {
	svc := newService()

	rejectedInvitation := validInvitation
	rejectedInvitation.RejectedAt = time.Now()
	acceptedInvitation := validInvitation
	acceptedInvitation.ConfirmedAt = time.Now()
	resentInvitation := validInvitation
	resentInvitation.Resend = true

	cases := []struct {
		desc                string
		session             authn.Session
		req                 domains.Invitation
		retrieveRoleErr     error
		retrieveDomainErr   error
		createInvitationErr error
		retrieveInvRes      domains.Invitation
		retrieveInvErr      error
		updateRejectionErr  error
		err                 error
	}{
		{
			desc:    "send invitation successful",
			session: validSession,
			req:     validInvitation,
			err:     nil,
		},
		{
			desc:    "send invitation with invalid role id",
			session: validSession,
			req: domains.Invitation{
				DomainID:      testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				RoleID:        inValid,
			},
			retrieveRoleErr: repoerr.ErrNotFound,
			err:             svcerr.ErrInvalidRole,
		},
		{
			desc:              "send invitation with failed to retrieve domain",
			session:           validSession,
			req:               validInvitation,
			retrieveDomainErr: repoerr.ErrNotFound,
			err:               svcerr.ErrViewEntity,
		},
		{
			desc:                "send invitations with failed to save invitation",
			session:             validSession,
			req:                 validInvitation,
			createInvitationErr: repoerr.ErrCreateEntity,
			err:                 svcerr.ErrCreateEntity,
		},
		{
			desc:           "resend invitation successfully",
			session:        validSession,
			req:            resentInvitation,
			retrieveInvRes: rejectedInvitation,
			retrieveInvErr: nil,
			err:            nil,
		},
		{
			desc:           "resend invitation with failed to retrieve invitation",
			session:        validSession,
			req:            resentInvitation,
			retrieveInvRes: domains.Invitation{},
			retrieveInvErr: repoerr.ErrNotFound,
			err:            svcerr.ErrViewEntity,
		},
		{
			desc:           "resend an invitation that is already accepted",
			session:        validSession,
			req:            resentInvitation,
			retrieveInvRes: acceptedInvitation,
			retrieveInvErr: nil,
			err:            svcerr.ErrInvitationAlreadyAccepted,
		},
		{
			desc:               "resend invitation with failed to update rejection",
			session:            validSession,
			req:                resentInvitation,
			retrieveInvRes:     rejectedInvitation,
			retrieveInvErr:     nil,
			updateRejectionErr: repoerr.ErrUpdateEntity,
			err:                svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveRole", context.Background(), tc.req.RoleID).Return(roles.Role{Name: "admin"}, tc.retrieveRoleErr)
			repoCall1 := drepo.On("RetrieveDomainByID", context.Background(), tc.req.DomainID).Return(domains.Domain{Name: "test_domain"}, tc.retrieveDomainErr)
			repoCall2 := drepo.On("SaveInvitation", context.Background(), mock.Anything).Return(tc.createInvitationErr)
			repoCall3 := drepo.On("RetrieveInvitation", context.Background(), tc.req.InviteeUserID, tc.req.DomainID).Return(tc.retrieveInvRes, tc.retrieveInvErr)
			repoCall4 := drepo.On("UpdateRejection", context.Background(), mock.Anything).Return(tc.updateRejectionErr)
			_, err := svc.SendInvitation(context.Background(), tc.session, tc.req)
			assert.True(t, errors.Contains(err, tc.err))
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			repoCall3.Unset()
			repoCall4.Unset()
		})
	}
}

func TestListInvitations(t *testing.T) {
	svc := newService()

	validPageMeta := domains.InvitationPageMeta{
		Offset: 0,
		Limit:  10,
	}
	validResp := domains.InvitationPage{
		Total:  1,
		Offset: 0,
		Limit:  10,
		Invitations: []domains.Invitation{
			{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
				RoleName:      "admin",
				CreatedAt:     time.Now().Add(-time.Hour),
				UpdatedAt:     time.Now().Add(-time.Hour),
				ConfirmedAt:   time.Now().Add(-time.Hour),
			},
		},
	}

	cases := []struct {
		desc    string
		session authn.Session
		page    domains.InvitationPageMeta
		resp    domains.InvitationPage
		err     error
		repoErr error
	}{
		{
			desc:    "list invitations successful",
			session: validSession,
			page:    validPageMeta,
			resp:    validResp,
			err:     nil,
			repoErr: nil,
		},

		{
			desc:    "list invitations unsuccessful",
			session: validSession,
			page:    validPageMeta,
			err:     svcerr.ErrViewEntity,
			resp:    domains.InvitationPage{},
			repoErr: repoerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveAllInvitations", context.Background(), mock.Anything).Return(tc.resp, tc.repoErr)
			resp, err := svc.ListInvitations(context.Background(), tc.session, tc.page)
			assert.True(t, errors.Contains(err, tc.err), tc.desc)
			assert.Equal(t, tc.resp, resp, tc.desc)
			repoCall.Unset()
		})
	}
}

func TestListDomainInvitations(t *testing.T) {
	svc := newService()

	validPageMeta := domains.InvitationPageMeta{
		Offset: 0,
		Limit:  10,
	}
	validResp := domains.InvitationPage{
		Total:  1,
		Offset: 0,
		Limit:  10,
		Invitations: []domains.Invitation{
			{
				InvitedBy:     testsutil.GenerateUUID(t),
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
				RoleName:      "admin",
				CreatedAt:     time.Now().Add(-time.Hour),
				UpdatedAt:     time.Now().Add(-time.Hour),
				ConfirmedAt:   time.Now().Add(-time.Hour),
			},
		},
	}

	cases := []struct {
		desc    string
		session authn.Session
		page    domains.InvitationPageMeta
		resp    domains.InvitationPage
		repoErr error
		err     error
	}{
		{
			desc:    "list domain invitations successful",
			session: validSession,
			page:    validPageMeta,
			resp:    validResp,
			repoErr: nil,
			err:     nil,
		},

		{
			desc:    "list domain invitations unsuccessful",
			session: validSession,
			page:    validPageMeta,
			resp:    domains.InvitationPage{},
			repoErr: repoerr.ErrViewEntity,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveAllInvitations", context.Background(), mock.Anything).Return(tc.resp, tc.repoErr)
			resp, err := svc.ListDomainInvitations(context.Background(), tc.session, tc.page)
			assert.True(t, errors.Contains(err, tc.err), tc.desc)
			assert.Equal(t, tc.resp, resp, tc.desc)
			repoCall.Unset()
		})
	}
}

func TestAcceptInvitation(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc                  string
		domainID              string
		session               authn.Session
		resp                  domains.Invitation
		retrieveInvitationErr error
		retrieveDomainErr     error
		retrieveRoleErr       error
		updateConfirmationErr error
		addRoleMemberErr      error
		err                   error
	}{
		{
			desc:     "accept invitation successful",
			domainID: validID,
			session:  validSession,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:                  "accept invitation with failed to retrieve invitation",
			session:               validSession,
			retrieveInvitationErr: repoerr.ErrNotFound,
			err:                   svcerr.ErrNotFound,
		},
		{
			desc:    "accept invitation with of different user",
			session: validSession,
			resp: domains.Invitation{
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
			},
			err: svcerr.ErrAuthorization,
		},
		{
			desc:     "accept invitation with failed to add role member",
			domainID: validID,
			session:  validSession,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
			},
			addRoleMemberErr: repoerr.ErrMalformedEntity,
			err:              svcerr.ErrUpdateEntity,
		},
		{
			desc:     "accept invitation with failed update confirmation",
			session:  validSession,
			domainID: validID,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      validID,
				RoleID:        testsutil.GenerateUUID(t),
			},
			updateConfirmationErr: repoerr.ErrNotFound,
			err:                   svcerr.ErrUpdateEntity,
		},
		{
			desc:     "accept invitation that is already confirmed",
			session:  validSession,
			domainID: validID,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
				ConfirmedAt:   time.Now(),
			},
			err: svcerr.ErrInvitationAlreadyAccepted,
		},
		{
			desc:     "accept rejected invitation",
			session:  validSession,
			domainID: validID,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
				RejectedAt:    time.Now(),
			},
			err: svcerr.ErrInvitationAlreadyRejected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveInvitation", context.Background(), tc.session.UserID, tc.domainID).Return(tc.resp, tc.retrieveInvitationErr)
			repoCall1 := drepo.On("RetrieveDomainByID", context.Background(), tc.domainID).Return(domains.Domain{Name: "test_domain"}, tc.retrieveDomainErr)
			repoCall2 := drepo.On("RetrieveRole", context.Background(), tc.resp.RoleID).Return(roles.Role{Name: "admin"}, tc.retrieveRoleErr)
			repoCall3 := drepo.On("RetrieveEntityRole", context.Background(), tc.domainID, tc.resp.RoleID).Return(roles.Role{}, tc.addRoleMemberErr)
			policyCall := policy.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addRoleMemberErr)
			repoCall4 := drepo.On("RoleAddMembers", context.Background(), mock.Anything, []string{tc.resp.InviteeUserID}).Return([]string{}, tc.addRoleMemberErr)
			repoCall5 := drepo.On("UpdateConfirmation", context.Background(), mock.Anything).Return(tc.updateConfirmationErr)
			_, err := svc.AcceptInvitation(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			repoCall3.Unset()
			policyCall.Unset()
			repoCall4.Unset()
			repoCall5.Unset()
		})
	}
}

func TestRejectInvitation(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc                  string
		domainID              string
		session               authn.Session
		resp                  domains.Invitation
		retrieveInvitationErr error
		updateConfirmationErr error
		addRoleMemberErr      error
		err                   error
	}{
		{
			desc:     "reject invitation successful",
			domainID: validID,
			session:  validSession,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:                  "reject invitation with failed to retrieve invitation",
			session:               validSession,
			retrieveInvitationErr: repoerr.ErrNotFound,
			err:                   svcerr.ErrNotFound,
		},
		{
			desc:    "reject invitation with of different user",
			session: validSession,
			resp: domains.Invitation{
				InviteeUserID: testsutil.GenerateUUID(t),
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
			},
			err: svcerr.ErrAuthorization,
		},
		{
			desc:     "reject invitation with failed update confirmation",
			session:  validSession,
			domainID: validID,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      validID,
				RoleID:        testsutil.GenerateUUID(t),
			},
			updateConfirmationErr: repoerr.ErrNotFound,
			err:                   svcerr.ErrUpdateEntity,
		},
		{
			desc:     "reject invitation that is already confirmed",
			session:  validSession,
			domainID: validID,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
				ConfirmedAt:   time.Now(),
			},
			err: svcerr.ErrInvitationAlreadyAccepted,
		},
		{
			desc:     "reject rejected invitation",
			session:  validSession,
			domainID: validID,
			resp: domains.Invitation{
				InviteeUserID: userID,
				DomainID:      testsutil.GenerateUUID(t),
				RoleID:        testsutil.GenerateUUID(t),
				RejectedAt:    time.Now(),
			},
			err: svcerr.ErrInvitationAlreadyRejected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveInvitation", context.Background(), tc.session.UserID, tc.domainID).Return(tc.resp, tc.retrieveInvitationErr)
			repoCall1 := drepo.On("RetrieveDomainByID", context.Background(), tc.domainID).Return(domains.Domain{Name: "test_domain"}, nil)
			repoCall2 := drepo.On("RetrieveRole", context.Background(), tc.resp.RoleID).Return(roles.Role{Name: "admin"}, nil)
			repoCall3 := drepo.On("UpdateRejection", context.Background(), mock.Anything).Return(tc.updateConfirmationErr)
			_, err := svc.RejectInvitation(context.Background(), tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			repoCall3.Unset()
		})
	}
}

func TestDeleteInvitation(t *testing.T) {
	svc := newService()

	acceptedInv := validInvitation
	acceptedInv.ConfirmedAt = time.Now()
	rejectedInv := validInvitation
	rejectedInv.RejectedAt = time.Now()

	cases := []struct {
		desc                  string
		userID                string
		domainID              string
		session               authn.Session
		resp                  domains.Invitation
		retrieveInvitationErr error
		deleteInvitationErr   error
		err                   error
	}{
		{
			desc:     "delete invitations successful",
			userID:   testsutil.GenerateUUID(t),
			domainID: testsutil.GenerateUUID(t),
			session:  validSession,
			resp:     validInvitation,
			err:      nil,
		},
		{
			desc:     "delete invitations for the same user",
			userID:   validInvitation.InviteeUserID,
			domainID: validInvitation.DomainID,
			resp:     validInvitation,
			session:  authn.Session{UserID: validInvitation.InviteeUserID},
			err:      nil,
		},
		{
			desc:     "delete invitations for the invited user",
			userID:   validInvitation.InviteeUserID,
			domainID: validInvitation.DomainID,
			session:  validSession,
			resp:     validInvitation,
			err:      nil,
		},
		{
			desc:     "delete accepted invitation as non invitee user",
			userID:   validID,
			domainID: validInvitation.DomainID,
			session:  validSession,
			resp:     acceptedInv,
			err:      svcerr.ErrInvitationAlreadyAccepted,
		},
		{
			desc:     "delete rejected invitation as non invitee user",
			userID:   validID,
			domainID: validInvitation.DomainID,
			session:  validSession,
			resp:     rejectedInv,
			err:      svcerr.ErrInvitationAlreadyRejected,
		},
		{
			desc:                  "delete invitation with error retrieving invitation",
			userID:                validInvitation.InviteeUserID,
			domainID:              validInvitation.DomainID,
			session:               validSession,
			resp:                  domains.Invitation{},
			retrieveInvitationErr: repoerr.ErrNotFound,
			err:                   svcerr.ErrRemoveEntity,
		},
		{
			desc:                "delete invitation with error deleting invitation",
			userID:              validInvitation.InviteeUserID,
			domainID:            validInvitation.DomainID,
			session:             validSession,
			resp:                domains.Invitation{},
			deleteInvitationErr: repoerr.ErrNotFound,
			err:                 svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := drepo.On("RetrieveInvitation", context.Background(), mock.Anything, mock.Anything).Return(tc.resp, tc.retrieveInvitationErr)
			repoCall1 := drepo.On("DeleteUsersInvitations", context.Background(), mock.Anything, mock.Anything).Return(tc.deleteInvitationErr)
			err := svc.DeleteInvitation(context.Background(), tc.session, tc.userID, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err))
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}
