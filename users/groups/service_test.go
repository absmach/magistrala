// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/mainflux/mainflux/internal/testsutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users/clients"
	cmocks "github.com/mainflux/mainflux/users/clients/mocks"
	"github.com/mainflux/mainflux/users/groups"
	"github.com/mainflux/mainflux/users/groups/mocks"
	"github.com/mainflux/mainflux/users/hasher"
	"github.com/mainflux/mainflux/users/jwt"
	pmocks "github.com/mainflux/mainflux/users/policies/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	idProvider     = uuid.New()
	phasher        = hasher.New()
	secret         = "strongsecret"
	validGMetadata = mfclients.Metadata{"role": "client"}
	inValidToken   = "invalidToken"
	description    = "shortdescription"
	gName          = "groupname"
	group          = mfgroups.Group{
		Name:        gName,
		Description: description,
		Metadata:    validGMetadata,
		Status:      mfclients.EnabledStatus,
	}
	withinDuration  = 5 * time.Second
	passRegex       = regexp.MustCompile("^.{8,}$")
	accessDuration  = time.Minute * 1
	refreshDuration = time.Minute * 10
)

func TestCreateGroup(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	cases := []struct {
		desc  string
		group mfgroups.Group
		err   error
	}{
		{
			desc:  "create new group",
			group: group,
			err:   nil,
		},
		{
			desc:  "create group with existing name",
			group: group,
			err:   nil,
		},
		{
			desc: "create group with parent",
			group: mfgroups.Group{
				Name:   gName,
				Parent: testsutil.GenerateUUID(t, idProvider),
				Status: mfclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "create group with invalid parent",
			group: mfgroups.Group{
				Name:   gName,
				Parent: mocks.WrongID,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "create group with invalid owner",
			group: mfgroups.Group{
				Name:  gName,
				Owner: mocks.WrongID,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc:  "create group with missing name",
			group: mfgroups.Group{},
			err:   errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := gRepo.On("Save", context.Background(), mock.Anything).Return(tc.group, tc.err)
		createdAt := time.Now()
		expected, err := svc.CreateGroup(context.Background(), testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher), tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, expected.ID, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, expected.ID))
			assert.WithinDuration(t, expected.CreatedAt, createdAt, withinDuration, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected.CreatedAt, createdAt))
			tc.group.ID = expected.ID
			tc.group.CreatedAt = expected.CreatedAt
			tc.group.UpdatedAt = expected.UpdatedAt
			tc.group.UpdatedBy = expected.UpdatedBy
			tc.group.Owner = expected.Owner
			assert.Equal(t, tc.group, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.group, expected))
			ok := repoCall.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestUpdateGroup(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	group.ID = testsutil.GenerateUUID(t, idProvider)

	cases := []struct {
		desc     string
		token    string
		group    mfgroups.Group
		response mfgroups.Group
		err      error
	}{
		{
			desc: "update group name",
			group: mfgroups.Group{
				ID:   group.ID,
				Name: "NewName",
			},
			response: mfgroups.Group{
				ID:   group.ID,
				Name: "NewName",
			},
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:   nil,
		},
		{
			desc: "update group description",
			group: mfgroups.Group{
				ID:          group.ID,
				Description: "NewDescription",
			},
			response: mfgroups.Group{
				ID:          group.ID,
				Description: "NewDescription",
			},
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:   nil,
		},
		{
			desc: "update group metadata",
			group: mfgroups.Group{
				ID: group.ID,
				Metadata: mfclients.Metadata{
					"field": "value2",
				},
			},
			response: mfgroups.Group{
				ID: group.ID,
				Metadata: mfclients.Metadata{
					"field": "value2",
				},
			},
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:   nil,
		},
		{
			desc: "update group name with invalid group id",
			group: mfgroups.Group{
				ID:   mocks.WrongID,
				Name: "NewName",
			},
			response: mfgroups.Group{},
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:      errors.ErrNotFound,
		},
		{
			desc: "update group description with invalid group id",
			group: mfgroups.Group{
				ID:          mocks.WrongID,
				Description: "NewDescription",
			},
			response: mfgroups.Group{},
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:      errors.ErrNotFound,
		},
		{
			desc: "update group metadata with invalid group id",
			group: mfgroups.Group{
				ID: mocks.WrongID,
				Metadata: mfclients.Metadata{
					"field": "value2",
				},
			},
			response: mfgroups.Group{},
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:      errors.ErrNotFound,
		},
		{
			desc: "update group name with invalid token",
			group: mfgroups.Group{
				ID:   group.ID,
				Name: "NewName",
			},
			response: mfgroups.Group{},
			token:    inValidToken,
			err:      errors.ErrAuthentication,
		},
		{
			desc: "update group description with invalid token",
			group: mfgroups.Group{
				ID:          group.ID,
				Description: "NewDescription",
			},
			response: mfgroups.Group{},
			token:    inValidToken,
			err:      errors.ErrAuthentication,
		},
		{
			desc: "update group metadata with invalid token",
			group: mfgroups.Group{
				ID: group.ID,
				Metadata: mfclients.Metadata{
					"field": "value2",
				},
			},
			response: mfgroups.Group{},
			token:    inValidToken,
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("Update", context.Background(), mock.Anything).Return(tc.response, tc.err)
		expectedGroup, err := svc.UpdateGroup(context.Background(), tc.token, tc.group)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, expectedGroup, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, expectedGroup))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestViewGroup(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	group.ID = testsutil.GenerateUUID(t, idProvider)

	cases := []struct {
		desc     string
		token    string
		groupID  string
		response mfgroups.Group
		err      error
	}{
		{
			desc:     "view group",
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			groupID:  group.ID,
			response: group,
			err:      nil,
		},
		{
			desc:     "view group with invalid token",
			token:    "wrongtoken",
			groupID:  group.ID,
			response: mfgroups.Group{},
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "view group for wrong id",
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			groupID:  mocks.WrongID,
			response: mfgroups.Group{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("RetrieveByID", context.Background(), tc.groupID).Return(tc.response, tc.err)
		expected, err := svc.ViewGroup(context.Background(), tc.token, tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, expected, tc.response, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, tc.response))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.groupID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListGroups(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	nGroups := uint64(200)
	parentID := ""
	aGroups := []mfgroups.Group{}
	for i := uint64(0); i < nGroups; i++ {
		group := mfgroups.Group{
			ID:          testsutil.GenerateUUID(t, idProvider),
			Name:        fmt.Sprintf("Group_%d", i),
			Description: description,
			Metadata: mfclients.Metadata{
				"field": "value",
			},
			Parent: parentID,
		}
		parentID = group.ID
		aGroups = append(aGroups, group)
	}

	cases := []struct {
		desc     string
		token    string
		size     uint64
		response mfgroups.GroupsPage
		page     mfgroups.GroupsPage
		err      error
	}{
		{
			desc:  "list all groups",
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			size:  nGroups,
			err:   nil,
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Offset: 0,
					Total:  nGroups,
					Limit:  nGroups,
				},
			},
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Offset: 0,
					Total:  nGroups,
					Limit:  nGroups,
				},
				Groups: aGroups,
			},
		},
		{
			desc:  "list groups with an offset",
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			size:  150,
			err:   nil,
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Offset: 50,
					Total:  nGroups,
					Limit:  nGroups,
				},
			},
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Offset: 0,
					Total:  150,
					Limit:  nGroups,
				},
				Groups: aGroups[50:nGroups],
			},
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, tc.err)
		page, err := svc.ListGroups(context.Background(), tc.token, tc.page)
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableGroup(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	enabledGroup1 := mfgroups.Group{ID: testsutil.GenerateUUID(t, idProvider), Name: "group1", Status: mfclients.EnabledStatus}
	disabledGroup := mfgroups.Group{ID: testsutil.GenerateUUID(t, idProvider), Name: "group2", Status: mfclients.DisabledStatus}
	disabledGroup1 := disabledGroup
	disabledGroup1.Status = mfclients.EnabledStatus

	casesEnabled := []struct {
		desc     string
		id       string
		token    string
		group    mfgroups.Group
		response mfgroups.Group
		err      error
	}{
		{
			desc:     "enable disabled group",
			id:       disabledGroup.ID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			group:    disabledGroup,
			response: disabledGroup1,
			err:      nil,
		},
		{
			desc:     "enable enabled group",
			id:       enabledGroup1.ID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			group:    enabledGroup1,
			response: enabledGroup1,
			err:      mfclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "enable non-existing group",
			id:       mocks.WrongID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			group:    mfgroups.Group{},
			response: mfgroups.Group{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range casesEnabled {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.group, tc.err)
		repoCall2 := gRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.EnableGroup(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}

	casesDisabled := []struct {
		desc     string
		status   mfclients.Status
		size     uint64
		response mfgroups.GroupsPage
	}{
		{
			desc:   "list activated groups",
			status: mfclients.EnabledStatus,
			size:   2,
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Groups: []mfgroups.Group{enabledGroup1, disabledGroup1},
			},
		},
		{
			desc:   "list deactivated groups",
			status: mfclients.DisabledStatus,
			size:   1,
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Groups: []mfgroups.Group{disabledGroup},
			},
		},
		{
			desc:   "list activated and deactivated groups",
			status: mfclients.AllStatus,
			size:   3,
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Groups: []mfgroups.Group{enabledGroup1, disabledGroup, disabledGroup1},
			},
		},
	}

	for _, tc := range casesDisabled {
		pm := mfgroups.GroupsPage{
			Page: mfgroups.Page{
				Offset: 0,
				Limit:  100,
				Status: tc.status,
			},
		}
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListGroups(context.Background(), testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher), pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDisableGroup(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	enabledGroup1 := mfgroups.Group{ID: testsutil.GenerateUUID(t, idProvider), Name: "group1", Status: mfclients.EnabledStatus}
	disabledGroup := mfgroups.Group{ID: testsutil.GenerateUUID(t, idProvider), Name: "group2", Status: mfclients.DisabledStatus}
	disabledGroup1 := enabledGroup1
	disabledGroup1.Status = mfclients.DisabledStatus

	casesDisabled := []struct {
		desc     string
		id       string
		token    string
		group    mfgroups.Group
		response mfgroups.Group
		err      error
	}{
		{
			desc:     "disable enabled group",
			id:       enabledGroup1.ID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			group:    enabledGroup1,
			response: disabledGroup1,
			err:      nil,
		},
		{
			desc:     "disable disabled group",
			id:       disabledGroup.ID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			group:    disabledGroup,
			response: mfgroups.Group{},
			err:      mfclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "disable non-existing group",
			id:       mocks.WrongID,
			group:    mfgroups.Group{},
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			response: mfgroups.Group{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range casesDisabled {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.group, tc.err)
		repoCall2 := gRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.DisableGroup(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}

	casesEnabled := []struct {
		desc     string
		status   mfclients.Status
		size     uint64
		response mfgroups.GroupsPage
	}{
		{
			desc:   "list activated groups",
			status: mfclients.EnabledStatus,
			size:   1,
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Groups: []mfgroups.Group{enabledGroup1},
			},
		},
		{
			desc:   "list deactivated groups",
			status: mfclients.DisabledStatus,
			size:   2,
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Groups: []mfgroups.Group{disabledGroup1, disabledGroup},
			},
		},
		{
			desc:   "list activated and deactivated groups",
			status: mfclients.AllStatus,
			size:   3,
			response: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Groups: []mfgroups.Group{enabledGroup1, disabledGroup, disabledGroup1},
			},
		},
	}

	for _, tc := range casesEnabled {
		pm := mfgroups.GroupsPage{
			Page: mfgroups.Page{
				Offset: 0,
				Limit:  100,
				Status: tc.status,
			},
		}
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListGroups(context.Background(), testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher), pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Groups))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListMemberships(t *testing.T) {
	cRepo := new(cmocks.Repository)
	gRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := cmocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)

	nGroups := uint64(100)
	aGroups := []mfgroups.Group{}
	owner := testsutil.GenerateUUID(t, idProvider)
	for i := uint64(1); i < nGroups; i++ {
		group := mfgroups.Group{
			Name:     fmt.Sprintf("membership_%d@example.com", i),
			Metadata: mfclients.Metadata{"role": "group"},
		}
		if i%3 == 0 {
			group.Owner = owner
		}
		aGroups = append(aGroups, group)
	}
	validID := testsutil.GenerateUUID(t, idProvider)
	validToken := testsutil.GenerateValidToken(t, validID, csvc, cRepo, phasher)

	cases := []struct {
		desc     string
		token    string
		clientID string
		page     mfgroups.GroupsPage
		response mfgroups.MembershipsPage
		err      error
	}{
		{
			desc:     "list clients with authorized token",
			token:    validToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Action:  "g_list",
					Subject: validID,
				},
			},
			response: mfgroups.MembershipsPage{
				Page: mfgroups.Page{
					Total:  nGroups,
					Offset: 0,
					Limit:  0,
				},
				Memberships: aGroups,
			},
			err: nil,
		},
		{
			desc:     "list clients with offset and limit",
			token:    validToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Offset:  6,
					Total:   nGroups,
					Limit:   nGroups,
					Status:  mfclients.AllStatus,
					Subject: validID,
					Action:  "g_list",
				},
			},
			response: mfgroups.MembershipsPage{
				Page: mfgroups.Page{
					Total: nGroups - 6,
				},
				Memberships: aGroups[6:nGroups],
			},
		},
		{
			desc:     "list clients with an invalid token",
			token:    inValidToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Action:  "g_list",
					Subject: validID,
				},
			},
			response: mfgroups.MembershipsPage{
				Page: mfgroups.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: errors.ErrAuthentication,
		},
		{
			desc:     "list clients with an invalid id",
			token:    validToken,
			clientID: mocks.WrongID,
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Action:  "g_list",
					Subject: validID,
				},
			},
			response: mfgroups.MembershipsPage{
				Page: mfgroups.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc:     "list clients with an owner",
			token:    validToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: mfgroups.GroupsPage{
				Page: mfgroups.Page{
					Offset:  0,
					Total:   nGroups,
					Limit:   nGroups,
					Status:  mfclients.AllStatus,
					Subject: owner,
					Action:  "g_list",
				},
			},
			response: mfgroups.MembershipsPage{
				Page: mfgroups.Page{
					Total: 4,
				},
				Memberships: []mfgroups.Group{aGroups[0], aGroups[3], aGroups[6], aGroups[9]},
			},
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := gRepo.On("Memberships", context.Background(), tc.clientID, tc.page).Return(tc.response, tc.err)
		page, err := svc.ListMemberships(context.Background(), tc.token, tc.clientID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Memberships", context.Background(), tc.clientID, tc.page)
			assert.True(t, ok, fmt.Sprintf("Memberships was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}
