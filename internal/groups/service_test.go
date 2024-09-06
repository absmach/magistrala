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
	policysvc "github.com/absmach/magistrala/pkg/policy"
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
	policy := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

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
		addPolErr    error
		deletePolErr error
		err          error
	}{
		{
			desc:  "successfully",
			token: token,
			kind:  policysvc.NewGroupKind,
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
		},
		{
			desc:   "with invalid token",
			token:  token,
			kind:   policysvc.NewGroupKind,
			group:  validGroup,
			idResp: &magistrala.IdentityRes{},
			idErr:  svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "with empty id or domain id but with no grpc error",
			token:  token,
			kind:   policysvc.NewGroupKind,
			group:  validGroup,
			idResp: &magistrala.IdentityRes{},
			idErr:  nil,
			err:    svcerr.ErrDomainAuthorization,
		},
		{
			desc:  "with failed to authorize domain membership",
			token: token,
			kind:  policysvc.NewGroupKind,
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
			kind:  policysvc.NewGroupKind,
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
			kind:  policysvc.NewGroupKind,
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
			kind:  policysvc.NewGroupKind,
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
		},
		{
			desc:  "unsuccessfully with parent due to authorization error",
			token: token,
			kind:  policysvc.NewGroupKind,
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
			err: svcerr.ErrAuthorization,
		},
		{
			desc:  "with repo error",
			token: token,
			kind:  policysvc.NewGroupKind,
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
			kind:  policysvc.NewGroupKind,
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
			addPolErr: svcerr.ErrAuthorization,
			err:       svcerr.ErrAuthorization,
		},
		{
			desc:  "with failed to delete policies response",
			token: token,
			kind:  policysvc.NewGroupKind,
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
			deletePolErr: svcerr.ErrAuthorization,
			err:          errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  policysvc.CreatePermission,
				Object:      tc.idResp.GetDomainId(),
				ObjectType:  policysvc.DomainType,
			}).Return(tc.authzResp, tc.authzErr)
			authCall2 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.TokenKind,
				Subject:     tc.token,
				Permission:  policysvc.EditPermission,
				Object:      tc.group.Parent,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.authzTknResp, tc.authzTknErr)
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			policyCall := policy.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPolErr)
			policyCall1 := policy.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePolErr)
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
			policyCall.Unset()
			policyCall1.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyClient)
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
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.TokenKind,
				Subject:     tc.token,
				Permission:  policysvc.ViewPermission,
				Object:      tc.id,
				ObjectType:  policysvc.GroupType,
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
	policy := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc     string
		token    string
		id       string
		idResp   *magistrala.IdentityRes
		idErr    error
		listResp policysvc.Permissions
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
				policysvc.ViewPermission,
				policysvc.EditPermission,
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
			policyCall := policy.On("ListPermissions", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.UserType,
				Subject:     tc.idResp.GetId(),
				Object:      tc.id,
				ObjectType:  policysvc.GroupType,
			}, []string{}).Return(tc.listResp, tc.listErr)
			got, err := svc.ViewGroupPerms(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.ElementsMatch(t, tc.listResp, got)
			}
			authCall.Unset()
			policyCall.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyClient)
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
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.TokenKind,
				Subject:     tc.token,
				Permission:  policysvc.EditPermission,
				Object:      tc.group.ID,
				ObjectType:  policysvc.GroupType,
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
	policy := new(policymocks.PolicyClient)
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
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.TokenKind,
				Subject:     tc.token,
				Permission:  policysvc.EditPermission,
				Object:      tc.id,
				ObjectType:  policysvc.GroupType,
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
	policy := new(policymocks.PolicyClient)
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
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.TokenKind,
				Subject:     tc.token,
				Permission:  policysvc.EditPermission,
				Object:      tc.id,
				ObjectType:  policysvc.GroupType,
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
	policy := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc            string
		token           string
		groupID         string
		permission      string
		memberKind      string
		authzResp       *magistrala.AuthorizeRes
		authzErr        error
		listSubjectResp policysvc.PolicyPage
		listSubjectErr  error
		listObjectResp  policysvc.PolicyPage
		listObjectErr   error
		err             error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
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
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			permission: policysvc.ViewPermission,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
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
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			permission: policysvc.ViewPermission,
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
			memberKind: policysvc.ThingsKind,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: policysvc.PolicyPage{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "failed to list subjects with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			permission: policysvc.ViewPermission,
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: policysvc.PolicyPage{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.TokenKind,
				Subject:     tc.token,
				Permission:  policysvc.ViewPermission,
				Object:      tc.groupID,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			policyCall := policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.GroupType,
				Subject:     tc.groupID,
				Relation:    policysvc.GroupRelation,
				ObjectType:  policysvc.ThingType,
			}).Return(tc.listObjectResp, tc.listObjectErr)
			policyCall1 := policy.On("ListAllSubjects", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.UserType,
				Permission:  tc.permission,
				Object:      tc.groupID,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.listSubjectResp, tc.listSubjectErr)
			got, err := svc.ListMembers(context.Background(), tc.token, tc.groupID, tc.permission, tc.memberKind)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			authCall.Unset()
			policyCall.Unset()
			policyCall1.Unset()
		})
	}
}

