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
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/groups/mocks"
	policymocks "github.com/absmach/magistrala/pkg/policy/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider = uuid.New()
	token      = "token"
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
)

func TestCreateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc          string
		token         string
		kind          string
		group         mggroups.Group
		idResp        *magistrala.IdentityRes
		idErr         error
		authzResp     *magistrala.AuthorizeRes
		authzErr      error
		authzTknResp  *magistrala.AuthorizeRes
		authzTknErr   error
		repoResp      mggroups.Group
		repoErr       error
		addPolResp    bool
		addPolErr     error
		deletePolResp bool
		deletePolErr  error
		err           error
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
				Domain:    testsutil.GenerateUUID(t),
			},
			addPolResp: true,
		},
		{
			desc:   "with invalid token",
			token:  token,
			kind:   auth.NewGroupKind,
			group:  validGroup,
			idResp: &magistrala.IdentityRes{},
			idErr:  svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "with empty id or domain id but with no grpc error",
			token:  token,
			kind:   auth.NewGroupKind,
			group:  validGroup,
			idResp: &magistrala.IdentityRes{},
			idErr:  nil,
			err:    svcerr.ErrDomainAuthorization,
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
			err: svcerr.ErrAuthorization,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			err: svcerr.ErrInvalidStatus,
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
				Domain:    testsutil.GenerateUUID(t),
				Parent:    testsutil.GenerateUUID(t),
			},
			addPolResp: true,
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
			authzTknErr:  svcerr.ErrAuthorization,
			repoResp: mggroups.Group{
				ID:     testsutil.GenerateUUID(t),
				Parent: testsutil.GenerateUUID(t),
			},
			addPolResp: true,
			err:        svcerr.ErrAuthorization,
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
			addPolResp: false,
			addPolErr:  svcerr.ErrAuthorization,
			err:        svcerr.ErrAuthorization,
		},
		{
			desc:  "with failed to delete policies response",
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
			repoErr:      errors.ErrMalformedEntity,
			addPolResp:   true,
			deletePolErr: svcerr.ErrAuthorization,
			err:          errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  auth.CreatePermission,
				Object:      tc.idResp.GetDomainId(),
				ObjectType:  auth.DomainType,
			}).Return(tc.authzResp, tc.authzErr)
			authCall2 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.group.Parent,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzTknResp, tc.authzTknErr)
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			authCall3 := policy.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPolResp, tc.addPolErr)
			authCall4 := policy.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolResp, tc.deletePolErr)
			got, err := svc.CreateGroup(context.Background(), tc.token, tc.kind, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got.ID)
				assert.NotEmpty(t, got.CreatedAt)
				assert.NotEmpty(t, got.Domain)
				assert.WithinDuration(t, time.Now(), got.CreatedAt, 2*time.Second)
				ok := repoCall.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
			}
			authCall.Unset()
			authCall1.Unset()
			authCall2.Unset()
			repoCall.Unset()
			authCall3.Unset()
			authCall4.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.ViewPermission,
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.repoResp, tc.repoErr)
			got, err := svc.ViewGroup(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			authCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestViewGroupPerms(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc     string
		token    string
		id       string
		idResp   *magistrala.IdentityRes
		idErr    error
		listResp []string
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
			listResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
			},
		},
		{
			desc:   "with invalid token",
			token:  token,
			id:     testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{},
			idErr:  svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "with failed to list permissions",
			token: token,
			id:    testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			listErr: svcerr.ErrAuthorization,
			err:     svcerr.ErrAuthorization,
		},
		{
			desc:  "with empty permissions",
			token: token,
			id:    testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			listResp: []string{},
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := policy.On("ListPermissions", context.Background(), &magistrala.ListPermissionsReq{
				SubjectType: auth.UserType,
				Subject:     tc.idResp.GetId(),
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.listResp, tc.listErr)
			got, err := svc.ViewGroupPerms(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.listResp, got)
			}
			authCall.Unset()
			authCall1.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.group.ID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repoCall := repo.On("Update", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			got, err := svc.UpdateGroup(context.Background(), tc.token, tc.group)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "Update", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
			}
			authCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestEnableGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      svcerr.ErrAuthorization,
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
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:  "with retrieve error",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.EnableGroup(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.changeResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			authCall.Unset()
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestDisableGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:  "with failed to authorize",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: nil,
			err:      svcerr.ErrAuthorization,
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
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:  "with retrieve error",
			token: token,
			id:    testsutil.GenerateUUID(t),
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			retrieveResp: mggroups.Group{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.EditPermission,
				Object:      tc.id,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.DisableGroup(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.changeResp, got)
				ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			}
			authCall.Unset()
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestListMembers(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc            string
		token           string
		groupID         string
		permission      string
		memberKind      string
		authzResp       *magistrala.AuthorizeRes
		authzErr        error
		listSubjectResp []string
		listSubjectErr  error
		listObjectResp  []string
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
			listObjectResp: []string{
				testsutil.GenerateUUID(t),
				testsutil.GenerateUUID(t),
				testsutil.GenerateUUID(t),
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
			listSubjectResp: []string{
				testsutil.GenerateUUID(t),
				testsutil.GenerateUUID(t),
				testsutil.GenerateUUID(t),
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
			err: svcerr.ErrAuthorization,
		},
		{
			desc:       "failed to list objects with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: auth.ThingsKind,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: []string{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
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
			listSubjectResp: []string{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Subject:     tc.token,
				Permission:  auth.ViewPermission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			authCall1 := policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
				SubjectType: auth.GroupType,
				Subject:     tc.groupID,
				Relation:    auth.GroupRelation,
				ObjectType:  auth.ThingType,
			}).Return(tc.listObjectResp, tc.listObjectErr)
			authCall2 := policy.On("ListAllSubjects", context.Background(), &magistrala.ListSubjectsReq{
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
			authCall.Unset()
			authCall1.Unset()
			authCall2.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
		listSubjectResp      []string
		listSubjectErr       error
		listObjectResp       []string
		listObjectErr        error
		listObjectFilterResp []string
		listObjectFilterErr  error
		authSuperAdminResp   *magistrala.AuthorizeRes
		authSuperAdminErr    error
		repoResp             mggroups.Page
		repoErr              error
		listPermResp         []string
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
			listSubjectResp:      allowedIDs,
			listObjectFilterResp: allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
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
			listObjectResp:       allowedIDs,
			listObjectFilterResp: allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
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
			listSubjectResp:      allowedIDs,
			listObjectFilterResp: allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
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
			listObjectResp:       allowedIDs,
			listObjectFilterResp: allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
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
			listObjectResp: allowedIDs,

			listObjectFilterResp: allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			err: svcerr.ErrAuthorization,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			listSubjectResp: []string{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
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
			listSubjectResp:      allowedIDs,
			listObjectFilterResp: []string{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			listObjectResp: []string{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
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
			listObjectResp:       allowedIDs,
			listObjectFilterResp: []string{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			listSubjectResp: []string{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
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
			listSubjectResp:      allowedIDs,
			listObjectFilterResp: []string{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
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
			listObjectResp: []string{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
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
			listObjectResp:       allowedIDs,
			listObjectFilterResp: []string{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
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
			listObjectResp:       allowedIDs,
			listObjectFilterResp: allowedIDs,
			repoResp: mggroups.Page{
				Groups: []mggroups.Group{
					validGroup,
					validGroup,
					validGroup,
				},
			},
			listPermResp: []string{
				auth.ViewPermission,
				auth.EditPermission,
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
			listSubjectResp:      allowedIDs,
			listObjectFilterResp: allowedIDs,
			repoResp:             mggroups.Page{},
			repoErr:              repoerr.ErrViewEntity,
			err:                  repoerr.ErrViewEntity,
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
			listSubjectResp:      allowedIDs,
			listObjectFilterResp: allowedIDs,
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
			idErr:  svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := &mock.Call{}
			authCall2 := &mock.Call{}
			authCall3 := &mock.Call{}
			adminCheck := &mock.Call{}
			switch tc.memberKind {
			case auth.ThingsKind:
				authCall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  auth.ViewPermission,
					Object:      tc.memberID,
					ObjectType:  auth.ThingType,
				}).Return(tc.authzResp, tc.authzErr)
				authCall2 = policy.On("ListAllSubjects", context.Background(), &magistrala.ListSubjectsReq{
					SubjectType: auth.GroupType,
					Permission:  auth.GroupRelation,
					ObjectType:  auth.ThingType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				authCall3 = policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case auth.GroupsKind:
				authCall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					Object:      tc.memberID,
					ObjectType:  auth.GroupType,
				}).Return(tc.authzResp, tc.authzErr)
				authCall2 = policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.GroupType,
					Subject:     tc.memberID,
					Permission:  auth.ParentGroupRelation,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				authCall3 = policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case auth.ChannelsKind:
				authCall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  auth.ViewPermission,
					Object:      tc.memberID,
					ObjectType:  auth.GroupType,
				}).Return(tc.authzResp, tc.authzErr)
				authCall2 = policy.On("ListAllSubjects", context.Background(), &magistrala.ListSubjectsReq{
					SubjectType: auth.GroupType,
					Permission:  auth.ParentGroupRelation,
					ObjectType:  auth.GroupType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				authCall3 = policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
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
				authCall1 = authsvc.On("Authorize", context.Background(), authReq).Return(tc.authzResp, tc.authzErr)
				authCall2 = policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), tc.memberID),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				authCall3 = policy.On("ListAllObjects", context.Background(), &magistrala.ListObjectsReq{
					SubjectType: auth.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  auth.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			}
			repoCall := repo.On("RetrieveByIDs", context.Background(), mock.Anything, mock.Anything).Return(tc.repoResp, tc.repoErr)
			authCall4 := policy.On("ListPermissions", mock.Anything, mock.Anything).Return(tc.listPermResp, tc.listPermErr)
			got, err := svc.ListGroups(context.Background(), tc.token, tc.memberKind, tc.memberID, tc.page)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			authCall.Unset()
			repoCall.Unset()
			switch tc.memberKind {
			case auth.ThingsKind, auth.GroupsKind, auth.ChannelsKind, auth.UsersKind:
				authCall1.Unset()
				authCall2.Unset()
				authCall3.Unset()
				authCall4.Unset()
				if tc.memberID == "" {
					adminCheck.Unset()
				}
			}
		})
	}
}

func TestAssign(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
		addPoliciesRes          bool
		addPoliciesErr          error
		repoResp                mggroups.Page
		repoErr                 error
		addParentPoliciesRes    bool
		addParentPoliciesErr    error
		deleteParentPoliciesRes bool
		deleteParentPoliciesErr error
		repoParentGroupErr      error
		err                     error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: true,
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ChannelsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: true,
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			addPoliciesRes:     true,
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: true,
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			repoErr:  repoerr.ErrViewEntity,
			err:      repoerr.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with groups kind due to empty page",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			relation:   auth.ContributorRelation,
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
			err: repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			addPoliciesRes: false,
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			addPoliciesRes:     true,
			repoParentGroupErr: repoerr.ErrConflict,
			err:                repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent and delete policy",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			addPoliciesRes:          true,
			deleteParentPoliciesRes: false,
			deleteParentPoliciesErr: svcerr.ErrAuthorization,
			repoParentGroupErr:      repoerr.ErrConflict,
			err:                     apiutil.ErrRollbackTx,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			relation:   auth.ContributorRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp:     &magistrala.IdentityRes{},
			idErr:      svcerr.ErrAuthentication,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "unsuccessfully with failed to authorize",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesRes: false,
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
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
				deletePoliciesCall = policy.On("DeletePolicies", context.Background(), &deletePolicies).Return(tc.deleteParentPoliciesRes, tc.deleteParentPoliciesErr)
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
			authCall2 := policy.On("AddPolicies", context.Background(), &policies).Return(tc.addPoliciesRes, tc.addPoliciesErr)
			err := svc.Assign(context.Background(), tc.token, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			authCall.Unset()
			authCall1.Unset()
			authCall2.Unset()
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
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
		deletePoliciesRes       bool
		deletePoliciesErr       error
		repoResp                mggroups.Page
		repoErr                 error
		addParentPoliciesRes    bool
		addParentPoliciesErr    error
		deleteParentPoliciesRes bool
		deleteParentPoliciesErr error
		repoParentGroupErr      error
		err                     error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: true,
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ChannelsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: true,
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			deletePoliciesRes:  true,
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: true,
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			repoErr:  repoerr.ErrViewEntity,
			err:      repoerr.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with groups kind due to empty page",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			relation:   auth.ContributorRelation,
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
			err: repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			deletePoliciesRes: false,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			deletePoliciesRes:  true,
			repoParentGroupErr: repoerr.ErrConflict,
			err:                repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent and add policy",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			deletePoliciesRes:    true,
			repoParentGroupErr:   repoerr.ErrConflict,
			addParentPoliciesRes: false,
			addParentPoliciesErr: svcerr.ErrAuthorization,
			err:                  repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
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
			relation:   auth.ContributorRelation,
			memberKind: auth.UsersKind,
			memberIDs:  allowedIDs,
			idResp:     &magistrala.IdentityRes{},
			idErr:      svcerr.ErrAuthentication,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "unsuccessfully with failed to authorize",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: false,
			},
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with failed to add policies",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   auth.ContributorRelation,
			memberKind: auth.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: false,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
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
				addPoliciesCall = policy.On("AddPolicies", context.Background(), &addPolicies).Return(tc.addParentPoliciesRes, tc.addParentPoliciesErr)
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
			authCall2 := policy.On("DeletePolicies", context.Background(), &policies).Return(tc.deletePoliciesRes, tc.deletePoliciesErr)
			err := svc.Unassign(context.Background(), tc.token, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			authCall.Unset()
			authCall1.Unset()
			authCall2.Unset()
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
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyService)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc              string
		token             string
		groupID           string
		idResp            *magistrala.IdentityRes
		idErr             error
		authzResp         *magistrala.AuthorizeRes
		authzErr          error
		deletePoliciesRes bool
		deletePoliciesErr error
		repoErr           error
		err               error
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
			deletePoliciesRes: true,
		},
		{
			desc:              "unsuccessfully with invalid token",
			token:             token,
			groupID:           testsutil.GenerateUUID(t),
			idResp:            &magistrala.IdentityRes{},
			deletePoliciesRes: false,
			idErr:             svcerr.ErrAuthentication,
			err:               svcerr.ErrAuthentication,
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
			deletePoliciesRes: false,
			authzErr:          svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove policy",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesRes: false,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
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
			deletePoliciesRes: true,
			repoErr:           repoerr.ErrNotFound,
			err:               repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  auth.DeletePermission,
				Object:      tc.groupID,
				ObjectType:  auth.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			authCall2 := policy.On("DeleteEntityPolicies", context.Background(), &magistrala.DeleteEntityPoliciesReq{
				EntityType: auth.GroupType,
				Id:         tc.groupID,
			}).Return(tc.deletePoliciesRes, tc.deletePoliciesErr)
			repoCall := repo.On("Delete", context.Background(), tc.groupID).Return(tc.repoErr)
			err := svc.DeleteGroup(context.Background(), tc.token, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			authCall.Unset()
			authCall1.Unset()
			authCall2.Unset()
			repoCall.Unset()
		})
	}
}
