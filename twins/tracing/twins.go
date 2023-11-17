// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/twins"
	"go.opentelemetry.io/otel/trace"
)

const (
	saveTwinOp                 = "save_twin"
	saveTwinsOp                = "save_twins"
	updateTwinOp               = "update_twin"
	retrieveTwinByIDOp         = "retrieve_twin_by_id"
	retrieveAllTwinsOp         = "retrieve_all_twins"
	retrieveTwinsByAttributeOp = "retrieve_twins_by_attribute"
	removeTwinOp               = "remove_twin"
)

var _ twins.TwinRepository = (*twinRepositoryMiddleware)(nil)

type twinRepositoryMiddleware struct {
	tracer trace.Tracer
	repo   twins.TwinRepository
}

// TwinRepositoryMiddleware tracks request and their latency, and adds spans to context.
func TwinRepositoryMiddleware(tracer trace.Tracer, repo twins.TwinRepository) twins.TwinRepository {
	return twinRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm twinRepositoryMiddleware) Save(ctx context.Context, tw twins.Twin) (string, error) {
	ctx, span := createSpan(ctx, trm.tracer, saveTwinOp)
	defer span.End()

	return trm.repo.Save(ctx, tw)
}

func (trm twinRepositoryMiddleware) Update(ctx context.Context, tw twins.Twin) error {
	ctx, span := createSpan(ctx, trm.tracer, updateTwinOp)
	defer span.End()

	return trm.repo.Update(ctx, tw)
}

func (trm twinRepositoryMiddleware) RetrieveByID(ctx context.Context, twinID string) (twins.Twin, error) {
	ctx, span := createSpan(ctx, trm.tracer, retrieveTwinByIDOp)
	defer span.End()

	return trm.repo.RetrieveByID(ctx, twinID)
}

func (trm twinRepositoryMiddleware) RetrieveAll(ctx context.Context, owner string, offset, limit uint64, name string, metadata twins.Metadata) (twins.Page, error) {
	ctx, span := createSpan(ctx, trm.tracer, retrieveAllTwinsOp)
	defer span.End()

	return trm.repo.RetrieveAll(ctx, owner, offset, limit, name, metadata)
}

func (trm twinRepositoryMiddleware) RetrieveByAttribute(ctx context.Context, channel, subtopic string) ([]string, error) {
	ctx, span := createSpan(ctx, trm.tracer, retrieveAllTwinsOp)
	defer span.End()

	return trm.repo.RetrieveByAttribute(ctx, channel, subtopic)
}

func (trm twinRepositoryMiddleware) Remove(ctx context.Context, twinID string) error {
	ctx, span := createSpan(ctx, trm.tracer, removeTwinOp)
	defer span.End()

	return trm.repo.Remove(ctx, twinID)
}

type twinCacheMiddleware struct {
	tracer trace.Tracer
	cache  twins.TwinCache
}

// TwinCacheMiddleware tracks request and their latency, and adds spans to context.
func TwinCacheMiddleware(tracer trace.Tracer, cache twins.TwinCache) twins.TwinCache {
	return twinCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (tcm twinCacheMiddleware) Save(ctx context.Context, twin twins.Twin) error {
	ctx, span := createSpan(ctx, tcm.tracer, saveTwinOp)
	defer span.End()

	return tcm.cache.Save(ctx, twin)
}

func (tcm twinCacheMiddleware) SaveIDs(ctx context.Context, channel, subtopic string, ids []string) error {
	ctx, span := createSpan(ctx, tcm.tracer, saveTwinsOp)
	defer span.End()

	return tcm.cache.SaveIDs(ctx, channel, subtopic, ids)
}

func (tcm twinCacheMiddleware) Update(ctx context.Context, twin twins.Twin) error {
	ctx, span := createSpan(ctx, tcm.tracer, updateTwinOp)
	defer span.End()

	return tcm.cache.Update(ctx, twin)
}

func (tcm twinCacheMiddleware) IDs(ctx context.Context, channel, subtopic string) ([]string, error) {
	ctx, span := createSpan(ctx, tcm.tracer, retrieveTwinsByAttributeOp)
	defer span.End()

	return tcm.cache.IDs(ctx, channel, subtopic)
}

func (tcm twinCacheMiddleware) Remove(ctx context.Context, twinID string) error {
	ctx, span := createSpan(ctx, tcm.tracer, removeTwinOp)
	defer span.End()

	return tcm.cache.Remove(ctx, twinID)
}

func createSpan(ctx context.Context, tracer trace.Tracer, opName string) (context.Context, trace.Span) {
	return tracer.Start(ctx, opName)
}
