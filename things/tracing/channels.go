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
	saveChannelOp             = "save_channel"
	updateChannelOp           = "update_channel"
	retrieveChannelByIDOp     = "retrieve_channel_by_id"
	retrieveAllChannelsOp     = "retrieve_all_channels"
	retrieveChannelsByThingOp = "retrieve_channels_by_thing"
	removeChannelOp           = "retrieve_channel"
	connectOp                 = "connect"
	disconnectOp              = "disconnect"
	hasThingOp                = "has_thing"
	hasThingByIDOp            = "has_thing_by_id"
)

var (
	_ things.ChannelRepository = (*channelRepositoryMiddleware)(nil)
	_ things.ChannelCache      = (*channelCacheMiddleware)(nil)
)

type channelRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.ChannelRepository
}

// ChannelRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func ChannelRepositoryMiddleware(tracer opentracing.Tracer, repo things.ChannelRepository) things.ChannelRepository {
	return channelRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (crm channelRepositoryMiddleware) Save(ctx context.Context, ch things.Channel) (string, error) {
	span := createSpan(ctx, crm.tracer, saveChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, ch)
}

func (crm channelRepositoryMiddleware) Update(ctx context.Context, ch things.Channel) error {
	span := createSpan(ctx, crm.tracer, updateChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, ch)
}

func (crm channelRepositoryMiddleware) RetrieveByID(ctx context.Context, owner, id string) (things.Channel, error) {
	span := createSpan(ctx, crm.tracer, retrieveChannelByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByID(ctx, owner, id)
}

func (crm channelRepositoryMiddleware) RetrieveAll(ctx context.Context, owner string, offset, limit uint64, name string) (things.ChannelsPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllChannelsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx, owner, offset, limit, name)
}

func (crm channelRepositoryMiddleware) RetrieveByThing(ctx context.Context, owner, thing string, offset, limit uint64) (things.ChannelsPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveChannelsByThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, owner, thing, offset, limit)
}

func (crm channelRepositoryMiddleware) Remove(ctx context.Context, owner, id string) error {
	span := createSpan(ctx, crm.tracer, removeChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, owner, id)
}

func (crm channelRepositoryMiddleware) Connect(ctx context.Context, owner, chanID, thingID string) error {
	span := createSpan(ctx, crm.tracer, connectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Connect(ctx, owner, chanID, thingID)
}

func (crm channelRepositoryMiddleware) Disconnect(ctx context.Context, owner, chanID, thingID string) error {
	span := createSpan(ctx, crm.tracer, disconnectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Disconnect(ctx, owner, chanID, thingID)
}

func (crm channelRepositoryMiddleware) HasThing(ctx context.Context, chanID, key string) (string, error) {
	span := createSpan(ctx, crm.tracer, hasThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.HasThing(ctx, chanID, key)
}

func (crm channelRepositoryMiddleware) HasThingByID(ctx context.Context, chanID, thingID string) error {
	span := createSpan(ctx, crm.tracer, hasThingByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.HasThingByID(ctx, chanID, thingID)
}

type channelCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  things.ChannelCache
}

// ChannelCacheMiddleware tracks request and their latency, and adds spans
// to context.
func ChannelCacheMiddleware(tracer opentracing.Tracer, cache things.ChannelCache) things.ChannelCache {
	return channelCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (ccm channelCacheMiddleware) Connect(ctx context.Context, chanID, thingID string) error {
	span := createSpan(ctx, ccm.tracer, connectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Connect(ctx, chanID, thingID)
}

func (ccm channelCacheMiddleware) HasThing(ctx context.Context, chanID, thingID string) bool {
	span := createSpan(ctx, ccm.tracer, hasThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.HasThing(ctx, chanID, thingID)
}

func (ccm channelCacheMiddleware) Disconnect(ctx context.Context, chanID, thingID string) error {
	span := createSpan(ctx, ccm.tracer, disconnectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Disconnect(ctx, chanID, thingID)
}

func (ccm channelCacheMiddleware) Remove(ctx context.Context, chanID string) error {
	span := createSpan(ctx, ccm.tracer, removeChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Remove(ctx, chanID)
}
