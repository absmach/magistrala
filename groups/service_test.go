// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	chmocks "github.com/absmach/magistrala/channels/mocks"
	climocks "github.com/absmach/magistrala/clients/mocks"
	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/groups/mocks"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider = uuid.New()
	namegen    = namegenerator.NewGenerator()
	validGroup = groups.Group{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Status: groups.EnabledStatus,
	}
	parentGroupID = testsutil.GenerateUUID(&testing.T{})
	childGroupID  = testsutil.GenerateUUID(&testing.T{})
	childGroup    = groups.Group{
		ID:          childGroupID,
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Status: groups.EnabledStatus,
		Parent: parentGroupID,
	}
	children    = []*groups.Group{&childGroup}
	parentGroup = groups.Group{
		ID:          parentGroupID,
		Name:        namegen.Generate(),
		Description: namegen.Generate(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Status:   groups.EnabledStatus,
		Children: children,
	}
	validID          = testsutil.GenerateUUID(&testing.T{})
	errRollbackRoles = errors.New("failed to rollback roles")
	validSession     = authn.Session{UserID: validID, DomainID: validID, DomainUserID: validID}
)

var (
	repo     *mocks.Repository
	policies *policymocks.Service
	channels *chmocks.ChannelsServiceClient
	clients  *climocks.ClientsServiceClient
)

func newService(t *testing.T) groups.Service {
	repo = new(mocks.Repository)
	policies = new(policymocks.Service)
	channels = new(chmocks.ChannelsServiceClient)
	clients = new(climocks.ClientsServiceClient)
	availableActions := []roles.Action{}
	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		groups.BuiltInRoleAdmin: availableActions,
	}
	svc, err := groups.NewService(repo, policies, idProvider, channels, clients, idProvider, availableActions, builtInRoles)
	assert.Nil(t, err, fmt.Sprintf(" Unexpected error  while creating service %v", err))
	return svc
}

func TestCreateGroup(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		group             groups.Group
		saveResp          groups.Group
		saveErr           error
		deleteErr         error
		addPoliciesErr    error
		deletePoliciesErr error
		addRoleErr        error
		err               error
	}{
		{
			desc:  "create group successfully",
			group: validGroup,
			saveResp: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    validID,
			},
			err: nil,
		},
		{
			desc: "create group with invalid status",
			group: groups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      groups.Status(100),
			},
			err: svcerr.ErrInvalidStatus,
		},
		{
			desc: "create group successfully with parent",
			group: groups.Group{
				Name:        namegen.Generate(),
				Description: namegen.Generate(),
				Status:      groups.EnabledStatus,
				Parent:      testsutil.GenerateUUID(t),
			},
			saveResp: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    testsutil.GenerateUUID(t),
				Parent:    testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:     "create group with failed to save",
			group:    validGroup,
			saveResp: groups.Group{},
			saveErr:  errors.ErrMalformedEntity,
			err:      errors.Wrap(svcerr.ErrCreateEntity, errors.ErrMalformedEntity),
		},
		{
			desc:  " create group with failed to add policies",
			group: validGroup,
			saveResp: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    validID,
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            errors.Wrap(svcerr.ErrAddPolicies, errors.Wrap(svcerr.ErrCreateEntity, svcerr.ErrAuthorization)),
		},
		{
			desc:  " create group with failed to add policies and failed rollback",
			group: validGroup,
			saveResp: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    validID,
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			deleteErr:      svcerr.ErrRemoveEntity,
			err:            errors.Wrap(svcerr.ErrAddPolicies, errors.Wrap(apiutil.ErrRollbackTx, svcerr.ErrRemoveEntity)),
		},
		{
			desc:  "create group with failed to add roles",
			group: validGroup,
			saveResp: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    validID,
			},
			addRoleErr: svcerr.ErrCreateEntity,
			err:        errors.Wrap(svcerr.ErrAddPolicies, errors.Wrap(svcerr.ErrCreateEntity, svcerr.ErrCreateEntity)),
		},
		{
			desc:  "create groups with failed to add roles and failed to delete policies",
			group: validGroup,
			saveResp: groups.Group{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    validID,
			},
			addRoleErr:        svcerr.ErrCreateEntity,
			deletePoliciesErr: svcerr.ErrRemoveEntity,
			err:               errors.Wrap(svcerr.ErrAddPolicies, errors.Wrap(svcerr.ErrCreateEntity, errors.Wrap(errRollbackRoles, svcerr.ErrRemoveEntity))),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.saveResp, tc.saveErr)
			policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesErr)
			policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesErr)
			repoCall1 := repo.On("AddRoles", context.Background(), mock.Anything).Return([]roles.Role{}, tc.addRoleErr)
			repoCall2 := repo.On("Delete", context.Background(), mock.Anything).Return(tc.deleteErr)
			got, err := svc.CreateGroup(context.Background(), validSession, tc.group)
			assert.Equal(t, tc.err, err, fmt.Sprintf("expected error %v but got %v", tc.err, err))
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
			repoCall1.Unset()
			repoCall2.Unset()
		})
	}
}

