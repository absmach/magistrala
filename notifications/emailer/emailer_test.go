// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/absmach/magistrala/notifications"
	"github.com/absmach/magistrala/notifications/emailer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
}

func (m *mockUsersClient) FetchUsers(ctx context.Context, userIDs []string) (map[string]emailer.User, error) {
	args := m.Called(ctx, userIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]emailer.User), args.Error(1)
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
				usersClient.On("FetchUsers", mock.Anything, mock.MatchedBy(func(userIDs []string) bool {
					return len(userIDs) == 2 &&
						((userIDs[0] == inviterID && userIDs[1] == inviteeID) ||
							(userIDs[0] == inviteeID && userIDs[1] == inviterID))
				})).Return(map[string]emailer.User{
					inviterID: {
						ID:        inviterID,
						Email:     inviterEmail,
						FirstName: inviterFirst,
						LastName:  inviterLast,
					},
					inviteeID: {
						ID:        inviteeID,
						Email:     inviteeEmail,
						FirstName: inviteeFirst,
						LastName:  inviteeLast,
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
				usersClient.On("FetchUsers", mock.Anything, mock.MatchedBy(func(userIDs []string) bool {
					return len(userIDs) == 2
				})).Return(map[string]emailer.User{
					inviterID: {
						ID:        inviterID,
						Email:     inviterEmail,
						FirstName: inviterFirst,
						LastName:  inviterLast,
					},
					inviteeID: {
						ID:        inviteeID,
						Email:     inviteeEmail,
						FirstName: inviteeFirst,
						LastName:  inviteeLast,
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
				usersClient.On("FetchUsers", mock.Anything, mock.MatchedBy(func(userIDs []string) bool {
					return len(userIDs) == 2
				})).Return(map[string]emailer.User{
					inviterID: {
						ID:        inviterID,
						Email:     inviterEmail,
						FirstName: inviterFirst,
						LastName:  inviterLast,
					},
					inviteeID: {
						ID:        inviteeID,
						Email:     inviteeEmail,
						FirstName: inviteeFirst,
						LastName:  inviteeLast,
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
				usersClient.On("FetchUsers", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("atom error")).Once()
			},
			expectedError: fmt.Errorf("atom error"),
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
