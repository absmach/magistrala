// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// The DeleteHandler is a cron job that runs periodically to delete users that have been marked as deleted
// for a certain period of time together with the user's policies from the auth service.
// The handler runs in a separate goroutine and checks for users that have been marked as deleted for a certain period of time.
// If the user has been marked as deleted for more than the specified period,
// the handler deletes the user's policies from the auth service and deletes the user from the database.

package users

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/users/postgres"
)

const defLimit = uint64(100)

type handler struct {
	clients       postgres.Repository
	policy        magistrala.PolicyServiceClient
	checkInterval time.Duration
	deleteAfter   time.Duration
	logger        *slog.Logger
}

func NewDeleteHandler(ctx context.Context, clients postgres.Repository, policyClient magistrala.PolicyServiceClient, defCheckInterval, deleteAfter time.Duration, logger *slog.Logger) {
	handler := &handler{
		clients:       clients,
		policy:        policyClient,
		checkInterval: defCheckInterval,
		deleteAfter:   deleteAfter,
		logger:        logger,
	}

	go func() {
		ticker := time.NewTicker(handler.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handler.handle(ctx)
			}
		}
	}()
}

func (h *handler) handle(ctx context.Context) {
	pm := mgclients.Page{Limit: defLimit, Offset: 0, Status: mgclients.DeletedStatus}

	for {
		dbUsers, err := h.clients.RetrieveAll(ctx, pm)
		if err != nil {
			h.logger.Error("failed to retrieve users", slog.Any("error", err))
			break
		}
		if dbUsers.Total == 0 {
			break
		}

		for _, u := range dbUsers.Clients {
			if time.Since(u.UpdatedAt) < h.deleteAfter {
				continue
			}

			deletedRes, err := h.policy.DeleteUserPolicies(ctx, &magistrala.DeleteUserPoliciesReq{
				Id: u.ID,
			})
			if err != nil {
				h.logger.Error("failed to delete user policies", slog.Any("error", err))
				continue
			}
			if !deletedRes.Deleted {
				h.logger.Error("failed to delete user policies", slog.Any("error", svcerr.ErrAuthorization))
				continue
			}

			if err := h.clients.Delete(ctx, u.ID); err != nil {
				h.logger.Error("failed to delete user", slog.Any("error", err))
				continue
			}

			h.logger.Info("user deleted", slog.Group("user",
				slog.String("id", u.ID),
				slog.String("name", u.Name),
			))
		}
	}
}
