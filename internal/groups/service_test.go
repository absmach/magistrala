// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/clients"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/groups/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider = uuid.New()
	token      = "token"
	namegen    = namegenerator.NewNameGenerator()
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
)

func TestCreateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc         string
		token        string
		kind         string
		group        mggroups.Group
		idResp       *magistrala.IdentityRes
		idErr        error
		authzResp    *magistrala.AuthorizeRes
		authzErr     error
		authzTknResp *magistrala.AuthorizeRes
		authzTknErr  error
		repoResp     mggroups.Group
		repoErr      error
		addPolResp   *magistrala.AddPoliciesRes
		addPolErr    error
		err          error
	}{
		{
			desc:  "successfully",
			token: token,
			kind:  auth.NewGroupKind,
			group: validGroup,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			authzTknResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Owner:     testsutil.GenerateUUID(t),
			},
			addPolResp: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
		},
		{
			desc:   "with invalid token",
			token:  token,
			kind:   auth.NewGroupKind,
			group:  validGroup,
			idResp: &magistrala.IdentityRes{},
			idErr:  errors.ErrAuthentication,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "with empty id or domain id but with no grpc error",
			token:  token,
			kind:   auth.NewGroupKind,
			group:  validGroup,
			idResp: &magistrala.IdentityRes{},
			idErr:  nil,
			err:    errors.ErrDomainAuthorization,
		},
		{
			desc:  "with failed to authorize domain membership",
			token: token,
			kind:  auth.NewGroupKind,
			group: validGroup,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			err: errors.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize domain membership with grpc error",
			token: token,
			kind:  auth.NewGroupKind,
			group: validGroup,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with invalid status",
			token: token,
			kind:  auth.NewGroupKind,
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      clients.Status(100),
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			err: apiutil.ErrInvalidStatus,
		},
		{
			desc:  "successfully with parent",
			token: token,
			kind:  auth.NewGroupKind,
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      clients.Status(groups.EnabledStatus),
				Parent:      testsutil.GenerateUUID(t),
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			authzTknResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Owner:     testsutil.GenerateUUID(t),
				Parent:    testsutil.GenerateUUID(t),
			},
			addPolResp: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
		},
		{
			desc:  "unsuccessfully with parent due to authorization error",
			token: token,
			kind:  auth.NewGroupKind,
			group: mggroups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      clients.Status(groups.EnabledStatus),
				Parent:      testsutil.GenerateUUID(t),
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			authzTknResp: &magistrala.AuthorizeRes{},
			authzTknErr:  errors.ErrAuthorization,
			repoResp: mggroups.Group{
				ID:     testsutil.GenerateUUID(t),
				Parent: testsutil.GenerateUUID(t),
			},
			addPolResp: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
			err: errors.ErrAuthorization,
		},
		{
			desc:  "with repo error",
			token: token,
			kind:  auth.NewGroupKind,
			group: validGroup,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			authzTknResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Group{},
			repoErr:  errors.ErrMalformedEntity,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:  "with failed to add policies",
			token: token,
			kind:  auth.NewGroupKind,
			group: validGroup,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			authzTknResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Group{
				ID: testsutil.GenerateUUID(t),
			},
			addPolResp: &magistrala.AddPoliciesRes{},
			addPolErr:  errors.ErrAuthorization,
			err:        errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			repocall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  auth.MembershipPermission,
				Object:      tc.idResp.GetDomainId(),
				ObjectType:  auth.DomainType,
			}).Return(tc.authzResp, tc.authzErr)
			repocall2 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.group.Parent,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzTknResp, tc.authzTknErr)
			repocall3 := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			repocall4 := authsvc.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPolResp, tc.addPolErr)
			got, err := svc.CreateGroup(context.Background(), tc.token, tc.kind, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got.ID)
				assert.NotEmpty(t, got.CreatedAt)
				assert.NotEmpty(t, got.Owner)
				assert.WithinDuration(t, time.Now(), got.CreatedAt, 2*time.Second)
				ok := repocall3.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
			}
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
			repocall3.Unset()
			repocall4.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc      string
		token     string
		id        string
		authzResp *magistrala.AuthorizeRes
		authzErr  error
		repoResp  mggroups.Group
		repoErr   error
		err       error
	}{
		{
			desc:  "successfully",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: validGroup,
		},
		{
			desc:  "with invalid token",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.ViewPermission,
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.repoResp, tc.repoErr)
			got, err := svc.ViewGroup(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
		})
	}
}

