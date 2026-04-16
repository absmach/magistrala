// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/events"
	"github.com/absmach/magistrala/re/mocks"
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
	validRule = generateTestRule(&testing.T{})
	validPage = re.Page{
		Offset: 0,
		Limit:  10,
		Total:  1,
		Rules:  []re.Rule{validRule},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, re.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.NewEventStoreMiddleware(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestAddRule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc       string
		session    authn.Session
		rule       re.Rule
		svcRes     re.Rule
		svcRoleRes []roles.RoleProvision
		svcErr     error
		resp       re.Rule
		err        error
	}{
		{
			desc:       "publish successfully",
			session:    validSession,
			rule:       validRule,
			svcRes:     validRule,
			svcRoleRes: []roles.RoleProvision{},
			svcErr:     nil,
			resp:       validRule,
			err:        nil,
		},
		{
			desc:       "failed to publish with service error",
			session:    validSession,
			rule:       validRule,
			svcRes:     re.Rule{},
			svcRoleRes: []roles.RoleProvision{},
			svcErr:     svcerr.ErrCreateEntity,
			resp:       re.Rule{},
			err:        svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AddRule", validCtx, tc.session, tc.rule).Return(tc.svcRes, tc.svcRoleRes, tc.svcErr)
			resp, _, err := nsvc.AddRule(validCtx, tc.session, tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestViewRule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		ruleID    string
		withRoles bool
		svcRes    re.Rule
		svcErr    error
		resp      re.Rule
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			ruleID:    validRule.ID,
			withRoles: false,
			svcRes:    validRule,
			svcErr:    nil,
			resp:      validRule,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			ruleID:    validRule.ID,
			withRoles: false,
			svcRes:    re.Rule{},
			svcErr:    svcerr.ErrViewEntity,
			resp:      re.Rule{},
			err:       svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewRule", validCtx, tc.session, tc.ruleID, tc.withRoles).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ViewRule(validCtx, tc.session, tc.ruleID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateRule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedRule := validRule
	updatedRule.Name = "updatedName"

	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		svcRes  re.Rule
		svcErr  error
		resp    re.Rule
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			rule:    updatedRule,
			svcRes:  updatedRule,
			svcErr:  nil,
			resp:    updatedRule,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			rule:    updatedRule,
			svcRes:  re.Rule{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    re.Rule{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateRule", validCtx, tc.session, tc.rule).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateRule(validCtx, tc.session, tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateRuleTags(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	taggedRule := validRule
	taggedRule.Tags = []string{"newtag1", "newtag2"}

	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		svcRes  re.Rule
		svcErr  error
		resp    re.Rule
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			rule:    taggedRule,
			svcRes:  taggedRule,
			svcErr:  nil,
			resp:    taggedRule,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			rule:    taggedRule,
			svcRes:  re.Rule{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    re.Rule{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateRuleTags", validCtx, tc.session, tc.rule).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateRuleTags(validCtx, tc.session, tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateRuleSchedule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		svcRes  re.Rule
		svcErr  error
		resp    re.Rule
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			rule:    validRule,
			svcRes:  validRule,
			svcErr:  nil,
			resp:    validRule,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			rule:    validRule,
			svcRes:  re.Rule{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    re.Rule{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateRuleSchedule", validCtx, tc.session, tc.rule).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateRuleSchedule(validCtx, tc.session, tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListRules(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta re.PageMeta
		svcRes   re.Page
		svcErr   error
		resp     re.Page
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: re.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validPage,
			svcErr: nil,
			resp:   validPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: re.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: re.Page{},
			svcErr: svcerr.ErrViewEntity,
			resp:   re.Page{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListRules", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListRules(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestRemoveRule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		ruleID  string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			ruleID:  validRule.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			ruleID:  validRule.ID,
			svcErr:  svcerr.ErrRemoveEntity,
			err:     svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveRule", validCtx, tc.session, tc.ruleID).Return(tc.svcErr)
			err := nsvc.RemoveRule(validCtx, tc.session, tc.ruleID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestEnableRule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		ruleID  string
		svcRes  re.Rule
		svcErr  error
		resp    re.Rule
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			ruleID:  validRule.ID,
			svcRes:  validRule,
			svcErr:  nil,
			resp:    validRule,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			ruleID:  validRule.ID,
			svcRes:  re.Rule{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    re.Rule{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("EnableRule", validCtx, tc.session, tc.ruleID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.EnableRule(validCtx, tc.session, tc.ruleID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisableRule(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		ruleID  string
		svcRes  re.Rule
		svcErr  error
		resp    re.Rule
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			ruleID:  validRule.ID,
			svcRes:  validRule,
			svcErr:  nil,
			resp:    validRule,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			ruleID:  validRule.ID,
			svcRes:  re.Rule{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    re.Rule{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DisableRule", validCtx, tc.session, tc.ruleID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.DisableRule(validCtx, tc.session, tc.ruleID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestStartScheduler(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	cases := []struct {
		desc   string
		svcErr error
		err    error
	}{
		{
			desc:   "start scheduler successfully",
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "failed with service error",
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("StartScheduler", context.Background()).Return(tc.svcErr)
			err := nsvc.StartScheduler(context.Background())
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestHandle(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	msg := &messaging.Message{Channel: "test.channel"}

	cases := []struct {
		desc   string
		msg    *messaging.Message
		svcErr error
		err    error
	}{
		{
			desc:   "handle successfully",
			msg:    msg,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "failed with service error",
			msg:    msg,
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Handle", tc.msg).Return(tc.svcErr)
			err := nsvc.Handle(tc.msg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestCancel(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	cases := []struct {
		desc   string
		svcErr error
		err    error
	}{
		{
			desc:   "cancel successfully",
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "failed with service error",
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Cancel").Return(tc.svcErr)
			err := nsvc.Cancel()
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func generateTestRule(t *testing.T) re.Rule {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return re.Rule{
		ID:           testsutil.GenerateUUID(t),
		Name:         "testrule",
		DomainID:     testsutil.GenerateUUID(t),
		InputChannel: "test.channel",
		Status:       re.EnabledStatus,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
}
