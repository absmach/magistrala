// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/roles"
	rmMW "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/go-chi/chi/v5/middleware"
)

var _ domains.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    domains.Service
	rmMW.RoleManagerLoggingMiddleware
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc domains.Service, logger *slog.Logger) domains.Service {
	rmlm := rmMW.NewRoleManagerLoggingMiddleware("domains", svc, logger)
	return &loggingMiddleware{
		logger:                       logger,
		svc:                          svc,
		RoleManagerLoggingMiddleware: rmlm,
	}
}

func (lm *loggingMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (do domains.Domain, rps []roles.RoleProvision, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("domain",
				slog.String("id", d.ID),
				slog.String("name", d.Name),
			),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Create domain failed", args...)
			return
		}
		lm.logger.Info("Create domain completed successfully", args...)
	}(time.Now())
	return lm.svc.CreateDomain(ctx, session, d)
}

func (lm *loggingMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string) (do domains.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("domain_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Retrieve domain failed", args...)
			return
		}
		lm.logger.Info("Retrieve domain completed successfully", args...)
	}(time.Now())
	return lm.svc.RetrieveDomain(ctx, session, id)
}

func (lm *loggingMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (do domains.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("domain",
				slog.String("id", id),
				slog.Any("name", d.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update domain failed", args...)
			return
		}
		lm.logger.Info("Update domain completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateDomain(ctx, session, id, d)
}

func (lm *loggingMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (do domains.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("domain",
				slog.String("id", id),
				slog.String("name", do.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable domain failed", args...)
			return
		}
		lm.logger.Info("Enable domain completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableDomain(ctx, session, id)
}

func (lm *loggingMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (do domains.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("domain",
				slog.String("id", id),
				slog.String("name", do.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable domain failed", args...)
			return
		}
		lm.logger.Info("Disable domain completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableDomain(ctx, session, id)
}

func (lm *loggingMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (do domains.Domain, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("domain",
				slog.String("id", id),
				slog.String("name", do.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Freeze domain failed", args...)
			return
		}
		lm.logger.Info("Freeze domain completed successfully", args...)
	}(time.Now())
	return lm.svc.FreezeDomain(ctx, session, id)
}

func (lm *loggingMiddleware) ListDomains(ctx context.Context, session authn.Session, page domains.Page) (do domains.DomainsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("page",
				slog.Uint64("limit", page.Limit),
				slog.Uint64("offset", page.Offset),
				slog.Uint64("total", page.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List domains failed", args...)
			return
		}
		lm.logger.Info("List domains completed successfully", args...)
	}(time.Now())
	return lm.svc.ListDomains(ctx, session, page)
}

func (lm *loggingMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("invitee_user_id", invitation.InviteeUserID),
			slog.String("domain_id", invitation.DomainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Send invitation failed", args...)
			return
		}
		lm.logger.Info("Send invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.SendInvitation(ctx, session, invitation)
}

func (lm *loggingMiddleware) ViewInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (invitation domains.Invitation, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("invitee_user_id", inviteeUserID),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View invitation failed", args...)
			return
		}
		lm.logger.Info("View invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewInvitation(ctx, session, inviteeUserID, domainID)
}

func (lm *loggingMiddleware) ListInvitations(ctx context.Context, session authn.Session, pm domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("total", invs.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List invitations failed", args...)
			return
		}
		lm.logger.Info("List invitations completed successfully", args...)
	}(time.Now())
	return lm.svc.ListInvitations(ctx, session, pm)
}

func (lm *loggingMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Accept invitation failed", args...)
			return
		}
		lm.logger.Info("Accept invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.AcceptInvitation(ctx, session, domainID)
}

func (lm *loggingMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Reject invitation failed", args...)
			return
		}
		lm.logger.Info("Reject invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.RejectInvitation(ctx, session, domainID)
}

func (lm *loggingMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("invitee_user_id", inviteeUserID),
			slog.String("domain_id", domainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete invitation failed", args...)
			return
		}
		lm.logger.Info("Delete invitation completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteInvitation(ctx, session, inviteeUserID, domainID)
}