func TestViewGroupPerms(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc     string
		token    string
		id       string
		idResp   *magistrala.IdentityRes
		idErr    error
		listResp *magistrala.ListPermissionsRes
		listErr  error
		err      error
	}{
		{
			desc:  "successfully",
			token: token,
			id:    testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			listResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:   "with invalid token",
			token:  token,
			id:     testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{},
			idErr:  errors.ErrAuthentication,
			err:    errors.ErrAuthentication,
		},
		{
			desc:  "with failed to list permissions",
			token: token,
			id:    testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			listErr: errors.ErrAuthorization,
			err:     errors.ErrAuthorization,
		},
		{
			desc:  "with empty permissions",
			token: token,
			id:    testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			listResp: &magistrala.ListPermissionsRes{
				Permissions: []string{},
			},
			err: errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			repocall1 := authsvc.On("ListPermissions", context.Background(), &magistrala.ListPermissionsReq{
				SubjectType: auth.UserType,
				Subject:     tc.idResp.GetId(),
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.listResp, tc.listErr)
			got, err := svc.ViewGroupPerms(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.listResp.Permissions, got)
			}
			repocall.Unset()
			repocall1.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc      string
		token     string
		group     mggroups.Group
		authzResp *magistrala.AuthorizeRes
		authzErr  error
		repoResp  mggroups.Group
		repoErr   error
		err       error
	}{
		{
			desc:  "successfully",
			token: token,
			group: mggroups.Group{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: validGroup,
		},
		{
			desc:  "with invalid token",
			token: token,
			group: mggroups.Group{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			group: mggroups.Group{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.group.ID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repo.On("Update", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			got, err := svc.UpdateGroup(context.Background(), tc.token, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "Update", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
			}
		})
	}
}

func TestEnableGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc         string
		token        string
		id           string
		authzResp    *magistrala.AuthorizeRes
		authzErr     error
		retrieveResp mggroups.Group
		retrieveErr  error
		changeResp   mggroups.Group
		changeErr    error
		err          error
	}{
		{
			desc:  "successfully",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.DisabledStatus),
			},
			changeResp: validGroup,
		},
		{
			desc:  "with invalid token",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with enabled group",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.EnabledStatus),
			},
			err: mgclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:  "with retrieve error",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{},
			retrieveErr:  errors.ErrNotFound,
			err:          errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repocall1 := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repocall2 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.EnableGroup(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.changeResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
		})
	}
}

func TestDisableGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc         string
		token        string
		id           string
		authzResp    *magistrala.AuthorizeRes
		authzErr     error
		retrieveResp mggroups.Group
		retrieveErr  error
		changeResp   mggroups.Group
		changeErr    error
		err          error
	}{
		{
			desc:  "successfully",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.EnabledStatus),
			},
			changeResp: validGroup,
		},
		{
			desc:  "with invalid token",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      errors.ErrAuthorization,
		},
		{
			desc:  "with enabled group",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{
				Status: clients.Status(groups.DisabledStatus),
			},
			err: mgclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:  "with retrieve error",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{},
			retrieveErr:  errors.ErrNotFound,
			err:          errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repocall1 := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repocall2 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.DisableGroup(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.changeResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
		})
	}
}

