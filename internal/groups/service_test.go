// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/auth"
	pauth "github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/groups/mocks"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider = uuid.New()
	namegen    = namegenerator.NewGenerator()
	validGroup = mggroups.Group{
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Status: clients.Status(groups.EnabledStatus),
	}
	allowedIDs = []string{
		testsutil.GenerateUUID(&testing.T{}),
		testsutil.GenerateUUID(&testing.T{}),
		testsutil.GenerateUUID(&testing.T{}),
	}
	validID = testsutil.GenerateUUID(&testing.T{})
)

func TestCreateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc         string
		session      auth.Session
		kind         string
		group        mggroups.Group
		repoResp     mggroups.Group
		repoErr      error
		addPolErr    error
		deletePolErr error
		err          error
	}{
		{
			desc:    "successfully",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			kind:    policysvc.NewGroupKind,
			group:   validGroup,
			repoResp: mggroups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:    "with invalid status",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			kind:    policysvc.NewGroupKind,
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      clients.Status(100),
			},
			err: svcerr.ErrInvalidStatus,
		},
		{
			desc:    "successfully with parent",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			kind:    policysvc.NewGroupKind,
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      clients.Status(groups.EnabledStatus),
				Parent:      testsutil.GenerateUUID(t),
			},
			repoResp: mggroups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    testsutil.GenerateUUID(t),
				Parent:    testsutil.GenerateUUID(t),
			},
		},
		{
			desc:     "with repo error",
			session:  auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			kind:     policysvc.NewGroupKind,
			group:    validGroup,
			repoResp: mggroups.Group{},
			repoErr:  errors.ErrMalformedEntity,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:    "with failed to add policies",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			kind:    policysvc.NewGroupKind,
			group:   validGroup,
			repoResp: mggroups.Group{
				ID: testsutil.GenerateUUID(t),
			},
			addPolErr: svcerr.ErrAuthorization,
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:    "with failed to delete policies response",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			kind:    policysvc.NewGroupKind,
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      clients.Status(groups.EnabledStatus),
				Parent:      testsutil.GenerateUUID(t),
			},
			repoErr:      errors.ErrMalformedEntity,
			deletePolErr: svcerr.ErrAuthorization,
			err:          errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPolErr)
			policyCall1 := policies.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolErr)
			got, err := svc.CreateGroup(context.Background(), tc.session, tc.kind, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got.ID)
				assert.NotEmpty(t, got.CreatedAt)
				assert.NotEmpty(t, got.Domain)
				assert.WithinDuration(t, time.Now(), got.CreatedAt, 2*time.Second)
				ok := repoCall.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
			}
			repoCall.Unset()
			policyCall.Unset()
			policyCall1.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc     string
		id       string
		repoResp mggroups.Group
		repoErr  error
		err      error
	}{
		{
			desc:     "successfully",
			id:       testsutil.GenerateUUID(t),
			repoResp: validGroup,
		},
		{
			desc:    "with repo error",
			id:      testsutil.GenerateUUID(t),
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.repoResp, tc.repoErr)
			got, err := svc.ViewGroup(context.Background(), pauth.Session{}, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			repoCall.Unset()
		})
	}
}

