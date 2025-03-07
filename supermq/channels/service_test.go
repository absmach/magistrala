// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package channels_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/channels/mocks"
	"github.com/absmach/supermq/clients"
	smqclients "github.com/absmach/supermq/clients"
	clmocks "github.com/absmach/supermq/clients/mocks"
	gpmocks "github.com/absmach/supermq/groups/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/authn"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	policysvc "github.com/absmach/supermq/pkg/policies"
	policymocks "github.com/absmach/supermq/pkg/policies/mocks"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider   = uuid.New()
	namegen      = namegenerator.NewGenerator()
	validChannel = channels.Channel{
		ID:   testsutil.GenerateUUID(&testing.T{}),
		Name: namegen.Generate(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Tags:   []string{"tag1", "tag2"},
		Domain: testsutil.GenerateUUID(&testing.T{}),
		Status: clients.EnabledStatus,
	}
	parentGroupID    = testsutil.GenerateUUID(&testing.T{})
	validID          = testsutil.GenerateUUID(&testing.T{})
	validSession     = authn.Session{UserID: validID, DomainID: validID, DomainUserID: validID}
	errRollbackRoles = errors.New("failed to rollback roles")
)

var (
	repo       *mocks.Repository
	policies   *policymocks.Service
	clientsSvc *clmocks.ClientsServiceClient
	groupsSvc  *gpmocks.GroupsServiceClient
)

func newService(t *testing.T) channels.Service {
	repo = new(mocks.Repository)
	policies = new(policymocks.Service)
	clientsSvc = new(clmocks.ClientsServiceClient)
	groupsSvc = new(gpmocks.GroupsServiceClient)
	availableActions := []roles.Action{}
	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		clients.BuiltInRoleAdmin: availableActions,
	}
	svc, err := channels.New(repo, policies, idProvider, clientsSvc, groupsSvc, idProvider, availableActions, builtInRoles)
	assert.Nil(t, err, fmt.Sprintf(" Unexpected error  while creating service %v", err))
	return svc
}