func TestListGroups(t *testing.T) {
	repo := new(mocks.Repository)
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyClient)
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
		listSubjectResp      policysvc.PolicyPage
		listSubjectErr       error
		listObjectResp       policysvc.PolicyPage
		listObjectErr        error
		listObjectFilterResp policysvc.PolicyPage
		listObjectFilterErr  error
		authSuperAdminResp   *magistrala.AuthorizeRes
		authSuperAdminErr    error
		repoResp             mggroups.Page
		repoErr              error
		listPermResp         policysvc.Permissions
		listPermErr          error
		err                  error
	}{
		{
			desc:       "successfully with things kind",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
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
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
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
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
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
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
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
			token:      token,
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			desc:       "unsuccessfully with users kind admin",
			token:      token,
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: policysvc.PolicyPage{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: policysvc.PolicyPage{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.GroupsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp: policysvc.PolicyPage{},
			listSubjectErr:  svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with channels kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ChannelsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to authorize",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp: policysvc.PolicyPage{},
			listObjectErr:  svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with users kind due to failed to list filtered objects",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listObjectResp:       policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{},
			listObjectFilterErr:  svcerr.ErrAuthorization,
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc:       "successfully with users kind admin",
			token:      token,
			memberKind: policysvc.UsersKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: "invalid",
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			listSubjectResp:      policysvc.PolicyPage{Policies: allowedIDs},
			listObjectFilterResp: policysvc.PolicyPage{Policies: allowedIDs},
			repoResp:             mggroups.Page{},
			repoErr:              repoerr.ErrViewEntity,
			err:                  repoerr.ErrViewEntity,
		},
		{
			desc:       "unsuccessfully with things kind due to failed to list permissions",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
				ListPerms:  true,
			},
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
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
		{
			desc:       "unsuccessfully with invalid token",
			token:      token,
			memberID:   testsutil.GenerateUUID(t),
			memberKind: policysvc.ThingsKind,
			page: mggroups.Page{
				Permission: policysvc.ViewPermission,
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
			policyCall := &mock.Call{}
			policyCall1 := &mock.Call{}
			adminCheck := &mock.Call{}
			switch tc.memberKind {
			case policysvc.ThingsKind:
				authCall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: policysvc.UserType,
					SubjectKind: policysvc.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  policysvc.ViewPermission,
					Object:      tc.memberID,
					ObjectType:  policysvc.ThingType,
				}).Return(tc.authzResp, tc.authzErr)
				policyCall = policy.On("ListAllSubjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.GroupType,
					Permission:  policysvc.GroupRelation,
					ObjectType:  policysvc.ThingType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				policyCall1 = policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case policysvc.GroupsKind:
				authCall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: policysvc.UserType,
					SubjectKind: policysvc.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					Object:      tc.memberID,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.authzResp, tc.authzErr)
				policyCall = policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.GroupType,
					Subject:     tc.memberID,
					Permission:  policysvc.ParentGroupRelation,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				policyCall1 = policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case policysvc.ChannelsKind:
				authCall1 = authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: policysvc.UserType,
					SubjectKind: policysvc.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  policysvc.ViewPermission,
					Object:      tc.memberID,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.authzResp, tc.authzErr)
				policyCall = policy.On("ListAllSubjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.GroupType,
					Permission:  policysvc.ParentGroupRelation,
					ObjectType:  policysvc.GroupType,
					Object:      tc.memberID,
				}).Return(tc.listSubjectResp, tc.listSubjectErr)
				policyCall1 = policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			case policysvc.UsersKind:
				adminCheckReq := &magistrala.AuthorizeReq{
					SubjectType: policysvc.UserType,
					Subject:     tc.idResp.GetUserId(),
					Permission:  policysvc.AdminPermission,
					Object:      policysvc.MagistralaObject,
					ObjectType:  policysvc.PlatformType,
				}
				adminCheck = authsvc.On("Authorize", context.Background(), adminCheckReq).Return(tc.authzResp, tc.authzErr)
				authReq := &magistrala.AuthorizeReq{
					Domain:      tc.idResp.GetDomainId(),
					SubjectType: policysvc.UserType,
					SubjectKind: policysvc.UsersKind,
					Subject:     tc.idResp.GetId(),
					Permission:  policysvc.AdminPermission,
					Object:      tc.idResp.GetDomainId(),
					ObjectType:  policysvc.DomainType,
				}
				if tc.memberID == "" {
					authReq.Domain = ""
					authReq.Permission = policysvc.MembershipPermission
				}
				authCall1 = authsvc.On("Authorize", context.Background(), authReq).Return(tc.authzResp, tc.authzErr)
				policyCall = policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), tc.memberID),
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectResp, tc.listObjectErr)
				policyCall1 = policy.On("ListAllObjects", context.Background(), policysvc.PolicyReq{
					SubjectType: policysvc.UserType,
					Subject:     tc.idResp.GetId(),
					Permission:  tc.page.Permission,
					ObjectType:  policysvc.GroupType,
				}).Return(tc.listObjectFilterResp, tc.listObjectFilterErr)
			}
			repoCall := repo.On("RetrieveByIDs", context.Background(), mock.Anything, mock.Anything).Return(tc.repoResp, tc.repoErr)
			authCall4 := policy.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything).Return(tc.listPermResp, tc.listPermErr)
			got, err := svc.ListGroups(context.Background(), tc.token, tc.memberKind, tc.memberID, tc.page)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.NotEmpty(t, got)
			}
			authCall.Unset()
			repoCall.Unset()
			switch tc.memberKind {
			case policysvc.ThingsKind, policysvc.GroupsKind, policysvc.ChannelsKind, policysvc.UsersKind:
				authCall1.Unset()
				policyCall.Unset()
				policyCall1.Unset()
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
	policy := new(policymocks.PolicyClient)
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
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ChannelsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.UsersKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			repoParentGroupErr: repoerr.ErrConflict,
			err:                repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to assign parent and delete policy",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			deleteParentPoliciesErr: svcerr.ErrAuthorization,
			repoParentGroupErr:      repoerr.ErrConflict,
			err:                     apiutil.ErrRollbackTx,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.UsersKind,
			memberIDs:  allowedIDs,
			idResp:     &magistrala.IdentityRes{},
			idErr:      svcerr.ErrAuthentication,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "unsuccessfully with failed to authorize",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  policysvc.EditPermission,
				Object:      tc.groupID,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			retrieveByIDsCall := &mock.Call{}
			deletePoliciesCall := &mock.Call{}
			assignParentCall := &mock.Call{}
			policies := []policysvc.PolicyReq{}
			switch tc.memberKind {
			case policysvc.ThingsKind:
				for _, memberID := range tc.memberIDs {
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
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
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: policysvc.GroupType,
						Subject:     tc.groupID,
						Relation:    policysvc.ParentGroupRelation,
						ObjectType:  policysvc.GroupType,
						Object:      group.ID,
					})
				}
				deletePoliciesCall = policy.On("DeletePolicies", context.Background(), policies).Return(tc.deleteParentPoliciesErr)
				assignParentCall = repo.On("AssignParentGroup", context.Background(), tc.groupID, tc.memberIDs).Return(tc.repoParentGroupErr)
			case policysvc.ChannelsKind:
				for _, memberID := range tc.memberIDs {
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: policysvc.GroupType,
						Subject:     memberID,
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			case policysvc.UsersKind:
				for _, memberID := range tc.memberIDs {
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: policysvc.UserType,
						Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), memberID),
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			}
			policyCall := policy.On("AddPolicies", context.Background(), policies).Return(tc.addPoliciesErr)
			err := svc.Assign(context.Background(), tc.token, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			authCall.Unset()
			authCall1.Unset()
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
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyClient)
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
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
		},
		{
			desc:       "successfully with channels kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ChannelsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
		},
		{
			desc:       "successfully with groups kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			repoParentGroupErr: nil,
		},
		{
			desc:       "successfully with users kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.UsersKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
		},
		{
			desc:       "unsuccessfully with groups kind due to repo err",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			repoParentGroupErr: repoerr.ErrConflict,
			err:                repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with groups kind due to failed to unassign parent and add policy",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.GroupsKind,
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
			repoParentGroupErr:   repoerr.ErrConflict,
			addParentPoliciesErr: svcerr.ErrAuthorization,
			err:                  repoerr.ErrConflict,
		},
		{
			desc:       "unsuccessfully with invalid kind",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.UsersKind,
			memberIDs:  allowedIDs,
			idResp:     &magistrala.IdentityRes{},
			idErr:      svcerr.ErrAuthentication,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "unsuccessfully with failed to authorize",
			token:      token,
			groupID:    testsutil.GenerateUUID(t),
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
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
			relation:   policysvc.ContributorRelation,
			memberKind: policysvc.ThingsKind,
			memberIDs:  allowedIDs,
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  policysvc.EditPermission,
				Object:      tc.groupID,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			retrieveByIDsCall := &mock.Call{}
			addPoliciesCall := &mock.Call{}
			assignParentCall := &mock.Call{}
			policies := []policysvc.PolicyReq{}
			switch tc.memberKind {
			case policysvc.ThingsKind:
				for _, memberID := range tc.memberIDs {
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
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
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: policysvc.GroupType,
						Subject:     tc.groupID,
						Relation:    policysvc.ParentGroupRelation,
						ObjectType:  policysvc.GroupType,
						Object:      group.ID,
					})
				}
				addPoliciesCall = policy.On("AddPolicies", context.Background(), policies).Return(tc.addParentPoliciesErr)
				assignParentCall = repo.On("UnassignParentGroup", context.Background(), tc.groupID, tc.memberIDs).Return(tc.repoParentGroupErr)
			case policysvc.ChannelsKind:
				for _, memberID := range tc.memberIDs {
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: policysvc.GroupType,
						Subject:     memberID,
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			case policysvc.UsersKind:
				for _, memberID := range tc.memberIDs {
					policies = append(policies, policysvc.PolicyReq{
						Domain:      tc.idResp.GetDomainId(),
						SubjectType: policysvc.UserType,
						Subject:     auth.EncodeDomainUserID(tc.idResp.GetDomainId(), memberID),
						Relation:    tc.relation,
						ObjectType:  policysvc.GroupType,
						Object:      tc.groupID,
					})
				}
			}
			policyCall := policy.On("DeletePolicies", context.Background(), policies).Return(tc.deletePoliciesErr)
			err := svc.Unassign(context.Background(), tc.token, tc.groupID, tc.relation, tc.memberKind, tc.memberIDs...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			authCall.Unset()
			authCall1.Unset()
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
	authsvc := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyClient)
	svc := groups.NewService(repo, idProvider, authsvc, policy)

	cases := []struct {
		desc                     string
		token                    string
		groupID                  string
		idResp                   *magistrala.IdentityRes
		idErr                    error
		authzResp                *magistrala.AuthorizeRes
		authzErr                 error
		deleteSubjectPoliciesErr error
		deleteObjectPoliciesErr  error
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
		},
		{
			desc:    "unsuccessfully with invalid token",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp:  &magistrala.IdentityRes{},
			idErr:   svcerr.ErrAuthentication,
			err:     svcerr.ErrAuthentication,
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
			authzErr: svcerr.ErrAuthorization,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove subject policies",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteSubjectPoliciesErr: svcerr.ErrAuthorization,
			err:                      svcerr.ErrAuthorization,
		},
		{
			desc:    "unsuccessfully with failed to remove object policies",
			token:   token,
			groupID: testsutil.GenerateUUID(t),
			idResp: &magistrala.IdentityRes{
				Id:       testsutil.GenerateUUID(t),
				DomainId: testsutil.GenerateUUID(t),
			},
			authzResp: &magistrala.AuthorizeRes{
				Authorized: true,
			},
			deleteObjectPoliciesErr: svcerr.ErrAuthorization,
			err:                     svcerr.ErrAuthorization,
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
			repoErr: repoerr.ErrNotFound,
			err:     repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authsvc.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(tc.idResp, tc.idErr)
			authCall1 := authsvc.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
				Domain:      tc.idResp.GetDomainId(),
				SubjectType: policysvc.UserType,
				SubjectKind: policysvc.UsersKind,
				Subject:     tc.idResp.GetId(),
				Permission:  policysvc.DeletePermission,
				Object:      tc.groupID,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.authzResp, tc.authzErr)
			policyCall := policy.On("DeletePolicyFilter", context.Background(), policysvc.PolicyReq{
				SubjectType: policysvc.GroupType,
				Subject:     tc.groupID,
			}).Return(tc.deleteSubjectPoliciesErr)
			policyCall2 := policy.On("DeletePolicyFilter", context.Background(), policysvc.PolicyReq{
				ObjectType: policysvc.GroupType,
				Object:     tc.groupID,
			}).Return(tc.deleteObjectPoliciesErr)
			repoCall := repo.On("Delete", context.Background(), tc.groupID).Return(tc.repoErr)
			err := svc.DeleteGroup(context.Background(), tc.token, tc.groupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			authCall.Unset()
			authCall1.Unset()
			policyCall.Unset()
			policyCall2.Unset()
			repoCall.Unset()
		})
	}
}