func TestViewGroupPerms(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc     string
		session  auth.Session
		id       string
		listResp policysvc.Permissions
		listErr  error
		err      error
	}{
		{
			desc:    "successfully",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:      testsutil.GenerateUUID(t),
			listResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:    "with failed to list permissions",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:      testsutil.GenerateUUID(t),
			listErr: svcerr.ErrAuthorization,
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:     "with empty permissions",
			session:  auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:       testsutil.GenerateUUID(t),
			listResp: []string{},
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			policyCall := policies.On("ListPermissions", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.UserType,
				Subject:     validID,
				Object:      tc.id,
				ObjectType:  policysvc.GroupType,
			}, []string{}).Return(tc.listResp, tc.listErr)
			got, err := svc.ViewGroupPerms(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.ElementsMatch(t, tc.listResp, got)
			}
			policyCall.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc     string
		session  auth.Session
		group    mggroups.Group
		repoResp mggroups.Group
		repoErr  error
		err      error
	}{
		{
			desc:    "successfully",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			group: mggroups.Group{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			repoResp: validGroup,
		},
		{
			desc:    " with repo error",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			group: mggroups.Group{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("Update", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			got, err := svc.UpdateGroup(context.Background(), tc.session, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "Update", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
			}
			repoCall.Unset()
		})
	}
}

func TestEnableGroup(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc         string
		session      auth.Session
		id           string
		retrieveResp mggroups.Group
		retrieveErr  error
		changeResp   mggroups.Group
		changeErr    error
		err          error
	}{
		{
			desc:    "successfully",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:      testsutil.GenerateUUID(t),
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.DisabledStatus),
			},
			changeResp: validGroup,
		},
		{
			desc:    "with enabled group",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:      testsutil.GenerateUUID(t),
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.EnabledStatus),
			},
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:         "with retrieve error",
			session:      auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:           testsutil.GenerateUUID(t),
			retrieveResp: mggroups.Group{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.EnableGroup(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.changeResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestDisableGroup(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc         string
		session      auth.Session
		id           string
		retrieveResp mggroups.Group
		retrieveErr  error
		changeResp   mggroups.Group
		changeErr    error
		err          error
	}{
		{
			desc:    "successfully",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:      testsutil.GenerateUUID(t),
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.EnabledStatus),
			},
			changeResp: validGroup,
		},
		{
			desc:    "with enabled group",
			session: auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:      testsutil.GenerateUUID(t),
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.DisabledStatus),
			},
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:         "with retrieve error",
			session:      auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			id:           testsutil.GenerateUUID(t),
			retrieveResp: mggroups.Group{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.DisableGroup(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.changeResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestListMembers(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc            string
		groupID         string
		permission      string
		memberKind      string
		listSubjectResp policysvc.PolicyPage
		listSubjectErr  error
		listObjectResp  policysvc.PolicyPage
		listObjectErr   error
		err             error
	}{
		{
			desc:       "successfully with things kind",
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			listObjectResp: policysvc.PolicyPage{
				Policies: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
		},
		{
			desc:       "successfully with users kind",
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			permission: policysvc.ViewPermission,
			listSubjectResp: policysvc.PolicyPage{
				Policies: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
		},
		{
			desc:       "with invalid kind",
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			permission: policysvc.ViewPermission,
			err:        errors.New("invalid member kind"),
		},
		{
			desc:           "failed to list objects with things kind",
			groupID:        testsutil.GenerateUUID(t),
			memberKind:     policysvc.ThingsKind,
			listObjectResp: policysvc.PolicyPage{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:            "failed to list subjects with users kind",
			groupID:         testsutil.GenerateUUID(t),
			memberKind:      policysvc.UsersKind,
			permission:      policysvc.ViewPermission,
			listSubjectResp: policysvc.PolicyPage{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			policyCall := policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.GroupType,
				Subject:     tc.groupID,
				Relation:    policysvc.GroupRelation,
				ObjectType:  policysvc.ThingType,
			}).Return(tc.listObjectResp, tc.listObjectErr)
			policyCall1 := policies.On("ListAllSubjects", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.UserType,
				Permission:  tc.permission,
				Object:      tc.groupID,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.listSubjectResp, tc.listSubjectErr)
			got, err := svc.ListMembers(context.Background(), pauth.Session{}, tc.groupID, tc.permission, tc.memberKind)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			policyCall.Unset()
			policyCall1.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc                 string
		session              auth.Session
		memberKind           string
		memberID             string
		page                 mggroups.Page
		listSubjectResp      policysvc.PolicyPage
		listSubjectErr       error
		listObjectResp       policysvc.PolicyPage
		listObjectErr        error
		listObjectFilterResp policysvc.PolicyPage
		listObjectFilterErr  error
		repoResp             mggroups.Page
		repoErr              error
		listPermResp         policysvc.Permissions
		listPermErr          error
		err                  error
	}{
		{
			desc:       "successfully with things kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:       "successfully with groups kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:       "successfully with channels kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:       "successfully with users kind non admin",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:       "successfully with users kind admin",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list subjects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp: policysvc.PolicyPage{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list filtered objects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to list subjects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp: policysvc.PolicyPage{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to list filtered objects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to list subjects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp: policysvc.PolicyPage{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to list filtered objects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to list subjects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp: policysvc.PolicyPage{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to list filtered objects",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "successfully with users kind admin",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				policysvc.ViewPermission,
				policysvc.EditPermission,
			},
		},
		{
			desc:       "unsuccessfully with invalid kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: "invalid",
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			err: errors.New("invalid member kind"),
		},
		{
			desc:       "unsuccessfully with things kind due to repo error",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp:             mggroups.Page{},
			repoErr:              repoerr.ErrViewEntity,
			err:                  repoerr.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list permissions",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{},
			listPermErr:  svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			policyCall := &mock.Call{}
			policyCall1 := &mock.Call{}
			switch tc.memberKind {
			case policysvc.ThingsKind:
				policyCall = policies.On("ListAllSubjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.GroupType,
					Permission:  policysvc.GroupRelation,
					ObjectType:  policysvc.ThingType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				policyCall1 = policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     validID,
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case policysvc.GroupsKind:
				policyCall = policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.GroupType,
					Subject:     tc.memberID,
					Permission:  policysvc.ParentGroupRelation,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				policyCall1 = policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     validID,
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case policysvc.ChannelsKind:
				policyCall = policies.On("ListAllSubjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.GroupType,
					Permission:  policysvc.ParentGroupRelation,
					ObjectType:  policysvc.GroupType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				policyCall1 = policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     validID,
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case policysvc.UsersKind:
				policyCall = policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     mgauth.EncodeDomainUserID(validID, tc.memberID),
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				policyCall1 = policies.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     validID,
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			}
			repoCall := repo.On("RetrieveByIDs", context.Background(), mock.Anything, mock.Anything).Return(tc.repoResp, tc.repoErr)
			policyCall2 := policies.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermResp, tc.listPermErr)
			got, err := svc.ListGroups(context.Background(), tc.session, tc.memberKind, tc.memberID, tc.page)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			repoCall.Unset()
			switch tc.memberKind {
			case policysvc.ThingsKind, policysvc.GroupsKind, policysvc.ChannelsKind, policysvc.UsersKind:
				policyCall.Unset()
				policyCall1.Unset()
				policyCall2.Unset()
			}
		})
	}
}

func TestAssign(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc                    string
		session                 auth.Session
		groupID                 string
		relation                string
		memberKind              string
		memberIDs               []string
		addPoliciesErr          error
		repoResp                mggroups.Page
		repoErr                 error
		addParentPoliciesErr    error
		deleteParentPoliciesErr error
		repoParentGroupErr      error
		err                     error
	}{
		{
			desc:       "successfully with things kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
			memberIDs:  allowedIDs,
			err:        nil,
		},
		{
			desc:       "successfully with channels kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ChannelsKind,
			memberIDs:  allowedIDs,
			err:        nil,
		},
		{
			desc:       "successfully with groups kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.UsersKind,
			memberIDs:  allowedIDs,
			err:        nil,
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp:   mggroups.Page{},
			repoErr:    repoerr.ErrViewEntity,
			err:        repoerr.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with groups kind due to empty page",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{},
			},
			err: errors.New("invalid group ids"),
		},
		{
			desc:       "unsuccessfully with groups kind due to non empty parent",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					{
						ID:     testsutil.GenerateUUID(t),
						Parent: testsutil.GenerateUUID(t),
					},
				},
			},
			err: repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to add policies",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			repoParentGroupErr: repoerr.ErrConflict,
			err:                repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent and delete policies",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			deleteParentPoliciesErr: svcerr.ErrAuthorization,
			repoParentGroupErr:      repoerr.ErrConflict,
			err:                     apiutil.ErrRollbackTx,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: "invalid",
			memberIDs:  allowedIDs,
			err:        errors.New("invalid member kind"),
		},
		{
			desc:           "unsuccessfully with failed to add policies",
			session:        auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:        testsutil.GenerateUUID(t),
			relation:       policysvc.ContributorRelation,
			memberKind:     policysvc.ThingsKind,
			memberIDs:      allowedIDs,
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			retrieveByIDsCall := &mock.Call{}
			deletePoliciesCall := &mock.Call{}
			assignParentCall := &mock.Call{}
			policyList := []policysvc.PolicyReq{}
			switch tc.memberKind {
			case policysvc.ThingsKind:
				for _, memberID := range tc.memberIDs {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.GroupType,
						SubjectKind: policysvc.ChannelsKind,
						Subject:     tc.groupID,
						Relation:    tc.relation,
						ObjectType:  policysvc.ThingType,
						Object:      memberID,
					})
				}
			case policysvc.GroupsKind:
				retrieveByIDsCall = repo.On("RetrieveByIDs", context.Background(), mggroups.Page{PageMeta: mggroups.PageMeta{Limit: 1<<63 - 1}}, mock.Anything).Return(tc.repoResp, tc.repoErr)
				for _, group := range tc.repoResp.Groups {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.GroupType,
						Subject:     tc.groupID,
						Relation:    policysvc.ParentGroupRelation,
						ObjectType:  policysvc.GroupType,
						Object:      group.ID,
					})
				}
				deletePoliciesCall = policies.On("DeletePolicies", context.Background(), policyList).Return(tc.deleteParentPoliciesErr)
				assignParentCall = repo.On("AssignParentGroup", context.Background(), tc.groupID, tc.memberIDs).Return(tc.repoParentGroupErr)
			case policysvc.ChannelsKind:
				for _, memberID := range tc.memberIDs {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.GroupType,
						Subject:     memberID,
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			case policysvc.UsersKind:
				for _, memberID := range tc.memberIDs {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.UserType,
						Subject:     mgauth.EncodeDomainUserID(validID, memberID),
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			}
			policyCall := policies.On("AddPolicies", context.Background(), policyList).Return(tc.addPoliciesErr)
			err := svc.Assign(context.Background(), tc.session, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			policyCall.Unset()
			if tc.memberKind == policysvc.GroupsKind {
				retrieveByIDsCall.Unset()
				deletePoliciesCall.Unset()
				assignParentCall.Unset()
			}
		})
	}
}

