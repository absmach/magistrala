// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/absmach/magistrala/notifications"
	"github.com/absmach/magistrala/notifications/middleware"
	"github.com/absmach/magistrala/notifications/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLoggingMiddleware(t *testing.T) {
	notifier := new(mocks.Notifier)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	lm := middleware.NewLogging(notifier, logger)

	cases := []struct {
		desc         string
		notification notifications.Notification
		expectedErr  error
	}{
		{
			desc: "send invitation notification successfully",
			notification: notifications.Notification{
				Type:       notifications.Invitation,
				InviterID:  "inviter-1",
				InviteeID:  "invitee-1",
				DomainID:   "domain-1",
				DomainName: "Test Domain",
				RoleID:     "role-1",
				RoleName:   "Admin",
			},
			expectedErr: nil,
		},
		{
			desc: "send acceptance notification successfully",
			notification: notifications.Notification{
				Type:       notifications.Acceptance,
				InviterID:  "inviter-1",
				InviteeID:  "invitee-1",
				DomainID:   "domain-1",
				DomainName: "Test Domain",
				RoleID:     "role-1",
				RoleName:   "Admin",
			},
			expectedErr: nil,
		},
		{
			desc: "send rejection notification successfully",
			notification: notifications.Notification{
				Type:       notifications.Rejection,
				InviterID:  "inviter-1",
				InviteeID:  "invitee-1",
				DomainID:   "domain-1",
				DomainName: "Test Domain",
				RoleID:     "role-1",
				RoleName:   "Admin",
			},
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			notifier.On("Notify", mock.Anything, tc.notification).Return(tc.expectedErr).Once()
			err := lm.Notify(context.Background(), tc.notification)
			assert.Equal(t, tc.expectedErr, err)
			notifier.AssertExpectations(t)
		})
	}
}
