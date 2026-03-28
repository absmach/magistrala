// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/domains/events"
	"github.com/absmach/supermq/domains/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	storeClient  *redis.Client
	storeURL     string
	validSession = authn.Session{
		DomainID: testsutil.GenerateUUID(&testing.T{}),
		UserID:   testsutil.GenerateUUID(&testing.T{}),
	}
	validDomain      = generateTestDomain(&testing.T{})
	validDomainsPage = domains.DomainsPage{
		Limit:   10,
		Offset:  0,
		Total:   1,
		Domains: []domains.Domain{validDomain},
	}
	validInvitation      = generateTestInvitation(&testing.T{})
	validInvitationsPage = domains.InvitationPage{
		Total:       1,
		Offset:      0,
		Limit:       10,
		Invitations: []domains.Invitation{validInvitation},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, domains.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.NewEventStoreMiddleware(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestCreateDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validID := testsutil.GenerateUUID(t)
	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, validID)

	cases := []struct {
		desc        string
		session     authn.Session
		domain      domains.Domain
		svcRes      domains.Domain
		svcRoleRes  []roles.RoleProvision
		svcErr      error
		resp        domains.Domain
		respRoleRes []roles.RoleProvision
		err         error
	}{
		{
			desc:        "publish successfully",
			session:     validSession,
			domain:      validDomain,
			svcRes:      validDomain,
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      nil,
			resp:        validDomain,
			respRoleRes: []roles.RoleProvision{},
			err:         nil,
		},
		{
			desc:        "failed to publish with service error",
			session:     validSession,
			domain:      validDomain,
			svcRes:      domains.Domain{},
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      svcerr.ErrCreateEntity,
			resp:        domains.Domain{},
			respRoleRes: []roles.RoleProvision{},
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("CreateDomain", validCtx, tc.session, tc.domain).Return(tc.svcRes, tc.svcRoleRes, tc.svcErr)
			resp, respRoleRes, err := nsvc.CreateDomain(validCtx, tc.session, tc.domain)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			assert.Equal(t, tc.respRoleRes, respRoleRes, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.respRoleRes, respRoleRes))
			svcCall.Unset()
		})
	}
}

func TestRetrieveDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		withRoles bool
		svcRes    domains.Domain
		svcErr    error
		resp      domains.Domain
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			domainID:  validDomain.ID,
			withRoles: false,
			svcRes:    validDomain,
			svcErr:    nil,
			resp:      validDomain,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			domainID:  validDomain.ID,
			withRoles: false,
			svcRes:    domains.Domain{},
			svcErr:    svcerr.ErrViewEntity,
			resp:      domains.Domain{},
			err:       svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveDomain", validCtx, tc.session, tc.domainID, tc.withRoles).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.RetrieveDomain(validCtx, tc.session, tc.domainID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedDomain := validDomain
	updatedDomain.Name = "updatedName"
	domainReq := domains.DomainReq{
		Name: &updatedDomain.Name,
	}

	cases := []struct {
		desc      string
		session   authn.Session
		domainID  string
		domainReq domains.DomainReq
		svcRes    domains.Domain
		svcErr    error
		resp      domains.Domain
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			domainID:  validDomain.ID,
			domainReq: domainReq,
			svcRes:    updatedDomain,
			svcErr:    nil,
			resp:      updatedDomain,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			domainID:  validDomain.ID,
			domainReq: domainReq,
			svcRes:    domains.Domain{},
			svcErr:    svcerr.ErrUpdateEntity,
			resp:      domains.Domain{},
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateDomain", validCtx, tc.session, tc.domainID, tc.domainReq).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateDomain(validCtx, tc.session, tc.domainID, tc.domainReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestEnableDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		resp     domains.Domain
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   validDomain,
			svcErr:   nil,
			resp:     validDomain,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     domains.Domain{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("EnableDomain", validCtx, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.EnableDomain(validCtx, tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisableDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		resp     domains.Domain
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   validDomain,
			svcErr:   nil,
			resp:     validDomain,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     domains.Domain{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DisableDomain", validCtx, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.DisableDomain(validCtx, tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestFreezeDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		svcRes   domains.Domain
		svcErr   error
		resp     domains.Domain
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   validDomain,
			svcErr:   nil,
			resp:     validDomain,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   domains.Domain{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     domains.Domain{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("FreezeDomain", validCtx, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.FreezeDomain(validCtx, tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListDomains(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta domains.Page
		svcRes   domains.DomainsPage
		svcErr   error
		resp     domains.DomainsPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: domains.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validDomainsPage,
			svcErr: nil,
			resp:   validDomainsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: domains.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: domains.DomainsPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   domains.DomainsPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListDomains", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListDomains(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDeleteDomain(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		svcErr   error
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			domainID: validDomain.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			domainID: validDomain.ID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DeleteDomain", validCtx, tc.session, tc.domainID).Return(tc.svcErr)
			err := nsvc.DeleteDomain(validCtx, tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestSendInvitation(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc       string
		session    authn.Session
		invitation domains.Invitation
		svcRes     domains.Invitation
		svcErr     error
		resp       domains.Invitation
		err        error
	}{
		{
			desc:       "publish successfully",
			session:    validSession,
			invitation: validInvitation,
			svcRes:     validInvitation,
			svcErr:     nil,
			resp:       validInvitation,
			err:        nil,
		},
		{
			desc:       "failed to publish with service error",
			session:    validSession,
			invitation: validInvitation,
			svcRes:     domains.Invitation{},
			svcErr:     svcerr.ErrCreateEntity,
			resp:       domains.Invitation{},
			err:        svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SendInvitation", validCtx, tc.session, tc.invitation).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.SendInvitation(validCtx, tc.session, tc.invitation)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListInvitations(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta domains.InvitationPageMeta
		svcRes   domains.InvitationPage
		svcErr   error
		resp     domains.InvitationPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: domains.InvitationPageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validInvitationsPage,
			svcErr: nil,
			resp:   validInvitationsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: domains.InvitationPageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: domains.InvitationPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   domains.InvitationPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListInvitations", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListInvitations(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListDomainInvitations(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta domains.InvitationPageMeta
		svcRes   domains.InvitationPage
		svcErr   error
		resp     domains.InvitationPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: domains.InvitationPageMeta{
				Limit:    10,
				Offset:   0,
				DomainID: validDomain.ID,
			},
			svcRes: validInvitationsPage,
			svcErr: nil,
			resp:   validInvitationsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: domains.InvitationPageMeta{
				Limit:    10,
				Offset:   0,
				DomainID: validDomain.ID,
			},
			svcRes: domains.InvitationPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   domains.InvitationPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListDomainInvitations", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListDomainInvitations(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestAcceptInvitation(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		svcRes   domains.Invitation
		svcErr   error
		resp     domains.Invitation
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   validInvitation,
			svcErr:   nil,
			resp:     validInvitation,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   domains.Invitation{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     domains.Invitation{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AcceptInvitation", validCtx, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.AcceptInvitation(validCtx, tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDeleteInvitation(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc          string
		session       authn.Session
		inviteeUserID string
		domainID      string
		svcErr        error
		err           error
	}{
		{
			desc:          "publish successfully",
			session:       validSession,
			inviteeUserID: validInvitation.InvitedBy,
			domainID:      validDomain.ID,
			svcErr:        nil,
			err:           nil,
		},
		{
			desc:          "failed to publish with service error",
			session:       validSession,
			inviteeUserID: validInvitation.InvitedBy,
			domainID:      validDomain.ID,
			svcErr:        svcerr.ErrRemoveEntity,
			err:           svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DeleteInvitation", validCtx, tc.session, tc.inviteeUserID, tc.domainID).Return(tc.svcErr)
			err := nsvc.DeleteInvitation(validCtx, tc.session, tc.inviteeUserID, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRejectInvitation(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		domainID string
		svcRes   domains.Invitation
		svcErr   error
		resp     domains.Invitation
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   validInvitation,
			svcErr:   nil,
			resp:     validInvitation,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			domainID: validDomain.ID,
			svcRes:   domains.Invitation{},
			svcErr:   svcerr.ErrUpdateEntity,
			resp:     domains.Invitation{},
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RejectInvitation", validCtx, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.RejectInvitation(validCtx, tc.session, tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func generateTestDomain(t *testing.T) domains.Domain {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return domains.Domain{
		ID:        testsutil.GenerateUUID(t),
		Name:      "domainname",
		Tags:      []string{"tag1", "tag2"},
		Metadata:  domains.Metadata{"key1": "value1"},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    domains.EnabledStatus,
	}
}

func generateTestInvitation(t *testing.T) domains.Invitation {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return domains.Invitation{
		InvitedBy:     testsutil.GenerateUUID(t),
		InviteeUserID: testsutil.GenerateUUID(t),
		DomainID:      testsutil.GenerateUUID(t),
		RoleID:        testsutil.GenerateUUID(t),
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
}