func TestCreateChannel(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc              string
		channel           channels.Channel
		saveResp          []channels.Channel
		saveErr           error
		deleteErr         error
		addPoliciesErr    error
		deletePoliciesErr error
		addRoleErr        error
		err               error
	}{
		{
			desc:    "create channel successfully",
			channel: validChannel,
			saveResp: []channels.Channel{{
				ID:        testsutil.GenerateUUID(t),
				CreatedAt: time.Now(),
				Domain:    validID,
			}},
			err: nil,
		},
		{
			desc: "create channel with invalid status",
			channel: channels.Channel{
				Name:   namegen.Generate(),
				Status: clients.Status(100),
			},
			err: svcerr.ErrInvalidStatus,
		},
		{
			desc: "create channel successfully with parent",
			channel: channels.Channel{
				Name:        namegen.Generate(),
				Status:      clients.EnabledStatus,
				ParentGroup: testsutil.GenerateUUID(t),
			},
			saveResp: []channels.Channel{
				{
					ID:          testsutil.GenerateUUID(t),
					CreatedAt:   time.Now(),
					Domain:      testsutil.GenerateUUID(t),
					ParentGroup: testsutil.GenerateUUID(t),
				},
			},
			err: nil,
		},
		{
			desc:     "create channel with failed to save",
			channel:  validChannel,
			saveResp: []channels.Channel{},
			saveErr:  errors.ErrMalformedEntity,
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:    " create channel with failed to add policies",
			channel: validChannel,
			saveResp: []channels.Channel{
				{
					ID:        testsutil.GenerateUUID(t),
					CreatedAt: time.Now(),
					Domain:    validID,
				},
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAddPolicies,
		},
		{
			desc:    " create channel with failed to add policies and failed rollback",
			channel: validChannel,
			saveResp: []channels.Channel{
				{
					ID:        testsutil.GenerateUUID(t),
					CreatedAt: time.Now(),
					Domain:    validID,
				},
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			deleteErr:      svcerr.ErrRemoveEntity,
			err:            svcerr.ErrRollbackRepo,
		},
		{
			desc:    "create channel with failed to add roles",
			channel: validChannel,
			saveResp: []channels.Channel{
				{
					ID:        testsutil.GenerateUUID(t),
					CreatedAt: time.Now(),
					Domain:    validID,
				},
			},
			addRoleErr: svcerr.ErrCreateEntity,
			err:        svcerr.ErrAddPolicies,
		},
		{
			desc:    "create channels with failed to add roles and failed to delete policies",
			channel: validChannel,
			saveResp: []channels.Channel{
				{
					ID:        testsutil.GenerateUUID(t),
					CreatedAt: time.Now(),
					Domain:    validID,
				},
			},
			addRoleErr:        svcerr.ErrCreateEntity,
			deletePoliciesErr: svcerr.ErrRemoveEntity,
			err:               errRollbackRoles,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.saveResp, tc.saveErr)
			policyCall := policies.On("AddPolicies", context.Background(), mock.Anything).Return(tc.addPoliciesErr)
			policyCall1 := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesErr)
			repoCall1 := repo.On("AddRoles", context.Background(), mock.Anything).Return([]roles.RoleProvision{}, tc.addRoleErr)
			repoCall2 := repo.On("Remove", context.Background(), mock.Anything).Return(tc.deleteErr)
			_, _, err := svc.CreateChannels(context.Background(), validSession, tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v but got %v", tc.err, err))
			if err == nil {
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

func TestViewChannel(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc     string
		id       string
		repoResp channels.Channel
		repoErr  error
		err      error
	}{
		{
			desc:     "view channel successfully",
			id:       validChannel.ID,
			repoResp: validChannel,
		},
		{
			desc:    "view channel with failed to retrieve",
			id:      testsutil.GenerateUUID(t),
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.repoResp, tc.repoErr)
			got, err := svc.ViewChannel(context.Background(), validSession, tc.id)
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

func TestUpdateChannel(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc     string
		channel  channels.Channel
		repoResp channels.Channel
		repoErr  error
		err      error
	}{
		{
			desc: "update channel successfully",
			channel: channels.Channel{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			repoResp: validChannel,
		},
		{
			desc: "update channel with repo error",
			channel: channels.Channel{
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
			got, err := svc.UpdateChannel(context.Background(), validSession, tc.channel)
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

func TestUpdateChannelTags(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc      string
		updateReq channels.Channel
		repoResp  channels.Channel
		repoErr   error
		err       error
	}{
		{
			desc: "update channel tags successfully",
			updateReq: channels.Channel{
				ID:   testsutil.GenerateUUID(t),
				Tags: []string{"tag1", "tag2"},
			},
			repoResp: channels.Channel{
				ID:   testsutil.GenerateUUID(t),
				Tags: []string{"tag1", "tag2"},
			},
		},
		{
			desc: "update channel tags with repo error",
			updateReq: channels.Channel{
				ID:   testsutil.GenerateUUID(t),
				Tags: []string{"tag1", "tag2"},
			},
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			got, err := svc.UpdateChannelTags(context.Background(), validSession, tc.updateReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "UpdateTags", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
			}
			repoCall.Unset()
		})
	}
}

func TestEnableChannel(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc         string
		id           string
		retrieveResp channels.Channel
		retrieveErr  error
		changeResp   channels.Channel
		changeErr    error
		err          error
	}{
		{
			desc: "enable channel successfully",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{
				Status: clients.DisabledStatus,
			},
			changeResp: validChannel,
		},
		{
			desc: "enable channel with enabled channel",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{
				Status: clients.EnabledStatus,
			},
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:         "enable channel with retrieve error",
			id:           testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
		{
			desc: "enable channel with change status error",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{
				Status: clients.DisabledStatus,
			},
			changeErr: repoerr.ErrNotFound,
			err:       repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.EnableChannel(context.Background(), validSession, tc.id)
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

func TestDisableChannel(t *testing.T) {
	svc := newService(t)

	cases := []struct {
		desc         string
		id           string
		retrieveResp channels.Channel
		retrieveErr  error
		changeResp   channels.Channel
		changeErr    error
		err          error
	}{
		{
			desc: "disable channel successfully",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{
				Status: clients.EnabledStatus,
			},
			changeResp: validChannel,
		},
		{
			desc: "disable channel with disabled channel",
			id:   testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{
				Status: clients.DisabledStatus,
			},
			err: errors.ErrStatusAlreadyAssigned,
		},
		{
			desc:         "disable channel with retrieve error",
			id:           testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{},
			retrieveErr:  repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
		{
			desc:         "disable channel with change status error",
			id:           testsutil.GenerateUUID(t),
			retrieveResp: channels.Channel{Status: clients.EnabledStatus},
			changeErr:    repoerr.ErrNotFound,
			err:          repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), tc.id).Return(tc.retrieveResp, tc.retrieveErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.changeResp, tc.changeErr)
			got, err := svc.DisableChannel(context.Background(), validSession, tc.id)
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

func TestListChannels(t *testing.T) {
	svc := newService(t)

	adminID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	nonAdminID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc                string
		userKind            string
		session             smqauthn.Session
		page                channels.PageMetadata
		retrieveAllResponse channels.Page
		response            channels.Page
		id                  string
		size                uint64
		listObjectsErr      error
		retrieveAllErr      error
		listPermissionsErr  error
		err                 error
	}{
		{
			desc:     "list all channels successfully as non admin",
			userKind: "non-admin",
			session:  smqauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			retrieveAllResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Channels: []channels.Channel{validChannel, validChannel},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Channels: []channels.Channel{validChannel, validChannel},
			},
			err: nil,
		},
		{
			desc:     "list all channels as non admin with failed to retrieve all",
			userKind: "non-admin",
			session:  smqauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			retrieveAllResponse: channels.Page{},
			response:            channels.Page{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all channels as non admin with failed super admin",
			userKind: "non-admin",
			session:  smqauthn.Session{UserID: nonAdminID, DomainID: domainID, SuperAdmin: false},
			id:       nonAdminID,
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			response: channels.Page{},
			err:      nil,
		},
		{
			desc:     "list all channels as non admin with failed to list objects",
			userKind: "non-admin",
			id:       nonAdminID,
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			retrieveAllErr: repoerr.ErrNotFound,
			response:       channels.Page{},
			listObjectsErr: svcerr.ErrNotFound,
			err:            svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		retrieveAllCall := repo.On("RetrieveAll", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		retrieveUserClientsCall := repo.On("RetrieveUserChannels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListChannels(context.Background(), tc.session, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		retrieveAllCall.Unset()
		retrieveUserClientsCall.Unset()
	}

	cases2 := []struct {
		desc                string
		userKind            string
		session             smqauthn.Session
		page                channels.PageMetadata
		retrieveAllResponse channels.Page
		response            channels.Page
		id                  string
		size                uint64
		listObjectsErr      error
		retrieveAllErr      error
		listPermissionsErr  error
		err                 error
	}{
		{
			desc:     "list all clients as admin successfully",
			userKind: "admin",
			id:       adminID,
			session:  smqauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
				Domain: domainID,
			},
			retrieveAllResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Channels: []channels.Channel{validChannel, validChannel},
			},
			response: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Channels: []channels.Channel{validChannel, validChannel},
			},
			err: nil,
		},
		{
			desc:     "list all clients as admin with failed to retrieve all",
			userKind: "admin",
			id:       adminID,
			session:  smqauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
				Domain: domainID,
			},
			retrieveAllResponse: channels.Page{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
		{
			desc:     "list all clients as admin with failed to list clients",
			userKind: "admin",
			id:       adminID,
			session:  smqauthn.Session{UserID: adminID, DomainID: domainID, SuperAdmin: true},
			page: channels.PageMetadata{
				Offset: 0,
				Limit:  100,
				Domain: domainID,
			},
			retrieveAllResponse: channels.Page{},
			retrieveAllErr:      repoerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases2 {
		retrieveAllCall := repo.On("RetrieveAll", mock.Anything, mock.Anything).Return(tc.retrieveAllResponse, tc.retrieveAllErr)
		page, err := svc.ListChannels(context.Background(), tc.session, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		retrieveAllCall.Unset()
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(t)

	deletedChannel := validChannel
	deletedChannel.Status = clients.DeletedStatus

	channelWithParent := deletedChannel
	channelWithParent.ParentGroup = testsutil.GenerateUUID(t)

	cases := []struct {
		desc                  string
		id                    string
		connectionsRes        bool
		connectionsErr        error
		removeConnectionsErr  error
		changeStatusRes       channels.Channel
		changeStatusErr       error
		deletePoliciesErr     error
		deletePolicyFilterErr error
		removeErr             error
		err                   error
	}{
		{
			desc:            "remove channel without connections successfully",
			id:              validChannel.ID,
			connectionsRes:  false,
			changeStatusRes: deletedChannel,
			err:             nil,
		},
		{
			desc:           "remove channel with connections successfully",
			id:             validChannel.ID,
			connectionsRes: true,
			err:            nil,
		},
		{
			desc:            "remove channel with parent group successfully",
			id:              channelWithParent.ID,
			connectionsRes:  false,
			changeStatusRes: channelWithParent,
			err:             nil,
		},
		{
			desc:           "remove channel with failed check on connections",
			id:             validChannel.ID,
			connectionsErr: repoerr.ErrNotFound,
			err:            svcerr.ErrRemoveEntity,
		},
		{
			desc:                 "remove channel with failed to remove connections",
			id:                   validChannel.ID,
			connectionsRes:       true,
			removeConnectionsErr: svcerr.ErrAuthorization,
			err:                  svcerr.ErrRemoveEntity,
		},
		{
			desc:            "remove channel with failed to change status",
			id:              validChannel.ID,
			connectionsRes:  false,
			changeStatusErr: repoerr.ErrNotFound,
			err:             repoerr.ErrNotFound,
		},
		{
			desc:              "remove channel with failed to delete policies",
			id:                validChannel.ID,
			connectionsRes:    false,
			changeStatusRes:   deletedChannel,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrDeletePolicies,
		},
		{
			desc:                  "remove channel with failed to delete policy filter",
			id:                    validChannel.ID,
			connectionsRes:        false,
			changeStatusRes:       deletedChannel,
			deletePolicyFilterErr: svcerr.ErrAuthorization,
			err:                   svcerr.ErrDeletePolicies,
		},
		{
			desc:            "remove channel with failed to remove",
			id:              validChannel.ID,
			connectionsRes:  false,
			changeStatusRes: deletedChannel,
			removeErr:       repoerr.ErrNotFound,
			err:             svcerr.ErrRemoveEntity,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("DoesChannelHaveConnections", context.Background(), validChannel.ID).Return(tc.connectionsRes, tc.connectionsErr)
			clientsCall := clientsSvc.On("RemoveChannelConnections", context.Background(), &grpcClientsV1.RemoveChannelConnectionsReq{ChannelId: tc.id}).Return(&grpcClientsV1.RemoveChannelConnectionsRes{}, tc.removeConnectionsErr)
			repoCall1 := repo.On("ChangeStatus", context.Background(), channels.Channel{ID: tc.id, Status: smqclients.DeletedStatus}).Return(tc.changeStatusRes, tc.changeStatusErr)
			repoCall2 := repo.On("RetrieveEntitiesRolesActionsMembers", context.Background(), []string{tc.id}).Return([]roles.EntityActionRole{}, []roles.EntityMemberRole{}, nil)
			policyCall := policies.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.deletePoliciesErr)
			policyCall1 := policies.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.deletePolicyFilterErr)
			repoCall3 := repoCall.On("Remove", context.Background(), tc.id).Return(tc.removeErr)
			err := svc.RemoveChannel(context.Background(), validSession, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repoCall.Unset()
			clientsCall.Unset()
			repoCall1.Unset()
			policyCall.Unset()
			policyCall1.Unset()
			repoCall2.Unset()
			repoCall3.Unset()
		})
	}
}

func TestConnect(t *testing.T) {
	svc := newService(t)

	validDomainChannel := validChannel
	validDomainChannel.Domain = validID

	disabledChannel := validChannel
	disabledChannel.Status = clients.DisabledStatus

	cases := []struct {
		desc                     string
		channelIDs               []string
		thingIDs                 []string
		connTypes                []connections.ConnType
		repoConn                 channels.Connection
		clientsConn              []*grpcCommonV1.Connection
		retrieveByIDRes          channels.Channel
		retrieveByIDErr          error
		retrieveEntityRes        *grpcCommonV1.RetrieveEntityRes
		retrieveEntityErr        error
		checkConnErr             error
		addClientConnectionsErr  error
		addChannelConnectionsErr error
		err                      error
	}{
		{
			desc:            "connect successfully",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			checkConnErr: repoerr.ErrNotFound,
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			clientsConn: []*grpcCommonV1.Connection{
				{
					ClientId:  validID,
					ChannelId: validChannel.ID,
					DomainId:  validID,
					Type:      uint32(connections.Publish),
				},
			},
			err: nil,
		},
		{
			desc:            "connect with failed to retrieve channel",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: channels.Channel{},
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrNotFound,
		},
		{
			desc:            "connect to disabled channel",
			channelIDs:      []string{disabledChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: disabledChannel,
			err:             svcerr.ErrCreateEntity,
		},
		{
			desc:            "connect with different domain",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: validChannel,
			err:             svcerr.ErrCreateEntity,
		},
		{
			desc:              "connect with failed to retrieve entity",
			channelIDs:        []string{validChannel.ID},
			thingIDs:          []string{validID},
			retrieveByIDRes:   validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{},
			retrieveEntityErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:            "connect with disabled client",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.DisabledStatus),
				},
			},
			err: svcerr.ErrCreateEntity,
		},
		{
			desc:            "connect with client from different domain",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: testsutil.GenerateUUID(t),
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: svcerr.ErrCreateEntity,
		},
		{
			desc:            "connect with existing connection",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			checkConnErr: nil,
			err:          svcerr.ErrConflict,
		},
		{
			desc:            "connect with failed to check connection",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			checkConnErr: repoerr.ErrMalformedEntity,
			err:          svcerr.ErrCreateEntity,
		},
		{
			desc:            "connect with failed to add client connections",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			checkConnErr: repoerr.ErrNotFound,
			clientsConn: []*grpcCommonV1.Connection{
				{
					ClientId:  validID,
					ChannelId: validChannel.ID,
					DomainId:  validID,
					Type:      uint32(connections.Publish),
				},
			},
			addClientConnectionsErr: svcerr.ErrAuthorization,
			err:                     svcerr.ErrCreateEntity,
		},
		{
			desc:            "connect with failed to add channel connections",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			checkConnErr: repoerr.ErrNotFound,
			clientsConn: []*grpcCommonV1.Connection{
				{
					ClientId:  validID,
					ChannelId: validChannel.ID,
					DomainId:  validID,
					Type:      uint32(connections.Publish),
				},
			},
			addChannelConnectionsErr: svcerr.ErrAuthorization,
			err:                      svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), validChannel.ID).Return(tc.retrieveByIDRes, tc.retrieveByIDErr)
			clientsCall := clientsSvc.On("RetrieveEntity", context.Background(), &grpcCommonV1.RetrieveEntityReq{Id: validID}).Return(tc.retrieveEntityRes, tc.retrieveEntityErr)
			repoCall1 := repo.On("CheckConnection", context.Background(), tc.repoConn).Return(tc.checkConnErr)
			clientsCall1 := clientsSvc.On("AddConnections", context.Background(), &grpcCommonV1.AddConnectionsReq{Connections: tc.clientsConn}).Return(&grpcCommonV1.AddConnectionsRes{}, tc.addClientConnectionsErr)
			repoCall2 := repo.On("AddConnections", context.Background(), []channels.Connection{tc.repoConn}).Return(tc.addChannelConnectionsErr)
			err := svc.Connect(context.Background(), validSession, tc.channelIDs, tc.thingIDs, tc.connTypes)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", tc.err, err))
			repoCall.Unset()
			clientsCall.Unset()
			repoCall1.Unset()
			clientsCall1.Unset()
			repoCall2.Unset()
		})
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(t)

	validDomainChannel := validChannel
	validDomainChannel.Domain = validID

	cases := []struct {
		desc                        string
		channelIDs                  []string
		thingIDs                    []string
		connTypes                   []connections.ConnType
		repoConn                    channels.Connection
		clientsConn                 []*grpcCommonV1.Connection
		retrieveByIDRes             channels.Channel
		retrieveByIDErr             error
		retrieveEntityRes           *grpcCommonV1.RetrieveEntityRes
		retrieveEntityErr           error
		removeClientConnectionsErr  error
		removeChannelConnectionsErr error
		err                         error
	}{
		{
			desc:            "disconnect successfully",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			clientsConn: []*grpcCommonV1.Connection{
				{
					ClientId:  validID,
					ChannelId: validChannel.ID,
					DomainId:  validID,
					Type:      uint32(connections.Publish),
				},
			},
			err: nil,
		},
		{
			desc:            "disconnect with failed to retrieve channel",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: channels.Channel{},
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrNotFound,
		},
		{
			desc:            "disconnect with different domain",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: validChannel,
			err:             svcerr.ErrRemoveEntity,
		},
		{
			desc:              "disconnect with failed to retrieve entity",
			channelIDs:        []string{validChannel.ID},
			thingIDs:          []string{validID},
			retrieveByIDRes:   validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{},
			retrieveEntityErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:            "disconnect with client from different domain",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: testsutil.GenerateUUID(t),
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: svcerr.ErrRemoveEntity,
		},
		{
			desc:            "disconnect with failed to remove client connections",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			clientsConn: []*grpcCommonV1.Connection{
				{
					ClientId:  validID,
					ChannelId: validChannel.ID,
					DomainId:  validID,
					Type:      uint32(connections.Publish),
				},
			},
			removeClientConnectionsErr: svcerr.ErrAuthorization,
			err:                        svcerr.ErrRemoveEntity,
		},
		{
			desc:            "disconnect with failed to remove channel connections",
			channelIDs:      []string{validChannel.ID},
			thingIDs:        []string{validID},
			connTypes:       []connections.ConnType{connections.Publish},
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			repoConn: channels.Connection{
				ClientID:  validID,
				ChannelID: validChannel.ID,
				DomainID:  validID,
				Type:      connections.Publish,
			},
			clientsConn: []*grpcCommonV1.Connection{
				{
					ClientId:  validID,
					ChannelId: validChannel.ID,
					DomainId:  validID,
					Type:      uint32(connections.Publish),
				},
			},
			removeChannelConnectionsErr: svcerr.ErrAuthorization,
			err:                         svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveByID", context.Background(), validChannel.ID).Return(tc.retrieveByIDRes, tc.retrieveByIDErr)
			clientsCall := clientsSvc.On("RetrieveEntity", context.Background(), &grpcCommonV1.RetrieveEntityReq{Id: validID}).Return(tc.retrieveEntityRes, tc.retrieveEntityErr)
			clientsCall1 := clientsSvc.On("RemoveConnections", context.Background(), &grpcCommonV1.RemoveConnectionsReq{Connections: tc.clientsConn}).Return(&grpcCommonV1.RemoveConnectionsRes{}, tc.removeClientConnectionsErr)
			repoCall1 := repo.On("RemoveConnections", context.Background(), []channels.Connection{tc.repoConn}).Return(tc.removeChannelConnectionsErr)
			err := svc.Disconnect(context.Background(), validSession, tc.channelIDs, tc.thingIDs, tc.connTypes)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", tc.err, err))
			repoCall.Unset()
			clientsCall.Unset()
			clientsCall1.Unset()
			repoCall1.Unset()
		})
	}
}

func TestSetParentGroup(t *testing.T) {
	svc := newService(t)

	validDomainChannel := validChannel
	validDomainChannel.Domain = validID

	parentedChannel := validChannel
	parentedChannel.ParentGroup = testsutil.GenerateUUID(t)

	cases := []struct {
		desc              string
		session           authn.Session
		parentGroupID     string
		channelID         string
		retrieveByIDRes   channels.Channel
		retrieveByIDErr   error
		retrieveEntityRes *grpcCommonV1.RetrieveEntityRes
		retrieveEntityErr error
		addPoliciesErr    error
		setParentGroupErr error
		deletePoliciesErr error
		err               error
	}{
		{
			desc:            "set parent group successfully",
			parentGroupID:   parentGroupID,
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: nil,
		},
		{
			desc:            "set parent group with failed to retrieve channel",
			parentGroupID:   parentGroupID,
			channelID:       testsutil.GenerateUUID(t),
			retrieveByIDRes: channels.Channel{},
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrUpdateEntity,
		},
		{
			desc:              "set parent group with failed to retrieve entity",
			parentGroupID:     parentGroupID,
			channelID:         validChannel.ID,
			retrieveByIDRes:   validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{},
			retrieveEntityErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrAuthorization,
		},
		{
			desc:            "set parent group with parent of different domain",
			parentGroupID:   testsutil.GenerateUUID(t),
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: testsutil.GenerateUUID(t),
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: svcerr.ErrUpdateEntity,
		},
		{
			desc:            "set parent groups with disabled domain",
			parentGroupID:   parentGroupID,
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: validID,
					Status:   uint32(clients.DisabledStatus),
				},
			},
			err: svcerr.ErrUpdateEntity,
		},
		{
			desc:            "set parent group of channel with parent group",
			parentGroupID:   parentGroupID,
			channelID:       parentedChannel.ID,
			retrieveByIDRes: parentedChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: svcerr.ErrConflict,
		},
		{
			desc:            "set parent group with failed to add policies",
			parentGroupID:   parentGroupID,
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAddPolicies,
		},
		{
			desc:            "set parent group with failed to set parent group",
			parentGroupID:   parentGroupID,
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			setParentGroupErr: repoerr.ErrNotFound,
			err:               repoerr.ErrNotFound,
		},
		{
			desc:            "set parent group with failed to delete policies",
			parentGroupID:   parentGroupID,
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			retrieveEntityRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       parentGroupID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			setParentGroupErr: repoerr.ErrNotFound,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pols := []policysvc.Policy{
				{
					Domain:      validSession.DomainID,
					SubjectType: policysvc.GroupType,
					Subject:     tc.parentGroupID,
					Relation:    policysvc.ParentGroupRelation,
					ObjectType:  policysvc.ChannelType,
					Object:      tc.channelID,
				},
			}
			repoCall := repo.On("RetrieveByID", context.Background(), tc.channelID).Return(tc.retrieveByIDRes, tc.retrieveByIDErr)
			groupsCall := groupsSvc.On("RetrieveEntity", context.Background(), &grpcCommonV1.RetrieveEntityReq{Id: tc.parentGroupID}).Return(tc.retrieveEntityRes, tc.retrieveEntityErr)
			policyCall := policies.On("AddPolicies", context.Background(), pols).Return(tc.addPoliciesErr)
			repoCall1 := repo.On("SetParentGroup", context.Background(), mock.Anything).Return(tc.setParentGroupErr)
			policyCall1 := policies.On("DeletePolicies", context.Background(), pols).Return(tc.deletePoliciesErr)
			err := svc.SetParentGroup(context.Background(), validSession, tc.parentGroupID, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repoCall.Unset()
			groupsCall.Unset()
			policyCall.Unset()
			repoCall1.Unset()
			policyCall1.Unset()
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	svc := newService(t)

	validDomainChannel := validChannel
	validDomainChannel.Domain = validID

	parentedChannel := validChannel
	parentedChannel.ParentGroup = testsutil.GenerateUUID(t)

	cases := []struct {
		desc                 string
		session              authn.Session
		channelID            string
		retrieveByIDRes      channels.Channel
		retrieveByIDErr      error
		deletePoliciesErr    error
		removeParentGroupErr error
		addPoliciesErr       error
		err                  error
	}{
		{
			desc:            "remove parent group successfully",
			channelID:       validChannel.ID,
			retrieveByIDRes: validDomainChannel,
			err:             nil,
		},
		{
			desc:            "remove parent group with failed to retrieve channel",
			channelID:       testsutil.GenerateUUID(t),
			retrieveByIDRes: channels.Channel{},
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrUpdateEntity,
		},
		{
			desc:              "remove parent group with failed to delete policies",
			channelID:         validChannel.ID,
			retrieveByIDRes:   parentedChannel,
			deletePoliciesErr: svcerr.ErrAuthorization,
			err:               svcerr.ErrDeletePolicies,
		},
		{
			desc:                 "remove parent group with failed to remove parent group",
			channelID:            validChannel.ID,
			retrieveByIDRes:      parentedChannel,
			removeParentGroupErr: repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc:                 "remove parent group with failed to add policies",
			channelID:            validChannel.ID,
			retrieveByIDRes:      parentedChannel,
			removeParentGroupErr: repoerr.ErrNotFound,
			addPoliciesErr:       svcerr.ErrAuthorization,
			err:                  apiutil.ErrRollbackTx,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pols := []policysvc.Policy{
				{
					Domain:      validSession.DomainID,
					SubjectType: policysvc.GroupType,
					Subject:     tc.retrieveByIDRes.ParentGroup,
					Relation:    policysvc.ParentGroupRelation,
					ObjectType:  policysvc.ChannelType,
					Object:      tc.channelID,
				},
			}
			repoCall := repo.On("RetrieveByID", context.Background(), tc.channelID).Return(tc.retrieveByIDRes, tc.retrieveByIDErr)
			policyCall := policies.On("DeletePolicies", context.Background(), pols).Return(tc.deletePoliciesErr)
			repoCall1 := repo.On("RemoveParentGroup", context.Background(), mock.Anything).Return(tc.removeParentGroupErr)
			policyCall1 := policies.On("AddPolicies", context.Background(), pols).Return(tc.addPoliciesErr)
			err := svc.RemoveParentGroup(context.Background(), validSession, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			repoCall.Unset()
			policyCall.Unset()
			repoCall1.Unset()
			policyCall1.Unset()
		})
	}
}