func TestUnassign(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc                    string
		session                 auth.Session
		groupID                 string
		relation                string
		memberKind              string
		memberIDs               []string
		deletePoliciesErr       error
		repoResp                mggroups.Page
		repoErr                 error
		addParentPoliciesErr    error
		deleteParentPoliciesErr error
		repoParentGroupErr      error
		err                     error
	}{
		{
			desc:       "successfully with things kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
			memberIDs:  allowedIDs,
			err:        nil,
		},
		{
			desc:       "successfully with channels kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ChannelsKind,
			memberIDs:  allowedIDs,
			err:        nil,
		},
		{
			desc:       "successfully with groups kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.UsersKind,
			memberIDs:  allowedIDs,
			err:        nil,
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp:   mggroups.Page{},
			repoErr:    repoerr.ErrViewEntity,
			err:        repoerr.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with groups kind due to empty page",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{},
			},
			err: errors.New("invalid group ids"),
		},
		{
			desc:       "unsuccessfully with groups kind due to non empty parent",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					{
						ID:     testsutil.GenerateUUID(t),
						Parent: testsutil.GenerateUUID(t),
					},
				},
			},
			err: repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to add policies",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			repoParentGroupErr: repoerr.ErrConflict,
			err:                repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent and add policies",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
			memberIDs:  allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			repoParentGroupErr:   repoerr.ErrConflict,
			addParentPoliciesErr: svcerr.ErrAuthorization,
			err:                  repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			session:    auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: "invalid",
			memberIDs:  allowedIDs,
			err:        errors.New("invalid member kind"),
		},
		{
			desc:              "unsuccessfully with failed to add policies",
			session:           auth.Session{UserID: validID, DomainID: validID, DomainUserID: validID},
			groupID:           testsutil.GenerateUUID(t),
			relation:          policysvc.ContributorRelation,
			memberKind:        policysvc.ThingsKind,
			memberIDs:         allowedIDs,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			retrieveByIDsCall := &mock.Call{}
			addPoliciesCall := &mock.Call{}
			assignParentCall := &mock.Call{}
			policyList := []policysvc.PolicyReq{}
			switch tc.memberKind {
			case policysvc.ThingsKind:
				for _, memberID := range tc.memberIDs {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.GroupType,
						SubjectKind: policysvc.ChannelsKind,
						Subject:     tc.groupID,
						Relation:    tc.relation,
						ObjectType:  policysvc.ThingType,
						Object:      memberID,
					})
				}
			case policysvc.GroupsKind:
				retrieveByIDsCall = repo.On("RetrieveByIDs", context.Background(), mggroups.Page{PageMeta: mggroups.PageMeta{Limit: 1<<63 - 1}}, mock.Anything).Return(tc.repoResp, tc.repoErr)
				for _, group := range tc.repoResp.Groups {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.GroupType,
						Subject:     tc.groupID,
						Relation:    policysvc.ParentGroupRelation,
						ObjectType:  policysvc.GroupType,
						Object:      group.ID,
					})
				}
				addPoliciesCall = policies.On("AddPolicies", context.Background(), policyList).Return(tc.addParentPoliciesErr)
				assignParentCall = repo.On("UnassignParentGroup", context.Background(), tc.groupID, tc.memberIDs).Return(tc.repoParentGroupErr)
			case policysvc.ChannelsKind:
				for _, memberID := range tc.memberIDs {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.GroupType,
						Subject:     memberID,
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			case policysvc.UsersKind:
				for _, memberID := range tc.memberIDs {
					policyList = append(policyList, policysvc.PolicyReq{
						Domain:      validID,
						SubjectType: policysvc.UserType,
						Subject:     mgauth.EncodeDomainUserID(validID, memberID),
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			}
			policyCall := policies.On("DeletePolicies", context.Background(), policyList).Return(tc.deletePoliciesErr)
			err := svc.Unassign(context.Background(), tc.session, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			policyCall.Unset()
			if tc.memberKind == policysvc.GroupsKind {
				retrieveByIDsCall.Unset()
				addPoliciesCall.Unset()
				assignParentCall.Unset()
			}
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	repo := new(mocks.Repository)
	policies := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, policies)

	cases := []struct {
		desc                     string
		groupID                  string
		deleteSubjectPoliciesErr error
		deleteObjectPoliciesErr  error
		repoErr                  error
		err                      error
	}{
		{
			desc:    "successfully",
			groupID: testsutil.GenerateUUID(t),
			err:     nil,
		},
		{
			desc:                     "unsuccessfully with failed to remove subject policies",
			groupID:                  testsutil.GenerateUUID(t),
			deleteSubjectPoliciesErr: svcerr.ErrAuthorization,
			err:                      svcerr.ErrAuthorization,
		},
		{
			desc:                    "unsuccessfully with failed to remove object policies",
			groupID:                 testsutil.GenerateUUID(t),
			deleteObjectPoliciesErr: svcerr.ErrAuthorization,
			err:                     svcerr.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with repo err",
			groupID: testsutil.GenerateUUID(t),
			repoErr: repoerr.ErrNotFound,
			err:     repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			policyCall := policies.On("DeletePolicyFilter", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.GroupType,
				Subject:     tc.groupID,
			}).Return(tc.deleteSubjectPoliciesErr)
			policyCall2 := policies.On("DeletePolicyFilter", context.Background(), policysvc.PolicyReq{
				ObjectType: policysvc.GroupType,
				Object:     tc.groupID,
			}).Return(tc.deleteObjectPoliciesErr)
			repoCall := repo.On("Delete", context.Background(), tc.groupID).Return(tc.repoErr)
			err := svc.DeleteGroup(context.Background(), pauth.Session{}, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			policyCall.Unset()
			policyCall2.Unset()
			repoCall.Unset()
		})
	}
}
