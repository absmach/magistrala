// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/notifications"
)

var _ notifications.Notifier = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger   *slog.Logger
	notifier notifications.Notifier
}

// NewLogging adds logging facilities to the notifier.
func NewLogging(notifier notifications.Notifier, logger *slog.Logger) notifications.Notifier {
	return &loggingMiddleware{
		logger:   logger,
		notifier: notifier,
	}
}

func (lm *loggingMiddleware) Notify(ctx context.Context, n notifications.Notification) (err error) {
	defer func(begin time.Time) {
		groupName := notificationTypeToString(n.Type)
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group(groupName,
				slog.String("inviter_id", n.InviterID),
				slog.String("invitee_id", n.InviteeID),
				slog.String("domain_id", n.DomainID),
				slog.String("domain_name", n.DomainName),
				slog.String("role_id", n.RoleID),
				slog.String("role_name", n.RoleName),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Send "+groupName+" notification failed", args...)
			return
		}
		lm.logger.Info("Send "+groupName+" notification completed successfully", args...)
	}(time.Now())

	return lm.notifier.Notify(ctx, n)
}

func notificationTypeToString(t notifications.NotificationType) string {
	switch t {
	case notifications.Invitation:
		return "invitation"
	case notifications.Acceptance:
		return "acceptance"
	case notifications.Rejection:
		return "rejection"
	default:
		return "unknown"
	}
}