func TestListMembers(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc            string
		token           string
		groupID         string
		permission      string
		memberKind      string
		authzResp       *magistrala.AuthorizeRes
		authzErr        error
		listSubjectResp *magistrala.ListSubjectsRes
		listSubjectErr  error
		listObjectResp  *magistrala.ListObjectsRes
		listObjectErr   error
		err             error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: auth.UsersKind,
			permission: auth.ViewPermission,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: []string{
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
					testsutil.GenerateUUID(t),
				},
			},
		},
		{
			desc:       "with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: auth.GroupsKind,
			permission: auth.ViewPermission,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			err: errors.New("invalid member kind"),
		},
		{
			desc:  "with invalid token",
			token: token,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			err: errors.ErrAuthorization,
		},
		{
			desc:       "failed to list objects with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: []string{},
			},
			listObjectErr: errors.ErrAuthorization,
			err:           errors.ErrAuthorization,
		},
		{
			desc:       "failed to list subjects with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: auth.UsersKind,
			permission: auth.ViewPermission,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: []string{},
			},
			listSubjectErr: errors.ErrAuthorization,
			err:            errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.ViewPermission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repocall1 := authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
				SubjectType: auth.GroupType,
				Subject:     tc.groupID,
				Relation:    auth.GroupRelation,
				ObjectType:  auth.ThingType,
			}).Return(tc.listObjectResp, tc.listObjectErr)
			repocall2 := authsvc.On("ListAllSubjects", context.Background(), &magistrala.ListSubjectsReq{
				SubjectType: auth.UserType,
				Permission:  tc.permission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.listSubjectResp, tc.listSubjectErr)
			got, err := svc.ListMembers(context.Background(), tc.token, tc.groupID, tc.permission, tc.memberKind)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc                 string
		token                string
		memberKind           string
		memberID             string
		page                 mggroups.Page
		idResp               *magistrala.IdentityRes
		idErr                error
		authzResp            *magistrala.AuthorizeRes
		authzErr             error
		listSubjectResp      *magistrala.ListSubjectsRes
		listSubjectErr       error
		listObjectResp       *magistrala.ListObjectsRes
		listObjectErr        error
		listObjectFilterResp *magistrala.ListObjectsRes
		listObjectFilterErr  error
		authSuperAdminResp   *magistrala.AuthorizeRes
		authSuperAdminErr    error
		repoResp             mggroups.Page
		repoErr              error
		listPermResp         *magistrala.ListPermissionsRes
		listPermErr          error
		err                  error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.GroupsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ChannelsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:       "successfully with users kind non admin",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:       "successfully with users kind admin",
			token:      token,
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				UserId:   testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:       "unsuccessfully with users kind admin",
			token:      token,
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				UserId:   testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind admin with nil error",
			token:      token,
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				UserId:   testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			err: errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list subjects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{},
			listSubjectErr:  errors.ErrAuthorization,
			err:             errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{},
			listObjectFilterErr:  errors.ErrAuthorization,
			err:                  errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.GroupsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to list subjects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.GroupsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{},
			listObjectErr:  errors.ErrAuthorization,
			err:            errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.GroupsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{},
			listObjectFilterErr:  errors.ErrAuthorization,
			err:                  errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ChannelsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to list subjects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ChannelsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{},
			listSubjectErr:  errors.ErrAuthorization,
			err:             errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ChannelsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{},
			listObjectFilterErr:  errors.ErrAuthorization,
			err:                  errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to list subjects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{},
			listObjectErr:  errors.ErrAuthorization,
			err:            errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{},
			listObjectFilterErr:  errors.ErrAuthorization,
			err:                  errors.ErrAuthorization,
		},
		{
			desc:       "successfully with users kind admin",
			token:      token,
			memberKind: auth.UsersKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				UserId:   testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{
				Permissions: []string{
					auth.ViewPermission,
					auth.EditPermission,
				},
			},
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: "invalid",
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			err: errors.New("invalid member kind"),
		},
		{
			desc:       "unsuccessfully with things kind due to repo error",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{},
			repoErr:  errors.ErrViewEntity,
			err:      errors.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list permissions",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: &magistrala.ListSubjectsRes{
				Policies: allowedIDs,
			},
			listObjectFilterResp: &magistrala.ListObjectsRes{
				Policies: allowedIDs,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: &magistrala.ListPermissionsRes{},
			listPermErr:  errors.ErrAuthorization,
			err:          errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with invalid token",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			page: mggroups.Page{
				Permission: auth.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{},
			idErr:  errors.ErrAuthentication,
			err:    errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			repocall1 := &mock.Call{}
			repocall2 := &mock.Call{}
			repocall3 := &mock.Call{}
			adminCheck := &mock.Call{}
			switch tc.memberKind {
			case auth.ThingsKind:
				repocall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  auth.ViewPermission,
					Object:      tc.memberID,
					ObjectType:  auth.ThingType,
				}).Return(tc.authzResp, tc.authzErr)
				repocall2 = authsvc.On("ListAllSubjects", context.Background(), &magistrala.ListSubjectsReq{
					SubjectType: auth.GroupType,
					Permission:  auth.GroupRelation,
					ObjectType:  auth.ThingType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				repocall3 = authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case auth.GroupsKind:
				repocall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					Object:      tc.memberID,
					ObjectType:  auth.GroupType,
				}).Return(tc.authzResp, tc.authzErr)
				repocall2 = authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.GroupType,
					Subject:     tc.memberID,
					Permission:  auth.ParentGroupRelation,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				repocall3 = authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case auth.ChannelsKind:
				repocall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  auth.ViewPermission,
					Object:      tc.memberID,
					ObjectType:  auth.GroupType,
				}).Return(tc.authzResp, tc.authzErr)
				repocall2 = authsvc.On("ListAllSubjects", context.Background(), &magistrala.ListSubjectsReq{
					SubjectType: auth.GroupType,
					Permission:  auth.ParentGroupRelation,
					ObjectType:  auth.GroupType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				repocall3 = authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case auth.UsersKind:
				adminCheckReq := &magistrala.AuthorizeReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetUserId(),
					Permission:  auth.AdminPermission,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
				}
				adminCheck = authsvc.On("Authorize", context.Background(), adminCheckReq).Return(tc.authzResp, tc.authzErr)
				authReq := &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  auth.AdminPermission,
					Object:      tc.idResp.GetDomainId(),
					ObjectType:  auth.DomainType,
				}
				if tc.memberID == "" {
					authReq.Domain = ""
					authReq.Permission = auth.MembershipPermission
				}
				repocall1 = authsvc.On("Authorize", context.Background(), authReq).Return(tc.authzResp, tc.authzErr)
				repocall2 = authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), tc.memberID),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				repocall3 = authsvc.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			}
			repocall4 := repo.On("RetrieveByIDs", context.Background(), mock.Anything, mock.Anything).Return(tc.repoResp, tc.repoErr)
			repocall5 := authsvc.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermResp, tc.listPermErr)
			got, err := svc.ListGroups(context.Background(), tc.token, tc.memberKind, tc.memberID, tc.page)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			repocall.Unset()
			switch tc.memberKind {
			case auth.ThingsKind, auth.GroupsKind, auth.ChannelsKind, auth.UsersKind:
				repocall1.Unset()
				repocall2.Unset()
				repocall3.Unset()
				if tc.memberID == "" {
					adminCheck.Unset()
				}
			}
			repocall4.Unset()
			repocall5.Unset()
		})
	}
}

