// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/internal/groups"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	assign               = "assign"
	saveGroup            = "save_group"
	deleteGroup          = "delete_group"
	updateGroup          = "update_group"
	retrieveByID         = "retrieve_by_id"
	retrieveAllAncestors = "retrieve_all_ancestors"
	retrieveAllChildren  = "retrieve_all_children"
	retrieveAll          = "retrieve_all_groups"
	retrieveByName       = "retrieve_by_name"
	memberships          = "memberships"
	members              = "members"
	unassign             = "unassign"
)

var _ groups.Repository = (*groupRepositoryMiddleware)(nil)

type groupRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   groups.Repository
}

// GroupRepositoryMiddleware tracks request and their latency, and adds spans to context.
func GroupRepositoryMiddleware(tracer opentracing.Tracer, gr groups.Repository) groups.Repository {
	return groupRepositoryMiddleware{
		tracer: tracer,
		repo:   gr,
	}
}

func (grm groupRepositoryMiddleware) Save(ctx context.Context, g groups.Group) (groups.Group, error) {
	span := createSpan(ctx, grm.tracer, saveGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Save(ctx, g)
}

func (grm groupRepositoryMiddleware) Update(ctx context.Context, g groups.Group) (groups.Group, error) {
	span := createSpan(ctx, grm.tracer, updateGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Update(ctx, g)
}

func (grm groupRepositoryMiddleware) Delete(ctx context.Context, groupID string) error {
	span := createSpan(ctx, grm.tracer, deleteGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Delete(ctx, groupID)
}

func (grm groupRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (groups.Group, error) {
	span := createSpan(ctx, grm.tracer, retrieveByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveByID(ctx, id)
}

func (grm groupRepositoryMiddleware) RetrieveAllParents(ctx context.Context, groupID string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllAncestors)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAllParents(ctx, groupID, level, gm)
}

func (grm groupRepositoryMiddleware) RetrieveAllChildren(ctx context.Context, groupID string, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveAllChildren)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAllChildren(ctx, groupID, level, gm)
}

func (grm groupRepositoryMiddleware) RetrieveAll(ctx context.Context, level uint64, gm groups.Metadata) (groups.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, retrieveAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.RetrieveAll(ctx, level, gm)
}

func (grm groupRepositoryMiddleware) Memberships(ctx context.Context, memberID string, offset, limit uint64, gm groups.Metadata) (groups.GroupPage, error) {
	span := createSpan(ctx, grm.tracer, memberships)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Memberships(ctx, memberID, offset, limit, gm)
}

func (grm groupRepositoryMiddleware) Members(ctx context.Context, memberID string, offset, limit uint64, gm groups.Metadata) (groups.MemberPage, error) {
	span := createSpan(ctx, grm.tracer, members)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Members(ctx, memberID, offset, limit, gm)
}

func (grm groupRepositoryMiddleware) Unassign(ctx context.Context, memberID, groupID string) error {
	span := createSpan(ctx, grm.tracer, unassign)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Unassign(ctx, memberID, groupID)
}

func (grm groupRepositoryMiddleware) Assign(ctx context.Context, memberID, groupID string) error {
	span := createSpan(ctx, grm.tracer, assign)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return grm.repo.Assign(ctx, memberID, groupID)
}
