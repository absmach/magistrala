// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	assignUser        = "assign_user"
	saveGroup         = "save_group"
	deleteGroup       = "delete_group"
	updateGroup       = "update_group"
	retrieveGroupByID = "retrieve_group_by_id"
	retrieveAll       = "retrieve_all_groups"
	retrieveByName    = "retrieve_by_name"
	memberships       = "memberships"
	unassignUser      = "unassign_user"
)

var _ users.GroupRepository = (*groupRepositoryMiddleware)(nil)

type groupRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   users.GroupRepository
}

// GroupRepositoryMiddleware tracks request and their latency, and adds spans to context.
func GroupRepositoryMiddleware(repo users.GroupRepository, tracer opentracing.Tracer) users.GroupRepository {
	return groupRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}
func (grm groupRepositoryMiddleware) Save(ctx context.Context, group users.Group) (users.Group, error) {
	span := createSpan(ctx, grm.tracer, saveGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Save(ctx, group)
}
func (grm groupRepositoryMiddleware) Update(ctx context.Context, group users.Group) error {
	span := createSpan(ctx, grm.tracer, updateGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Update(ctx, group)
}

func (grm groupRepositoryMiddleware) Delete(ctx context.Context, groupID string) error {
	span := createSpan(ctx, grm.tracer, deleteGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Delete(ctx, groupID)
}

func (grm groupRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (users.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveGroupByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByID(ctx, id)
}

func (grm groupRepositoryMiddleware) RetrieveByName(ctx context.Context, name string) (users.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveByName)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByName(ctx, name)
}

func (grm groupRepositoryMiddleware) RetrieveAllWithAncestors(ctx context.Context, groupID string, offset, limit uint64, um users.Metadata) (users.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAllWithAncestors(ctx, groupID, offset, limit, um)
}

func (grm groupRepositoryMiddleware) RetrieveMemberships(ctx context.Context, userID string, offset, limit uint64, um users.Metadata) (users.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, memberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveMemberships(ctx, userID, offset, limit, um)
}

func (grm groupRepositoryMiddleware) Unassign(ctx context.Context, userID, groupID string) error {
	span := createSpan(ctx, grm.tracer, unassignUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Unassign(ctx, userID, groupID)
}

func (grm groupRepositoryMiddleware) Assign(ctx context.Context, userID, groupID string) error {
	span := createSpan(ctx, grm.tracer, assignUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Assign(ctx, userID, groupID)
}
