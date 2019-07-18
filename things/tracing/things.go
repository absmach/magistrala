//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveThingOp               = "save_thing"
	updateThingOp             = "update_thing"
	updateThingKeyOp          = "update_thing_by_key"
	retrieveThingByIDOp       = "retrieve_thing_by_id"
	retrieveThingByKeyOp      = "retrieve_thing_by_key"
	retrieveAllThingsOp       = "retrieve_all_things"
	retrieveThingsByChannelOp = "retrieve_things_by_chan"
	removeThingOp             = "remove_thing"
	retrieveThingIDByKeyOp    = "retrieve_id_by_key"
)

var (
	_ things.ThingRepository = (*thingRepositoryMiddleware)(nil)
	_ things.ThingCache      = (*thingCacheMiddleware)(nil)
)

type thingRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.ThingRepository
}

// ThingRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func ThingRepositoryMiddleware(tracer opentracing.Tracer, repo things.ThingRepository) things.ThingRepository {
	return thingRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm thingRepositoryMiddleware) Save(ctx context.Context, th things.Thing) (string, error) {
	span := createSpan(ctx, trm.tracer, saveThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Save(ctx, th)
}

func (trm thingRepositoryMiddleware) Update(ctx context.Context, th things.Thing) error {
	span := createSpan(ctx, trm.tracer, updateThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, th)
}

func (trm thingRepositoryMiddleware) UpdateKey(ctx context.Context, owner, id, key string) error {
	span := createSpan(ctx, trm.tracer, updateThingKeyOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.UpdateKey(ctx, owner, id, key)
}

func (trm thingRepositoryMiddleware) RetrieveByID(ctx context.Context, owner, id string) (things.Thing, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByID(ctx, owner, id)
}

func (trm thingRepositoryMiddleware) RetrieveByKey(ctx context.Context, key string) (string, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingByKeyOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByKey(ctx, key)
}

func (trm thingRepositoryMiddleware) RetrieveAll(ctx context.Context, owner string, offset, limit uint64, name string) (things.ThingsPage, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllThingsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveAll(ctx, owner, offset, limit, name)
}

func (trm thingRepositoryMiddleware) RetrieveByChannel(ctx context.Context, owner, channel string, offset, limit uint64) (things.ThingsPage, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingsByChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByChannel(ctx, owner, channel, offset, limit)
}

func (trm thingRepositoryMiddleware) Remove(ctx context.Context, owner, id string) error {
	span := createSpan(ctx, trm.tracer, removeThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Remove(ctx, owner, id)
}

type thingCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  things.ThingCache
}

// ThingCacheMiddleware tracks request and their latency, and adds spans
// to context.
func ThingCacheMiddleware(tracer opentracing.Tracer, cache things.ThingCache) things.ThingCache {
	return thingCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (tcm thingCacheMiddleware) Save(ctx context.Context, thingKey string, thingID string) error {
	span := createSpan(ctx, tcm.tracer, saveThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.Save(ctx, thingKey, thingID)
}

func (tcm thingCacheMiddleware) ID(ctx context.Context, thingKey string) (string, error) {
	span := createSpan(ctx, tcm.tracer, retrieveThingIDByKeyOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.ID(ctx, thingKey)
}

func (tcm thingCacheMiddleware) Remove(ctx context.Context, thingID string) error {
	span := createSpan(ctx, tcm.tracer, removeThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.Remove(ctx, thingID)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}

	return tracer.StartSpan(opName)
}
