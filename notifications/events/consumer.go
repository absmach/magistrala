// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/notifications"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	// Stream names.
	sendInvitationStream   = "events.magistrala.invitation.send"
	acceptInvitationStream = "events.magistrala.invitation.accept"
	rejectInvitationStream = "events.magistrala.invitation.reject"

	// Event data field keys.
	invitedByKey     = "invited_by"
	inviteeUserIDKey = "invitee_user_id"
	domainIDKey      = "domain_id"
	domainNameKey    = "domain_name"
	roleIDKey        = "role_id"
	roleNameKey      = "role_name"
)

// Start starts consuming invitation events from the event store.
func Start(ctx context.Context, consumer string, sub events.Subscriber, notifier notifications.Notifier) error {
	handlers := []struct {
		stream    string
		notifType notifications.NotificationType
		errorCtx  string
	}{
		{sendInvitationStream, notifications.Invitation, "invitation sent"},
		{acceptInvitationStream, notifications.Acceptance, "invitation accepted"},
		{rejectInvitationStream, notifications.Rejection, "invitation rejected"},
	}

	for _, h := range handlers {
		config := events.SubscriberConfig{
			Consumer: consumer,
			Stream:   h.stream,
			Handler:  handleInvitationEvent(notifier, h.notifType, h.errorCtx),
		}
		if err := sub.Subscribe(ctx, config); err != nil {
			return err
		}
	}

	return nil
}

func handleInvitationEvent(notifier notifications.Notifier, notifType notifications.NotificationType, errorContext string) handleFunc {
	return func(ctx context.Context, event events.Event) error {
		n, err := parseNotificationFromEvent(event, errorContext)
		if err != nil {
			return nil
		}

		n.Type = notifType

		if err := notifier.Notify(ctx, n); err != nil {
			slog.Error("failed to send notification", "error", err, "type", notifType, "context", errorContext)
		}

		return nil
	}
}

func parseNotificationFromEvent(event events.Event, errorContext string) (notifications.Notification, error) {
	data, err := event.Encode()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to encode %s event", errorContext), "error", err)
		return notifications.Notification{}, err
	}

	invitedBy, ok := data[invitedByKey].(string)
	if !ok || invitedBy == "" {
		slog.Error(fmt.Sprintf("missing or invalid %s in %s event", invitedByKey, errorContext))
		return notifications.Notification{}, fmt.Errorf("missing or invalid %s", invitedByKey)
	}

	inviteeUserID, ok := data[inviteeUserIDKey].(string)
	if !ok || inviteeUserID == "" {
		slog.Error(fmt.Sprintf("missing or invalid %s in %s event", inviteeUserIDKey, errorContext))
		return notifications.Notification{}, fmt.Errorf("missing or invalid %s", inviteeUserIDKey)
	}

	domainID, ok := data[domainIDKey].(string)
	if !ok || domainID == "" {
		slog.Error(fmt.Sprintf("missing or invalid %s in %s event", domainIDKey, errorContext))
		return notifications.Notification{}, fmt.Errorf("missing or invalid %s", domainIDKey)
	}

	// Optional fields - log if present but wrong type
	roleID := optionalString(data, roleIDKey, errorContext)
	domainName := optionalString(data, domainNameKey, errorContext)
	roleName := optionalString(data, roleNameKey, errorContext)

	return notifications.Notification{
		InviterID:  invitedBy,
		InviteeID:  inviteeUserID,
		DomainID:   domainID,
		DomainName: domainName,
		RoleID:     roleID,
		RoleName:   roleName,
	}, nil
}

func optionalString(data map[string]any, key, errorContext string) string {
	val, exists := data[key]
	if !exists {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		slog.Warn(fmt.Sprintf("field %s in %s event has wrong type, expected string", key, errorContext), "actual_type", fmt.Sprintf("%T", val))
		return ""
	}
	return strVal
}

type handleFunc func(ctx context.Context, event events.Event) error

func (h handleFunc) Handle(ctx context.Context, event events.Event) error {
	return h(ctx, event)
}

func (h handleFunc) Cancel() error {
	return nil
}
