// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// The DeleteHandler is a cron job that runs periodically to delete domains that have been marked as deleted
// for a certain period of time together with the domain's policies from the auth service.
// The handler runs in a separate goroutine and checks for domains that have been marked as deleted for a certain period of time.
// If the domain has been marked as deleted for more than the specified period,
// the handler deletes the domain's policies from the auth service and deletes the domain from the database.

package domains

import (
	"context"
	"log/slog"
	"time"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	grpcGroupsV1 "github.com/absmach/supermq/api/grpc/groups/v1"
	"github.com/absmach/supermq/pkg/policies"
)

const defLimit = uint64(100)

type handler struct {
	domains       Repository
	channels      grpcChannelsV1.ChannelsServiceClient
	clients       grpcClientsV1.ClientsServiceClient
	groups        grpcGroupsV1.GroupsServiceClient
	policies      policies.Service
	checkInterval time.Duration
	deleteAfter   time.Duration
	logger        *slog.Logger
}

func NewDeleteHandler(ctx context.Context, domains Repository, policyService policies.Service, channelsClient grpcChannelsV1.ChannelsServiceClient, clientsClient grpcClientsV1.ClientsServiceClient, groupsClient grpcGroupsV1.GroupsServiceClient, checkInterval, deleteAfter time.Duration, logger *slog.Logger) {
	handler := &handler{
		domains:       domains,
		channels:      channelsClient,
		clients:       clientsClient,
		groups:        groupsClient,
		policies:      policyService,
		checkInterval: checkInterval,
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
	pm := Page{Limit: defLimit, Offset: 0, Status: DeletedStatus}

	for {
		domainsPage, err := h.domains.ListDomains(ctx, pm)
		if err != nil {
			h.logger.Error("failed to list deleted domains", "error", err)
			break
		}
		if domainsPage.Total == 0 {
			break
		}

		for _, domain := range domainsPage.Domains {
			if time.Since(domain.UpdatedAt) < h.deleteAfter {
				continue
			}

			res, err := h.channels.DeleteDomainChannels(ctx, &grpcCommonV1.DeleteDomainEntitiesReq{
				DomainId: domain.ID,
			})
			if err != nil || !res.GetDeleted() {
				h.logger.Error("failed to delete domain channels", slog.String("domain_id", domain.ID), slog.String("error", err.Error()))
				continue
			}

			res, err = h.clients.DeleteDomainClients(ctx, &grpcCommonV1.DeleteDomainEntitiesReq{
				DomainId: domain.ID,
			})
			if err != nil || !res.GetDeleted() {
				h.logger.Error("failed to delete domain clients", slog.String("domain_id", domain.ID), slog.String("error", err.Error()))
				continue
			}

			res, err = h.groups.DeleteDomainGroups(ctx, &grpcCommonV1.DeleteDomainEntitiesReq{
				DomainId: domain.ID,
			})
			if err != nil || !res.GetDeleted() {
				h.logger.Error("failed to delete domain groups", slog.String("domain_id", domain.ID), slog.String("error", err.Error()))
				continue
			}

			if err := h.deleteDomainPolicies(ctx, domain.ID); err != nil {
				h.logger.Error("failed to delete domain policies", slog.String("domain_id", domain.ID), slog.String("error", err.Error()))
				continue
			}

			if err := h.domains.DeleteDomain(ctx, domain.ID); err != nil {
				h.logger.Error("failed to delete domain", slog.String("domain_id", domain.ID), slog.String("error", err.Error()))
				continue
			}

			h.logger.Info("domain deleted", slog.Group("domain",
				slog.String("id", domain.ID),
				slog.String("name", domain.Name),
			))
		}
	}
}

func (h *handler) deleteDomainPolicies(ctx context.Context, domainID string) error {
	ears, emrs, err := h.domains.RetrieveEntitiesRolesActionsMembers(ctx, []string{domainID})
	if err != nil {
		return err
	}
	deletePolicies := []policies.Policy{}
	for _, ear := range ears {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:         ear.RoleID,
			SubjectRelation: policies.MemberRelation,
			SubjectType:     policies.RoleType,
			Relation:        ear.Action,
			ObjectType:      policies.DomainType,
			Object:          ear.EntityID,
		})
	}
	for _, emr := range emrs {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:     policies.EncodeDomainUserID(domainID, emr.MemberID),
			SubjectType: policies.UserType,
			Relation:    policies.MemberRelation,
			ObjectType:  policies.RoleType,
			Object:      emr.RoleID,
		})
	}
	if err := h.policies.DeletePolicies(ctx, deletePolicies); err != nil {
		return err
	}

	filterDeletePolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     domainID,
		},
		{
			ObjectType: policies.DomainType,
			Object:     domainID,
		},
	}
	for _, filter := range filterDeletePolicies {
		if err := h.policies.DeletePolicyFilter(ctx, filter); err != nil {
			return err
		}
	}

	return nil
}