func TestAssign(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc                    string
		token                   string
		groupID                 string
		relation                string
		memberKind              string
		memberIDs               []string
		idResp                  *magistrala.IdentityRes
		idErr                   error
		authzResp               *magistrala.AuthorizeRes
		authzErr                error
		addPoliciesRes          *magistrala.AddPoliciesRes
		addPoliciesErr          error
		repoResp                mggroups.Page
		repoErr                 error
		addParentPoliciesRes    *magistrala.AddPoliciesRes
		addParentPoliciesErr    error
		deleteParentPoliciesRes *magistrala.DeletePoliciesRes
		deleteParentPoliciesErr error
		repoParentGroupErr      error
		err                     error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ChannelsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{},
			repoErr:  errors.ErrViewEntity,
			err:      errors.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with groups kind due to empty page",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{},
			},
			err: errors.New("invalid group ids"),
		},
		{
			desc:       "unsuccessfully with groups kind due to non empty parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					{
						ID:     testsutil.GenerateUUID(t),
						Parent: testsutil.GenerateUUID(t),
					},
				},
			},
			err: errors.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: false,
			},
			addPoliciesErr: errors.ErrAuthorization,
			err:            errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
			repoParentGroupErr: errors.ErrConflict,
			err:                errors.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent and delete policy",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: true,
			},
			deleteParentPoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: false,
			},
			deleteParentPoliciesErr: errors.ErrAuthorization,
			repoParentGroupErr:      errors.ErrConflict,
			err:                     apiutil.ErrRollbackTx,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: "invalid",
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			err: errors.New("invalid member kind"),
		},
		{
			desc:       "unsuccessfully with invalid token",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp:     &magistrala.IdentityRes{},
			idErr:      errors.ErrAuthentication,
			err:        errors.ErrAuthentication,
		},
		{
			desc:       "unsuccessfully with failed to authorize",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: false,
			},
			addPoliciesErr: errors.ErrAuthorization,
			err:            errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			repocall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  auth.EditPermission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			retrieveByIDsCall := &mock.Call{}
			deletePoliciesCall := &mock.Call{}
			assignParentCall := &mock.Call{}
			policies := magistrala.AddPoliciesReq{}
			switch tc.memberKind {
			case auth.ThingsKind:
				for _, memberID := range tc.memberIDs {
					policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						SubjectKind: auth.ChannelsKind,
						Subject:     tc.groupID,
						Relation:    tc.relation,
						ObjectType:  auth.ThingType,
						Object:      memberID,
					})
				}
			case auth.GroupsKind:
				retrieveByIDsCall = repo.On("RetrieveByIDs", context.Background(), mggroups.Page{PageMeta: mggroups.PageMeta{Limit: 1<<63 - 1}}, mock.Anything).Return(tc.repoResp, tc.repoErr)
				var deletePolicies magistrala.DeletePoliciesReq
				for _, group := range tc.repoResp.Groups {
					policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						Subject:     tc.groupID,
						Relation:    auth.ParentGroupRelation,
						ObjectType:  auth.GroupType,
						Object:      group.ID,
					})
					deletePolicies.DeletePoliciesReq = append(deletePolicies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						Subject:     tc.groupID,
						Relation:    auth.ParentGroupRelation,
						ObjectType:  auth.GroupType,
						Object:      group.ID,
					})
				}
				deletePoliciesCall = authsvc.On("DeletePolicies", context.Background(), &deletePolicies).Return(tc.deleteParentPoliciesRes, tc.deleteParentPoliciesErr)
				assignParentCall = repo.On("AssignParentGroup", context.Background(), tc.groupID, tc.memberIDs).Return(tc.repoParentGroupErr)
			case auth.ChannelsKind:
				for _, memberID := range tc.memberIDs {
					policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						Subject:     memberID,
						Relation:    tc.relation,
						ObjectType:  auth.GroupType,
						Object:      tc.groupID,
					})
				}
			case auth.UsersKind:
				for _, memberID := range tc.memberIDs {
					policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.UserType,
						Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), memberID),
						Relation:    tc.relation,
						ObjectType:  auth.GroupType,
						Object:      tc.groupID,
					})
				}
			}
			repocall2 := authsvc.On("AddPolicies", context.Background(), &policies).Return(tc.addPoliciesRes, tc.addPoliciesErr)
			err := svc.Assign(context.Background(), tc.token, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
			if tc.memberKind == auth.GroupsKind {
				retrieveByIDsCall.Unset()
				deletePoliciesCall.Unset()
				assignParentCall.Unset()
			}
		})
	}
}

