// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/notifications"
	"github.com/absmach/magistrala/notifications/events"
	"github.com/absmach/magistrala/notifications/mocks"
	smqevents "github.com/absmach/magistrala/pkg/events"
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
)

type testEvent struct {
	data map[string]any
}

func (e testEvent) Encode() (map[string]any, error) {
	return e.data, nil
}

type mockSubscriber struct {
	mock.Mock
}

func (m *mockSubscriber) Subscribe(ctx context.Context, cfg smqevents.SubscriberConfig) error {
	args := m.Called(ctx, cfg)
	return args.Error(0)
}

func (m *mockSubscriber) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestStart(t *testing.T) {
	notifier := new(mocks.Notifier)
	subscriber := new(mockSubscriber)

	subscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Times(3)

	err := events.Start(context.Background(), "test-consumer", subscriber, notifier)
	assert.NoError(t, err)
	subscriber.AssertExpectations(t)
}

func TestHandleInvitationSent(t *testing.T) {
	cases := []struct {
		desc     string
		event    smqevents.Event
		mockCall bool
	}{
		{
			desc: "successful invitation sent handling",
			event: testEvent{
				data: map[string]any{
					"invited_by":      inviterID,
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
					"domain_name":     domainName,
					"role_id":         roleID,
					"role_name":       roleName,
				},
			},
			mockCall: true,
		},
		{
			desc: "missing invited_by",
			event: testEvent{
				data: map[string]any{
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
				},
			},
			mockCall: false,
		},
		{
			desc: "missing invitee_user_id",
			event: testEvent{
				data: map[string]any{
					"invited_by": inviterID,
					"domain_id":  domainID,
				},
			},
			mockCall: false,
		},
		{
			desc: "missing domain_id",
			event: testEvent{
				data: map[string]any{
					"invited_by":      inviterID,
					"invitee_user_id": inviteeID,
				},
			},
			mockCall: false,
		},
		{
			desc: "optional fields with wrong type",
			event: testEvent{
				data: map[string]any{
					"invited_by":      inviterID,
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
					"domain_name":     domainName,
					"role_id":         123,  // wrong type: int instead of string
					"role_name":       true, // wrong type: bool instead of string
				},
			},
			mockCall: true, // Should still process with empty role_id and role_name
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			notifier := new(mocks.Notifier)
			subscriber := new(mockSubscriber)

			var handlerConfig smqevents.SubscriberConfig
			subscriber.On("Subscribe", mock.Anything, mock.MatchedBy(func(cfg smqevents.SubscriberConfig) bool {
				if cfg.Stream == "events.magistrala.invitation.send" {
					handlerConfig = cfg
					return true
				}
				return false
			})).Return(nil).Once()
			subscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Times(2)

			err := events.Start(context.Background(), "test-consumer", subscriber, notifier)
			assert.NoError(t, err)

			if tc.mockCall {
				expectedNotif := notifications.Notification{
					Type:       notifications.Invitation,
					InviterID:  inviterID,
					InviteeID:  inviteeID,
					DomainID:   domainID,
					DomainName: domainName,
					RoleID:     roleID,
					RoleName:   roleName,
				}
				// For the "wrong type" test case, expect empty role fields
				if tc.desc == "optional fields with wrong type" {
					expectedNotif.RoleID = ""
					expectedNotif.RoleName = ""
				}
				notifier.On("Notify", mock.Anything, expectedNotif).Return(nil).Once()
			}

			err = handlerConfig.Handler.Handle(context.Background(), tc.event)
			assert.NoError(t, err)

			if tc.mockCall {
				notifier.AssertExpectations(t)
			}
		})
	}
}

func TestHandleInvitationAccepted(t *testing.T) {
	cases := []struct {
		desc     string
		event    smqevents.Event
		mockCall bool
	}{
		{
			desc: "successful invitation accepted handling",
			event: testEvent{
				data: map[string]any{
					"invited_by":      inviterID,
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
					"domain_name":     domainName,
					"role_id":         roleID,
					"role_name":       roleName,
				},
			},
			mockCall: true,
		},
		{
			desc: "missing invited_by",
			event: testEvent{
				data: map[string]any{
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
				},
			},
			mockCall: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			notifier := new(mocks.Notifier)
			subscriber := new(mockSubscriber)

			var handlerConfig smqevents.SubscriberConfig
			subscriber.On("Subscribe", mock.Anything, mock.MatchedBy(func(cfg smqevents.SubscriberConfig) bool {
				if cfg.Stream == "events.magistrala.invitation.accept" {
					handlerConfig = cfg
					return true
				}
				return false
			})).Return(nil).Once()
			subscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Times(2)

			err := events.Start(context.Background(), "test-consumer", subscriber, notifier)
			assert.NoError(t, err)

			if tc.mockCall {
				expectedNotif := notifications.Notification{
					Type:       notifications.Acceptance,
					InviterID:  inviterID,
					InviteeID:  inviteeID,
					DomainID:   domainID,
					DomainName: domainName,
					RoleID:     roleID,
					RoleName:   roleName,
				}
				notifier.On("Notify", mock.Anything, expectedNotif).Return(nil).Once()
			}

			err = handlerConfig.Handler.Handle(context.Background(), tc.event)
			assert.NoError(t, err)

			if tc.mockCall {
				notifier.AssertExpectations(t)
			}
		})
	}
}

func TestHandleInvitationRejected(t *testing.T) {
	cases := []struct {
		desc     string
		event    smqevents.Event
		mockCall bool
	}{
		{
			desc: "successful invitation rejected handling",
			event: testEvent{
				data: map[string]any{
					"invited_by":      inviterID,
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
					"domain_name":     domainName,
					"role_id":         roleID,
					"role_name":       roleName,
				},
			},
			mockCall: true,
		},
		{
			desc: "missing invited_by",
			event: testEvent{
				data: map[string]any{
					"invitee_user_id": inviteeID,
					"domain_id":       domainID,
				},
			},
			mockCall: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			notifier := new(mocks.Notifier)
			subscriber := new(mockSubscriber)

			var handlerConfig smqevents.SubscriberConfig
			subscriber.On("Subscribe", mock.Anything, mock.MatchedBy(func(cfg smqevents.SubscriberConfig) bool {
				if cfg.Stream == "events.magistrala.invitation.reject" {
					handlerConfig = cfg
					return true
				}
				return false
			})).Return(nil).Once()
			subscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Times(2)

			err := events.Start(context.Background(), "test-consumer", subscriber, notifier)
			assert.NoError(t, err)

			if tc.mockCall {
				expectedNotif := notifications.Notification{
					Type:       notifications.Rejection,
					InviterID:  inviterID,
					InviteeID:  inviteeID,
					DomainID:   domainID,
					DomainName: domainName,
					RoleID:     roleID,
					RoleName:   roleName,
				}
				notifier.On("Notify", mock.Anything, expectedNotif).Return(nil).Once()
			}

			err = handlerConfig.Handler.Handle(context.Background(), tc.event)
			assert.NoError(t, err)

			if tc.mockCall {
				notifier.AssertExpectations(t)
			}
		})
	}
}
