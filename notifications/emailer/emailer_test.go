// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	grpcUsersV1 "github.com/absmach/supermq/api/grpc/users/v1"
	"github.com/absmach/supermq/notifications"
	"github.com/absmach/supermq/notifications/emailer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

const (
	inviterID  = "inviter-id"
	inviteeID  = "invitee-id"
	domainID   = "domain-id"
	domainName = "Test Domain"
	roleID     = "role-id"
	roleName   = "Admin"

	inviterEmail = "inviter@example.com"
	inviteeEmail = "invitee@example.com"
	inviterFirst = "John"
	inviterLast  = "Doe"
	inviteeFirst = "Jane"
	inviteeLast  = "Smith"

	envTrue = "true"
)

type mockUsersClient struct {
	mock.Mock
	grpcUsersV1.UsersServiceClient
}

func (m *mockUsersClient) RetrieveUsers(ctx context.Context, req *grpcUsersV1.RetrieveUsersReq, opts ...grpc.CallOption) (*grpcUsersV1.RetrieveUsersRes, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcUsersV1.RetrieveUsersRes), args.Error(1)
}

func TestNotify(t *testing.T) {
	if os.Getenv("MG_RUN_EMAIL_TESTS") != envTrue {
		t.Skip("Skipping email tests. Set MG_RUN_EMAIL_TESTS=true to run.")
	}

	usersClient := new(mockUsersClient)

	cfg := emailer.Config{
		FromAddress:        "test@example.com",
		FromName:           "Test Service",
		DomainAltName:      "domain",
		InvitationTemplate: "../../docker/templates/invitation-sent-email.tmpl",
		AcceptanceTemplate: "../../docker/templates/invitation-accepted-email.tmpl",
		RejectionTemplate:  "../../docker/templates/invitation-rejected-email.tmpl",
		EmailHost:          "localhost",
		EmailPort:          "1025",
		EmailUsername:      "",
		EmailPassword:      "",
	}

	notifier, err := emailer.New(usersClient, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, notifier)

	cases := []struct {
		desc          string
		notification  notifications.Notification
		setupMock     func()
		expectedError error
	}{
		{
			desc: "successful invitation notification",
			notification: notifications.Notification{
				Type:       notifications.Invitation,
				InviterID:  inviterID,
				InviteeID:  inviteeID,
				DomainID:   domainID,
				DomainName: domainName,
				RoleID:     roleID,
				RoleName:   roleName,
			},
			setupMock: func() {
				usersClient.On("RetrieveUsers", mock.Anything, mock.MatchedBy(func(req *grpcUsersV1.RetrieveUsersReq) bool {
					return len(req.Ids) == 2 &&
						((req.Ids[0] == inviterID && req.Ids[1] == inviteeID) ||
							(req.Ids[0] == inviteeID && req.Ids[1] == inviterID))
				}), mock.Anything).Return(&grpcUsersV1.RetrieveUsersRes{
					Users: []*grpcUsersV1.User{
						{
							Id:        inviterID,
							Email:     inviterEmail,
							FirstName: inviterFirst,
							LastName:  inviterLast,
						},
						{
							Id:        inviteeID,
							Email:     inviteeEmail,
							FirstName: inviteeFirst,
							LastName:  inviteeLast,
						},
					},
				}, nil).Once()
			},
			expectedError: nil,
		},
		{
			desc: "successful acceptance notification",
			notification: notifications.Notification{
				Type:       notifications.Acceptance,
				InviterID:  inviterID,
				InviteeID:  inviteeID,
				DomainID:   domainID,
				DomainName: domainName,
				RoleID:     roleID,
				RoleName:   roleName,
			},
			setupMock: func() {
				usersClient.On("RetrieveUsers", mock.Anything, mock.MatchedBy(func(req *grpcUsersV1.RetrieveUsersReq) bool {
					return len(req.Ids) == 2
				}), mock.Anything).Return(&grpcUsersV1.RetrieveUsersRes{
					Users: []*grpcUsersV1.User{
						{
							Id:        inviterID,
							Email:     inviterEmail,
							FirstName: inviterFirst,
							LastName:  inviterLast,
						},
						{
							Id:        inviteeID,
							Email:     inviteeEmail,
							FirstName: inviteeFirst,
							LastName:  inviteeLast,
						},
					},
				}, nil).Once()
			},
			expectedError: nil,
		},
		{
			desc: "successful rejection notification",
			notification: notifications.Notification{
				Type:       notifications.Rejection,
				InviterID:  inviterID,
				InviteeID:  inviteeID,
				DomainID:   domainID,
				DomainName: domainName,
				RoleID:     roleID,
				RoleName:   roleName,
			},
			setupMock: func() {
				usersClient.On("RetrieveUsers", mock.Anything, mock.MatchedBy(func(req *grpcUsersV1.RetrieveUsersReq) bool {
					return len(req.Ids) == 2
				}), mock.Anything).Return(&grpcUsersV1.RetrieveUsersRes{
					Users: []*grpcUsersV1.User{
						{
							Id:        inviterID,
							Email:     inviterEmail,
							FirstName: inviterFirst,
							LastName:  inviterLast,
						},
						{
							Id:        inviteeID,
							Email:     inviteeEmail,
							FirstName: inviteeFirst,
							LastName:  inviteeLast,
						},
					},
				}, nil).Once()
			},
			expectedError: nil,
		},
		{
			desc: "failed to fetch users",
			notification: notifications.Notification{
				Type:       notifications.Invitation,
				InviterID:  inviterID,
				InviteeID:  inviteeID,
				DomainID:   domainID,
				DomainName: domainName,
				RoleID:     roleID,
				RoleName:   roleName,
			},
			setupMock: func() {
				usersClient.On("RetrieveUsers", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("grpc error")).Once()
			},
			expectedError: fmt.Errorf("grpc error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.setupMock()
			err := notifier.Notify(context.Background(), tc.notification)
			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			usersClient.AssertExpectations(t)
		})
	}
}