func TestUnassign(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc                    string
		token                   string
		groupID                 string
		relation                string
		memberKind              string
		memberIDs               []string
		idResp                  *magistrala.IdentityRes
		idErr                   error
		authzResp               *magistrala.AuthorizeRes
		authzErr                error
		deletePoliciesRes       *magistrala.DeletePoliciesRes
		deletePoliciesErr       error
		repoResp                mggroups.Page
		repoErr                 error
		addParentPoliciesRes    *magistrala.AddPoliciesRes
		addParentPoliciesErr    error
		deleteParentPoliciesRes *magistrala.DeletePoliciesRes
		deleteParentPoliciesErr error
		repoParentGroupErr      error
		err                     error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: true,
			},
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ChannelsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: true,
			},
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: true,
			},
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: true,
			},
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{},
			repoErr:  errors.ErrViewEntity,
			err:      errors.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with groups kind due to empty page",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{},
			},
			err: errors.New("invalid group ids"),
		},
		{
			desc:       "unsuccessfully with groups kind due to non empty parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					{
						ID:     testsutil.GenerateUUID(t),
						Parent: testsutil.GenerateUUID(t),
					},
				},
			},
			err: errors.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: false,
			},
			deletePoliciesErr: errors.ErrAuthorization,
			err:               errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: true,
			},
			repoParentGroupErr: errors.ErrConflict,
			err:                errors.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent and add policy",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.GroupsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: true,
			},
			repoParentGroupErr: errors.ErrConflict,
			addParentPoliciesRes: &magistrala.AddPoliciesRes{
				Authorized: false,
			},
			addParentPoliciesErr: errors.ErrAuthorization,
			err:                  errors.ErrConflict,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: "invalid",
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			err: errors.New("invalid member kind"),
		},
		{
			desc:       "unsuccessfully with invalid token",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp:     &magistrala.IdentityRes{},
			idErr:      errors.ErrAuthentication,
			err:        errors.ErrAuthentication,
		},
		{
			desc:       "unsuccessfully with failed to authorize",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ViewerRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{
				Deleted: false,
			},
			deletePoliciesErr: errors.ErrAuthorization,
			err:               errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			repocall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  auth.EditPermission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			retrieveByIDsCall := &mock.Call{}
			addPoliciesCall := &mock.Call{}
			assignParentCall := &mock.Call{}
			policies := magistrala.DeletePoliciesReq{}
			switch tc.memberKind {
			case auth.ThingsKind:
				for _, memberID := range tc.memberIDs {
					policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						SubjectKind: auth.ChannelsKind,
						Subject:     tc.groupID,
						Relation:    tc.relation,
						ObjectType:  auth.ThingType,
						Object:      memberID,
					})
				}
			case auth.GroupsKind:
				retrieveByIDsCall = repo.On("RetrieveByIDs", context.Background(), mggroups.Page{PageMeta: mggroups.PageMeta{Limit: 1<<63 - 1}}, mock.Anything).Return(tc.repoResp, tc.repoErr)
				var addPolicies magistrala.AddPoliciesReq
				for _, group := range tc.repoResp.Groups {
					policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						Subject:     tc.groupID,
						Relation:    auth.ParentGroupRelation,
						ObjectType:  auth.GroupType,
						Object:      group.ID,
					})
					addPolicies.AddPoliciesReq = append(addPolicies.AddPoliciesReq, &magistrala.AddPolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						Subject:     tc.groupID,
						Relation:    auth.ParentGroupRelation,
						ObjectType:  auth.GroupType,
						Object:      group.ID,
					})
				}
				addPoliciesCall = authsvc.On("AddPolicies", context.Background(), &addPolicies).Return(tc.addParentPoliciesRes, tc.addParentPoliciesErr)
				assignParentCall = repo.On("UnassignParentGroup", context.Background(), tc.groupID, tc.memberIDs).Return(tc.repoParentGroupErr)
			case auth.ChannelsKind:
				for _, memberID := range tc.memberIDs {
					policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.GroupType,
						Subject:     memberID,
						Relation:    tc.relation,
						ObjectType:  auth.GroupType,
						Object:      tc.groupID,
					})
				}
			case auth.UsersKind:
				for _, memberID := range tc.memberIDs {
					policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: auth.UserType,
						Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), memberID),
						Relation:    tc.relation,
						ObjectType:  auth.GroupType,
						Object:      tc.groupID,
					})
				}
			}
			repocall2 := authsvc.On("DeletePolicies", context.Background(), &policies).Return(tc.deletePoliciesRes, tc.deletePoliciesErr)
			err := svc.Unassign(context.Background(), tc.token, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
			if tc.memberKind == auth.GroupsKind {
				retrieveByIDsCall.Unset()
				addPoliciesCall.Unset()
				assignParentCall.Unset()
			}
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.Service)
	svc := groups.NewService(repo, idProvider, authsvc)

	cases := []struct {
		desc                     string
		token                    string
		groupID                  string
		idResp                   *magistrala.IdentityRes
		idErr                    error
		authzResp                *magistrala.AuthorizeRes
		authzErr                 error
		deleteChildPoliciesRes   *magistrala.DeletePolicyRes
		deleteChildPoliciesErr   error
		deleteThingsPoliciesRes  *magistrala.DeletePolicyRes
		deleteThingsPoliciesErr  error
		deleteDomainsPoliciesRes *magistrala.DeletePolicyRes
		deleteDomainsPoliciesErr error
		deleteUsersPoliciesRes   *magistrala.DeletePolicyRes
		deleteUsersPoliciesErr   error
		repoErr                  error
		err                      error
	}{
		{
			desc:    "successfully",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteChildPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteThingsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteDomainsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteUsersPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
		},
		{
			desc:    "unsuccessfully with invalid token",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp:  &magistrala.IdentityRes{},
			idErr:   errors.ErrAuthentication,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "unsuccessfully with authorization error",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: errors.ErrAuthorization,
			err:      errors.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove child group policy",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteChildPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: false,
			},
			deleteChildPoliciesErr: errors.ErrAuthorization,
			err:                    errors.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove things policy",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteChildPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteThingsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: false,
			},
			deleteThingsPoliciesErr: errors.ErrAuthorization,
			err:                     errors.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove domain policy",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteChildPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteThingsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteDomainsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: false,
			},
			deleteDomainsPoliciesErr: errors.ErrAuthorization,
			err:                      errors.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove user policy",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteChildPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteThingsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteDomainsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteUsersPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: false,
			},
			deleteUsersPoliciesErr: errors.ErrAuthorization,
			err:                    errors.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with repo err",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteChildPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteThingsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			deleteDomainsPoliciesRes: &magistrala.DeletePolicyRes{
				Deleted: true,
			},
			repoErr: errors.ErrNotFound,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			repocall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  auth.DeletePermission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repocall2 := authsvc.On("DeletePolicy", context.Background(), &magistrala.DeletePolicyReq{
				SubjectType: auth.GroupType,
				Subject:     tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.deleteChildPoliciesRes, tc.deleteChildPoliciesErr)
			repocall3 := authsvc.On("DeletePolicy", context.Background(), &magistrala.DeletePolicyReq{
				SubjectType: auth.GroupType,
				Subject:     tc.groupID,
				ObjectType:  auth.ThingType,
			}).Return(tc.deleteThingsPoliciesRes, tc.deleteThingsPoliciesErr)
			repocall4 := authsvc.On("DeletePolicy", context.Background(), &magistrala.DeletePolicyReq{
				SubjectType: auth.DomainType,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.deleteDomainsPoliciesRes, tc.deleteDomainsPoliciesErr)
			repocall5 := repo.On("Delete", context.Background(), tc.groupID).Return(tc.repoErr)
			repocall6 := authsvc.On("DeletePolicy", context.Background(), &magistrala.DeletePolicyReq{
				SubjectType: auth.UserType,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.deleteUsersPoliciesRes, tc.deleteUsersPoliciesErr)
			err := svc.DeleteGroup(context.Background(), tc.token, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repocall.Unset()
			repocall1.Unset()
			repocall2.Unset()
			repocall3.Unset()
			repocall4.Unset()
			repocall5.Unset()
			repocall6.Unset()
		})
	}
}