func TestViewGroup(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc     string
		session  mgauthn.Session
		id       string
		repoResp groups.Group
		repoErr  error
		err      error
	}{
		{
			desc:     "view group successfully",
			id:       validGroup.ID,
			session:  validSession,
			repoResp: validGroup,
		},
		{
			desc:    "view group with failed to retrieve",
			id:      testsutil.GenerateUUID(t),
			session: validSession,
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByIDAndUser", context.Background(), tc.session.DomainID, tc.session.UserID, tc.id).Return(tc.repoResp, tc.repoErr)
			got, err := svc.ViewGroup(context.Background(), validSession, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "RetrieveByIDAndUser", context.Background(), tc.session.DomainID, tc.session.UserID, tc.id)
				assert.True(t, ok, fmt.Sprintf("RetrieveByIDAndUser was not called on %s", tc.desc))
			}
			repoCall.Unset()
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc     string
		group    groups.Group
		repoResp groups.Group
		repoErr  error
		err      error
	}{
		{
			desc: "update group successfully",
			group: groups.Group{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			repoResp: validGroup,
		},
		{
			desc: "update group with repo error",
			group: groups.Group{
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
			got, err := svc.UpdateGroup(context.Background(), validSession, tc.group)
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
	svc := newService(t)

	cases := []struct {
		desc         string
		id           string
		retrieveResp groups.Group
		retrieveErr  error
		changeResp   groups.Group
		changeErr    error
		err          error
	}{
		{
			desc: "enable group successfully",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: groups.Group{
				Status: groups.DisabledStatus,
			},
			changeResp: validGroup,
		},
		{
			desc: "enable group with enabled group",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: groups.Group{
				Status: groups.EnabledStatus,
			},
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:         "enable group with retrieve error",
			id:           testsutil.GenerateUUID(t),
			retrieveResp: groups.Group{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.EnableGroup(context.Background(), validSession, tc.id)
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
	svc := newService(t)

	cases := []struct {
		desc         string
		id           string
		retrieveResp groups.Group
		retrieveErr  error
		changeResp   groups.Group
		changeErr    error
		err          error
	}{
		{
			desc: "disable group successfully",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: groups.Group{
				Status: groups.EnabledStatus,
			},
			changeResp: validGroup,
		},
		{
			desc: "disable group with disabled group",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: groups.Group{
				Status: groups.DisabledStatus,
			},
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:         "disable group with retrieve error",
			id:           testsutil.GenerateUUID(t),
			retrieveResp: groups.Group{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.DisableGroup(context.Background(), validSession, tc.id)
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

func TestListGroups(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc                 string
		session              mgauthn.Session
		pageMeta             groups.PageMeta
		retrieveAllRes       groups.Page
		retrieveAllErr       error
		retrieveUserGroupRes groups.Page
		retrieveUserGroupErr error
		resp                 groups.Page
		err                  error
	}{
		{
			desc:    "list groups as super admin successfully",
			session: mgauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: true},
			pageMeta: groups.PageMeta{
				Limit:    10,
				Offset:   0,
				DomainID: validID,
			},
			retrieveAllRes: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			resp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: nil,
		},
		{
			desc:    "list groups as super admin with failed to retrieve",
			session: mgauthn.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: true},
			pageMeta: groups.PageMeta{
				Limit:    10,
				Offset:   0,
				DomainID: validID,
			},
			retrieveAllErr: repoerr.ErrNotFound,
			resp:           groups.Page{},
			err:            repoerr.ErrNotFound,
		},
		{
			desc:    "list groups as non admin successfully",
			session: validSession,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			retrieveUserGroupRes: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			resp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: nil,
		},
		{
			desc:    "list groups as non admin with failed to retrieve user groups",
			session: validSession,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			retrieveUserGroupErr: repoerr.ErrNotFound,
			resp:                 groups.Page{},
			err:                  svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveAll", context.Background(), tc.pageMeta).Return(tc.retrieveAllRes, tc.retrieveAllErr)
			repoCall1 := repo.On("RetrieveUserGroups", context.Background(), tc.session.DomainID, tc.session.UserID, tc.pageMeta).Return(tc.retrieveUserGroupRes, tc.retrieveUserGroupErr)
			got, err := svc.ListGroups(context.Background(), tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			assert.Equal(t, tc.resp, got)
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestListUserGroups(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc                 string
		session              mgauthn.Session
		userID               string
		pageMeta             groups.PageMeta
		retrieveUserGroupRes groups.Page
		retrieveUserGroupErr error
		resp                 groups.Page
		err                  error
	}{
		{
			desc:    "list user groups successfully",
			session: validSession,
			userID:  validID,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			retrieveUserGroupRes: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			resp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: nil,
		},
		{
			desc:    "list user groups with failed to retrieve",
			session: validSession,
			userID:  validID,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			retrieveUserGroupErr: repoerr.ErrNotFound,
			resp:                 groups.Page{},
			err:                  svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveUserGroups", context.Background(), tc.session.DomainID, tc.userID, tc.pageMeta).Return(tc.retrieveUserGroupRes, tc.retrieveUserGroupErr)
			got, err := svc.ListUserGroups(context.Background(), tc.session, tc.userID, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			assert.Equal(t, tc.resp, got)
			repoCall.Unset()
		})
	}
}

func TestRetrieveGroupHierarchy(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc                 string
		id                   string
		pageMeta             groups.HierarchyPageMeta
		retrieveHierarchyRes groups.HierarchyPage
		retrieveHierarchyErr error
		listAllObjectsRes    policysvc.PolicyPage
		listAllObjectsErr    error
		err                  error
	}{
		{
			desc: "retrieve group hierarchy successfully",
			id:   parentGroup.ID,
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			retrieveHierarchyRes: groups.HierarchyPage{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: -1,
					Tree:      false,
				},
				Groups: []groups.Group{parentGroup},
			},
			listAllObjectsRes: policysvc.PolicyPage{
				Policies: []string{parentGroupID, childGroupID},
			},
			err: nil,
		},
		{
			desc: "retrieve group hierarchy with failed to retrieve hierarchy",
			id:   parentGroup.ID,
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			retrieveHierarchyErr: repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc: "retrieve group hierarchy with failed to list all objects",
			id:   parentGroup.ID,
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			retrieveHierarchyRes: groups.HierarchyPage{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: -1,
					Tree:      false,
				},
				Groups: []groups.Group{parentGroup},
			},
			listAllObjectsErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc: "retrieve group hierarchy for group not allowed for user",
			id:   parentGroup.ID,
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			retrieveHierarchyRes: groups.HierarchyPage{
				HierarchyPageMeta: groups.HierarchyPageMeta{
					Level:     1,
					Direction: -1,
					Tree:      false,
				},
				Groups: []groups.Group{parentGroup},
			},
			listAllObjectsRes: policysvc.PolicyPage{
				Policies: []string{testsutil.GenerateUUID(t)},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveHierarchy", context.Background(), tc.id, tc.pageMeta).Return(tc.retrieveHierarchyRes, tc.retrieveHierarchyErr)
			policyCall := policies.On("ListAllObjects", context.Background(), policysvc.Policy{
				SubjectType: policysvc.UserType,
				Subject:     validID,
				Permission:  "read_permission",
				ObjectType:  policysvc.GroupType,
			}).Return(tc.listAllObjectsRes, tc.listAllObjectsErr)
			_, err := svc.RetrieveGroupHierarchy(context.Background(), validSession, tc.id, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if tc.err == nil {
				ok := repo.AssertCalled(t, "RetrieveHierarchy", context.Background(), tc.id, tc.pageMeta)
				assert.True(t, ok, fmt.Sprintf("RetrieveHierarchy was not called on %s", tc.desc))
			}
			repoCall.Unset()
			policyCall.Unset()
		})
	}
}

func TestAddParentGroup(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		id                string
		parentID          string
		retrieveResp      groups.Group
		retrieveErr       error
		addPoliciesErr    error
		deletePoliciesErr error
		assignParentErr   error
		err               error
	}{
		{
			desc:         "add parent group successfully",
			id:           validGroup.ID,
			parentID:     parentGroupID,
			retrieveResp: validGroup,
			err:          nil,
		},
		{
			desc:        "add parent group with failed to retrieve",
			id:          validGroup.ID,
			parentID:    parentGroupID,
			retrieveErr: repoerr.ErrNotFound,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:         "add parent group to group with parent",
			id:           childGroupID,
			parentID:     parentGroupID,
			retrieveResp: childGroup,
			err:          svcerr.ErrConflict,
		},
		{
			desc:           "add parent group with failed to add policies",
			id:             validGroup.ID,
			parentID:       parentGroupID,
			retrieveResp:   validGroup,
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAddPolicies,
		},
		{
			desc:            "add parent group with repo error in assign parent group",
			id:              validGroup.ID,
			parentID:        parentGroupID,
			retrieveResp:    validGroup,
			assignParentErr: repoerr.ErrNotFound,
			err:             repoerr.ErrNotFound,
		},
		{
			desc:              "add parent group with repo error in assign parent group and failed to delete policies",
			id:                validGroup.ID,
			parentID:          parentGroupID,
			retrieveResp:      validGroup,
			assignParentErr:   repoerr.ErrNotFound,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pol := policysvc.Policy{
				Domain:      validID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.parentID,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.GroupType,
				Object:      tc.id,
			}
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			policyCall := policies.On("AddPolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.addPoliciesErr)
			policyCall1 := policies.On("DeletePolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.deletePoliciesErr)
			repoCall1 := repo.On("AssignParentGroup", context.Background(), tc.parentID, []string{tc.id}).Return(tc.assignParentErr)
			err := svc.AddParentGroup(context.Background(), validSession, tc.id, tc.parentID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			repoCall.Unset()
			policyCall.Unset()
			policyCall1.Unset()
			repoCall1.Unset()
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		id                string
		retrieveResp      groups.Group
		retrieveErr       error
		deletePoliciesErr error
		addPoliciesErr    error
		unassignParentErr error
		err               error
	}{
		{
			desc:         "remove parent group successfully",
			id:           childGroupID,
			retrieveResp: childGroup,
			err:          nil,
		},
		{
			desc:        "remove parent group with failed to retrieve",
			id:          childGroupID,
			retrieveErr: repoerr.ErrNotFound,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:         "remove parent group with no parent",
			id:           validGroup.ID,
			retrieveResp: validGroup,
			err:          nil,
		},
		{
			desc:              "remove parent group with failed to delete policies",
			id:                childGroupID,
			retrieveResp:      childGroup,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrDeletePolicies,
		},
		{
			desc:              "remove parent group with repo error in unassign parent group",
			id:                childGroupID,
			retrieveResp:      childGroup,
			unassignParentErr: repoerr.ErrNotFound,
			err:               repoerr.ErrNotFound,
		},
		{
			desc:              "remove parent group with repo error in unassign parent group and failed to add policies",
			id:                childGroupID,
			retrieveResp:      childGroup,
			unassignParentErr: repoerr.ErrNotFound,
			addPoliciesErr:    svcerr.ErrAuthorization,
			err:               apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pol := policysvc.Policy{
				Domain:      validID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.retrieveResp.Parent,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.GroupType,
				Object:      tc.id,
			}
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			policyCall := policies.On("DeletePolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.deletePoliciesErr)
			policyCall1 := policies.On("AddPolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.addPoliciesErr)
			repoCall1 := repo.On("UnassignParentGroup", context.Background(), tc.retrieveResp.Parent, []string{tc.id}).Return(tc.unassignParentErr)
			err := svc.RemoveParentGroup(context.Background(), validSession, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			ok := repo.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			repoCall.Unset()
			policyCall.Unset()
			policyCall1.Unset()
			repoCall1.Unset()
		})
	}
}

func TestAddChildrenGroups(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		parentID          string
		childrenIDs       []string
		retrieveResp      groups.Page
		retrieveErr       error
		addPoliciesErr    error
		deletePoliciesErr error
		assignParentErr   error
		err               error
	}{
		{
			desc:        "add children groups successfully",
			parentID:    parentGroupID,
			childrenIDs: []string{validGroup.ID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: nil,
		},
		{
			desc:        "add children groups with failed to retrieve",
			parentID:    parentGroupID,
			childrenIDs: []string{validGroup.ID},
			retrieveErr: repoerr.ErrNotFound,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:         "add non existent child group",
			parentID:     parentGroupID,
			childrenIDs:  []string{testsutil.GenerateUUID(&testing.T{})},
			retrieveResp: groups.Page{},
			err:          groups.ErrGroupIDs,
		},
		{
			desc:        "add child group with parent",
			parentID:    parentGroupID,
			childrenIDs: []string{childGroupID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: svcerr.ErrConflict,
		},
		{
			desc:        "add children groups with failed to add policies",
			parentID:    parentGroupID,
			childrenIDs: []string{validGroup.ID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAddPolicies,
		},
		{
			desc:        "add children groups with repo error in assign children groups",
			parentID:    parentGroupID,
			childrenIDs: []string{validGroup.ID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			assignParentErr: repoerr.ErrNotFound,
			err:             repoerr.ErrNotFound,
		},
		{
			desc:        "add children groups with repo error in assign children groups and failed to delete policies",
			parentID:    parentGroupID,
			childrenIDs: []string{validGroup.ID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{validGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			assignParentErr:   repoerr.ErrNotFound,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pol := policysvc.Policy{
				Domain:      validID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.parentID,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.GroupType,
				Object:      validGroup.ID,
			}
			repoCall := repo.On("RetrieveByIDs", context.Background(), groups.PageMeta{Limit: 1<<63 - 1}, tc.childrenIDs).Return(tc.retrieveResp, tc.retrieveErr)
			policyCall := policies.On("AddPolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.addPoliciesErr)
			policyCall1 := policies.On("DeletePolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.deletePoliciesErr)
			repoCall1 := repo.On("AssignParentGroup", context.Background(), tc.parentID, tc.childrenIDs).Return(tc.assignParentErr)
			err := svc.AddChildrenGroups(context.Background(), validSession, tc.parentID, tc.childrenIDs)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repoCall.Unset()
			policyCall.Unset()
			policyCall1.Unset()
			repoCall1.Unset()
		})
	}
}

func TestRemoveChildrenGroups(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		parentID          string
		childrenIDs       []string
		retrieveResp      groups.Page
		retrieveErr       error
		deletePoliciesErr error
		addPoliciesErr    error
		unassignParentErr error
		err               error
	}{
		{
			desc:        "remove children groups successfully",
			parentID:    parentGroupID,
			childrenIDs: []string{childGroupID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: nil,
		},
		{
			desc:        "remove children groups with failed to retrieve",
			parentID:    parentGroupID,
			childrenIDs: []string{childGroupID},
			retrieveErr: repoerr.ErrNotFound,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:         "remove non existent child group",
			parentID:     parentGroupID,
			childrenIDs:  []string{testsutil.GenerateUUID(&testing.T{})},
			retrieveResp: groups.Page{},
			err:          groups.ErrGroupIDs,
		},
		{
			desc:        "remove children groups from different parent",
			parentID:    validGroup.ID,
			childrenIDs: []string{childGroupID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: svcerr.ErrConflict,
		},
		{
			desc:        "remove children groups with failed to delete policies",
			parentID:    parentGroupID,
			childrenIDs: []string{childGroupID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrDeletePolicies,
		},
		{
			desc:        "remove children groups with repo error in unassign children groups",
			parentID:    parentGroupID,
			childrenIDs: []string{childGroupID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			unassignParentErr: repoerr.ErrNotFound,
			err:               repoerr.ErrNotFound,
		},
		{
			desc:        "remove children groups with repo error in unassign children groups and failed to add policies",
			parentID:    parentGroupID,
			childrenIDs: []string{childGroupID},
			retrieveResp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			unassignParentErr: repoerr.ErrNotFound,
			addPoliciesErr:    svcerr.ErrAuthorization,
			err:               apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pol := policysvc.Policy{
				Domain:      validID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.parentID,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.GroupType,
				Object:      childGroupID,
			}
			repoCall := repo.On("RetrieveByIDs", context.Background(), groups.PageMeta{Limit: 1<<63 - 1}, tc.childrenIDs).Return(tc.retrieveResp, tc.retrieveErr)
			policyCall := policies.On("DeletePolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.deletePoliciesErr)
			policyCall1 := policies.On("AddPolicies", context.Background(), []policysvc.Policy{pol}).Return(tc.addPoliciesErr)
			repoCall1 := repo.On("UnassignParentGroup", context.Background(), tc.parentID, tc.childrenIDs).Return(tc.unassignParentErr)
			err := svc.RemoveChildrenGroups(context.Background(), validSession, tc.parentID, tc.childrenIDs)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repoCall.Unset()
			policyCall.Unset()
			policyCall1.Unset()
			repoCall1.Unset()
		})
	}
}

func TestRemoveAllChildrenGroups(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc                   string
		parentID               string
		deletePolicyErr        error
		unassignAllChildrenErr error
		err                    error
	}{
		{
			desc:     "remove all children groups successfully",
			parentID: parentGroupID,
			err:      nil,
		},
		{
			desc:            "remove all children groups with failed to delete policy",
			parentID:        parentGroupID,
			deletePolicyErr: svcerr.ErrAuthorization,
			err:             svcerr.ErrDeletePolicies,
		},
		{
			desc:                   "remove all children groups with failed to unassign all children",
			parentID:               parentGroupID,
			deletePolicyErr:        nil,
			unassignAllChildrenErr: repoerr.ErrNotFound,
			err:                    repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			policyCall := policies.On("DeletePolicyFilter", context.Background(), policysvc.Policy{
				Domain:      validID,
				SubjectType: policysvc.GroupType,
				Subject:     tc.parentID,
				Relation:    policysvc.ParentGroupRelation,
				ObjectType:  policysvc.GroupType,
			}).Return(tc.deletePolicyErr)
			repoCall := repo.On("UnassignAllChildrenGroups", context.Background(), tc.parentID).Return(tc.unassignAllChildrenErr)
			err := svc.RemoveAllChildrenGroups(context.Background(), validSession, tc.parentID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			policyCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestListAllChildrenGroups(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc        string
		session     mgauthn.Session
		pageMeta    groups.PageMeta
		parentID    string
		startLevel  int64
		endLevel    int64
		retrieveRes groups.Page
		retrieveErr error
		resp        groups.Page
		err         error
	}{
		{
			desc:     "list all children groups successfully",
			session:  validSession,
			parentID: parentGroupID,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			startLevel: 0,
			endLevel:   -1,
			retrieveRes: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			resp: groups.Page{
				Groups: []groups.Group{childGroup},
				PageMeta: groups.PageMeta{
					Total: 1,
				},
			},
			err: nil,
		},
		{
			desc:     "list all children groups with failed to retrieve",
			session:  validSession,
			parentID: parentGroupID,
			pageMeta: groups.PageMeta{
				Limit:  10,
				Offset: 0,
			},
			startLevel:  0,
			endLevel:    -1,
			retrieveErr: repoerr.ErrNotFound,
			resp:        groups.Page{},
			err:         svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveChildrenGroups", context.Background(), tc.session.DomainID, tc.session.UserID, tc.parentID, tc.startLevel, tc.endLevel, tc.pageMeta).Return(tc.retrieveRes, tc.retrieveErr)
			page, err := svc.ListChildrenGroups(context.Background(), tc.session, tc.parentID, tc.startLevel, tc.endLevel, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			assert.Equal(t, tc.resp, page)
			repoCall.Unset()
		})
	}
}

func TestDeleteGroup(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		id                string
		changeStatusRes   groups.Group
		changeStatusErr   error
		deletePoliciesErr error
		deleteErr         error
		unsetFromChannels error
		unsetFromClients  error
		err               error
	}{
		{
			desc: "delete group successfully",
			id:   validGroup.ID,
			err:  nil,
		},
		{
			desc:            "delete group with parent successfully",
			id:              childGroupID,
			changeStatusRes: childGroup,
			err:             nil,
		},
		{
			desc:              "delete group with failed to remove parent group from channels",
			id:                validGroup.ID,
			unsetFromChannels: svcerr.ErrRemoveEntity,
			err:               svcerr.ErrRemoveEntity,
		},
		{
			desc:              "delete group with failed to remove parent group from clients",
			id:                validGroup.ID,
			unsetFromChannels: nil,
			unsetFromClients:  svcerr.ErrRemoveEntity,
			err:               svcerr.ErrRemoveEntity,
		},
		{
			desc:            "delete group with failed to change status",
			id:              validGroup.ID,
			changeStatusErr: repoerr.ErrNotFound,
			err:             repoerr.ErrNotFound,
		},
		{
			desc:            "delete group with failed to delete",
			id:              validGroup.ID,
			changeStatusRes: validGroup,
			deleteErr:       repoerr.ErrNotFound,
			err:             repoerr.ErrNotFound,
		},
		{
			desc:              "delete group with failed to delete policies",
			id:                validGroup.ID,
			changeStatusRes:   validGroup,
			deleteErr:         nil,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrDeletePolicies,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ChangeStatus", context.Background(), groups.Group{ID: tc.id, Status: groups.DeletedStatus}).Return(tc.changeStatusRes, tc.changeStatusErr)
			repoCall1 := repo.On("Delete", context.Background(), tc.id).Return(tc.deleteErr)
			svcCall := channels.On("UnsetParentGroupFromChannels", context.Background(), &grpcChannelsV1.UnsetParentGroupFromChannelsReq{ParentGroupId: tc.id}).Return(&grpcChannelsV1.UnsetParentGroupFromChannelsRes{}, tc.unsetFromChannels)
			svcCall1 := clients.On("UnsetParentGroupFromClient", context.Background(), &grpcClientsV1.UnsetParentGroupFromClientReq{ParentGroupId: tc.id}).Return(&grpcClientsV1.UnsetParentGroupFromClientRes{}, tc.unsetFromClients)
			repoCall2 := repo.On("RetrieveEntitiesRolesActionsMembers", context.Background(), []string{tc.id}).Return([]roles.EntityActionRole{}, []roles.EntityMemberRole{}, nil)
			policyCall := policies.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePoliciesErr)
			policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(nil)
			err := svc.DeleteGroup(context.Background(), validSession, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			policyCall.Unset()
			repoCall.Unset()
			repoCall1.Unset()
			svcCall.Unset()
			svcCall1.Unset()
			repoCall2.Unset()
			policyCall1.Unset()
		})
	}
}
