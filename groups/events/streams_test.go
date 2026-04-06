// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/groups/events"
	"github.com/absmach/magistrala/groups/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/roles"
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
	validGroup      = generateTestGroup(&testing.T{})
	validGroupsPage = groups.Page{
		PageMeta: groups.PageMeta{
			Limit:  10,
			Offset: 0,
			Total:  1,
		},
		Groups: []groups.Group{validGroup},
	}
	validHierarchyPage = groups.HierarchyPage{
		HierarchyPageMeta: groups.HierarchyPageMeta{
			Level:     1,
			Direction: -1,
			Tree:      false,
		},
		Groups: []groups.Group{validGroup},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, groups.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.New(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestCreateGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validID := testsutil.GenerateUUID(t)
	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, validID)

	cases := []struct {
		desc        string
		session     authn.Session
		group       groups.Group
		svcRes      groups.Group
		svcRoleRes  []roles.RoleProvision
		svcErr      error
		resp        groups.Group
		respRoleRes []roles.RoleProvision
		err         error
	}{
		{
			desc:        "publish successfully",
			session:     validSession,
			group:       validGroup,
			svcRes:      validGroup,
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      nil,
			resp:        validGroup,
			respRoleRes: []roles.RoleProvision{},
			err:         nil,
		},
		{
			desc:        "failed to publish with service error",
			session:     validSession,
			group:       validGroup,
			svcRes:      groups.Group{},
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      svcerr.ErrCreateEntity,
			resp:        groups.Group{},
			respRoleRes: []roles.RoleProvision{},
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("CreateGroup", validCtx, tc.session, tc.group).Return(tc.svcRes, tc.svcRoleRes, tc.svcErr)
			resp, respRoleRes, err := nsvc.CreateGroup(validCtx, tc.session, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			assert.Equal(t, tc.respRoleRes, respRoleRes, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.respRoleRes, respRoleRes))
			svcCall.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		groupID   string
		withRoles bool
		svcRes    groups.Group
		svcErr    error
		resp      groups.Group
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			groupID:   validGroup.ID,
			withRoles: false,
			svcRes:    validGroup,
			svcErr:    nil,
			resp:      validGroup,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			groupID:   validGroup.ID,
			withRoles: false,
			svcRes:    groups.Group{},
			svcErr:    svcerr.ErrViewEntity,
			resp:      groups.Group{},
			err:       svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewGroup", validCtx, tc.session, tc.groupID, tc.withRoles).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ViewGroup(validCtx, tc.session, tc.groupID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedGroup := validGroup
	updatedGroup.Name = "updatedName"

	cases := []struct {
		desc    string
		session authn.Session
		group   groups.Group
		svcRes  groups.Group
		svcErr  error
		resp    groups.Group
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			group:   updatedGroup,
			svcRes:  updatedGroup,
			svcErr:  nil,
			resp:    updatedGroup,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			group:   updatedGroup,
			svcRes:  groups.Group{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    groups.Group{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateGroup", validCtx, tc.session, tc.group).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateGroup(validCtx, tc.session, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateGroupTags(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedGroup := validGroup
	updatedGroup.Tags = []string{"newTag1", "newTag2"}

	cases := []struct {
		desc    string
		session authn.Session
		group   groups.Group
		svcRes  groups.Group
		svcErr  error
		resp    groups.Group
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			group:   updatedGroup,
			svcRes:  updatedGroup,
			svcErr:  nil,
			resp:    updatedGroup,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			group:   updatedGroup,
			svcRes:  groups.Group{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    groups.Group{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateGroupTags", validCtx, tc.session, tc.group).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateGroupTags(validCtx, tc.session, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestEnableGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		groupID string
		svcRes  groups.Group
		svcErr  error
		resp    groups.Group
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			groupID: validGroup.ID,
			svcRes:  validGroup,
			svcErr:  nil,
			resp:    validGroup,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			groupID: validGroup.ID,
			svcRes:  groups.Group{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    groups.Group{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("EnableGroup", validCtx, tc.session, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.EnableGroup(validCtx, tc.session, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisableGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		groupID string
		svcRes  groups.Group
		svcErr  error
		resp    groups.Group
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			groupID: validGroup.ID,
			svcRes:  validGroup,
			svcErr:  nil,
			resp:    validGroup,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			groupID: validGroup.ID,
			svcRes:  groups.Group{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    groups.Group{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DisableGroup", validCtx, tc.session, tc.groupID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.DisableGroup(validCtx, tc.session, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta groups.PageMeta
		svcRes   groups.Page
		svcErr   error
		resp     groups.Page
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validGroupsPage,
			svcErr: nil,
			resp:   validGroupsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: groups.Page{},
			svcErr: svcerr.ErrViewEntity,
			resp:   groups.Page{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListGroups", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListGroups(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListUserGroups(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		userID   string
		pageMeta groups.PageMeta
		svcRes   groups.Page
		svcErr   error
		resp     groups.Page
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validGroupsPage,
			svcErr: nil,
			resp:   validGroupsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: groups.Page{},
			svcErr: svcerr.ErrViewEntity,
			resp:   groups.Page{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListUserGroups", validCtx, tc.session, tc.userID, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListUserGroups(validCtx, tc.session, tc.userID, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		groupID string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			groupID: validGroup.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			groupID: validGroup.ID,
			svcErr:  svcerr.ErrRemoveEntity,
			err:     svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DeleteGroup", validCtx, tc.session, tc.groupID).Return(tc.svcErr)
			err := nsvc.DeleteGroup(validCtx, tc.session, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRetrieveGroupHierarchy(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		groupID  string
		pageMeta groups.HierarchyPageMeta
		svcRes   groups.HierarchyPage
		svcErr   error
		resp     groups.HierarchyPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			groupID: validGroup.ID,
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			svcRes: validHierarchyPage,
			svcErr: nil,
			resp:   validHierarchyPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			groupID: validGroup.ID,
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			svcRes: groups.HierarchyPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   groups.HierarchyPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveGroupHierarchy", validCtx, tc.session, tc.groupID, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.RetrieveGroupHierarchy(validCtx, tc.session, tc.groupID, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestAddParentGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		groupID  string
		parentID string
		svcErr   error
		err      error
	}{
		{
			desc:     "publish successfully",
			session:  validSession,
			groupID:  validGroup.ID,
			parentID: testsutil.GenerateUUID(t),
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "failed to publish with service error",
			session:  validSession,
			groupID:  validGroup.ID,
			parentID: testsutil.GenerateUUID(t),
			svcErr:   svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AddParentGroup", validCtx, tc.session, tc.groupID, tc.parentID).Return(tc.svcErr)
			err := nsvc.AddParentGroup(validCtx, tc.session, tc.groupID, tc.parentID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		groupID string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			groupID: validGroup.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			groupID: validGroup.ID,
			svcErr:  svcerr.ErrUpdateEntity,
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveParentGroup", validCtx, tc.session, tc.groupID).Return(tc.svcErr)
			err := nsvc.RemoveParentGroup(validCtx, tc.session, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestAddChildrenGroups(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc             string
		session          authn.Session
		groupID          string
		childrenGroupIDs []string
		svcErr           error
		err              error
	}{
		{
			desc:             "publish successfully",
			session:          validSession,
			groupID:          validGroup.ID,
			childrenGroupIDs: []string{testsutil.GenerateUUID(t)},
			svcErr:           nil,
			err:              nil,
		},
		{
			desc:             "failed to publish with service error",
			session:          validSession,
			groupID:          validGroup.ID,
			childrenGroupIDs: []string{testsutil.GenerateUUID(t)},
			svcErr:           svcerr.ErrUpdateEntity,
			err:              svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AddChildrenGroups", validCtx, tc.session, tc.groupID, tc.childrenGroupIDs).Return(tc.svcErr)
			err := nsvc.AddChildrenGroups(validCtx, tc.session, tc.groupID, tc.childrenGroupIDs)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRemoveChildrenGroups(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc             string
		session          authn.Session
		groupID          string
		childrenGroupIDs []string
		svcErr           error
		err              error
	}{
		{
			desc:             "publish successfully",
			session:          validSession,
			groupID:          validGroup.ID,
			childrenGroupIDs: []string{testsutil.GenerateUUID(t)},
			svcErr:           nil,
			err:              nil,
		},
		{
			desc:             "failed to publish with service error",
			session:          validSession,
			groupID:          validGroup.ID,
			childrenGroupIDs: []string{testsutil.GenerateUUID(t)},
			svcErr:           svcerr.ErrUpdateEntity,
			err:              svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveChildrenGroups", validCtx, tc.session, tc.groupID, tc.childrenGroupIDs).Return(tc.svcErr)
			err := nsvc.RemoveChildrenGroups(validCtx, tc.session, tc.groupID, tc.childrenGroupIDs)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRemoveAllChildrenGroups(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc    string
		session authn.Session
		groupID string
		svcErr  error
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			groupID: validGroup.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			groupID: validGroup.ID,
			svcErr:  svcerr.ErrUpdateEntity,
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveAllChildrenGroups", validCtx, tc.session, tc.groupID).Return(tc.svcErr)
			err := nsvc.RemoveAllChildrenGroups(validCtx, tc.session, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestListChildrenGroups(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc       string
		session    authn.Session
		groupID    string
		startLevel int64
		endLevel   int64
		pageMeta   groups.PageMeta
		svcRes     groups.Page
		svcErr     error
		resp       groups.Page
		err        error
	}{
		{
			desc:       "publish successfully",
			session:    validSession,
			groupID:    validGroup.ID,
			startLevel: 1,
			endLevel:   5,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validGroupsPage,
			svcErr: nil,
			resp:   validGroupsPage,
			err:    nil,
		},
		{
			desc:       "failed to publish with service error",
			session:    validSession,
			groupID:    validGroup.ID,
			startLevel: 1,
			endLevel:   5,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			svcRes: groups.Page{},
			svcErr: svcerr.ErrViewEntity,
			resp:   groups.Page{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListChildrenGroups", validCtx, tc.session, tc.groupID, tc.startLevel, tc.endLevel, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListChildrenGroups(validCtx, tc.session, tc.groupID, tc.startLevel, tc.endLevel, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func generateTestGroup(t *testing.T) groups.Group {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return groups.Group{
		ID:        testsutil.GenerateUUID(t),
		Name:      "groupname",
		Domain:    testsutil.GenerateUUID(t),
		Tags:      []string{"tag1", "tag2"},
		Metadata:  groups.Metadata{"key1": "value1"},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    groups.EnabledStatus,
		Level:     1,
	}
}
