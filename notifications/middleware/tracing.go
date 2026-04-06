// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/notifications"
	smqTracing "github.com/absmach/magistrala/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ notifications.Notifier = (*tracing)(nil)

type tracing struct {
	tracer   trace.Tracer
	notifier notifications.Notifier
}

// NewTracing returns a new notifier with tracing capabilities.
func NewTracing(notifier notifications.Notifier, tracer trace.Tracer) notifications.Notifier {
	return &tracing{tracer, notifier}
}

func (tm *tracing) Notify(ctx context.Context, n notifications.Notification) error {
	spanName := notificationTypeToMethodName(n.Type)
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, spanName, trace.WithAttributes(
		attribute.String("inviter_id", n.InviterID),
		attribute.String("invitee_id", n.InviteeID),
		attribute.String("domain_id", n.DomainID),
		attribute.String("domain_name", n.DomainName),
		attribute.String("role_id", n.RoleID),
		attribute.String("role_name", n.RoleName),
	))
	defer span.End()

	return tm.notifier.Notify(ctx, n)
}

func notificationTypeToMethodName(t notifications.NotificationType) string {
	switch t {
	case notifications.Invitation:
		return "send_invitation_notification"
	case notifications.Acceptance:
		return "send_acceptance_notification"
	case notifications.Rejection:
		return "send_rejection_notification"
	default:
		return "unknown"
	}
}
