// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/invitations"
)

var _ invitations.Service = (*logging)(nil)

type logging struct {
	logger *slog.Logger
	svc    invitations.Service
}

func Logging(logger *slog.Logger, svc invitations.Service) invitations.Service {
	return &logging{logger, svc}
}

func (lm *logging) SendInvitation(ctx context.Context, token string, invitation invitations.Invitation) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", invitation.UserID),
			slog.String("domain_id", invitation.DomainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Send invitation failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Send invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.SendInvitation(ctx, token, invitation)
}

func (lm *logging) ViewInvitation(ctx context.Context, token, userID, domainID string) (invitation invitations.Invitation, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", userID),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View invitation failed to complete successfully", args...)
			return
		}
		lm.logger.Info("View invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewInvitation(ctx, token, userID, domainID)
}

func (lm *logging) ListInvitations(ctx context.Context, token string, page invitations.Page) (invs invitations.InvitationPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("offset", page.Offset),
				slog.Uint64("limit", page.Limit),
				slog.Uint64("total", invs.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List invitations failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List invitations completed successfully", args...)
	}(time.Now())
	return lm.svc.ListInvitations(ctx, token, page)
}

func (lm *logging) AcceptInvitation(ctx context.Context, token, domainID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Accept invitation failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Accept invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.AcceptInvitation(ctx, token, domainID)
}

func (lm *logging) DeleteInvitation(ctx context.Context, token, userID, domainID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", userID),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete invitation failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Delete invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteInvitation(ctx, token, userID, domainID)
}
