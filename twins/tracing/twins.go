// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/twins"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveTwinOp                 = "save_twin"
	updateTwinOp               = "update_twin"
	retrieveTwinByIDOp         = "retrieve_twin_by_id"
	retrieveAllTwinsOp         = "retrieve_all_twins"
	retrieveTwinsByAttributeOp = "retrieve_twins_by_attribute"
	removeTwinOp               = "remove_twin"
)

var (
	_ twins.TwinRepository = (*twinRepositoryMiddleware)(nil)
)

type twinRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   twins.TwinRepository
}

// TwinRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func TwinRepositoryMiddleware(tracer opentracing.Tracer, repo twins.TwinRepository) twins.TwinRepository {
	return twinRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm twinRepositoryMiddleware) Save(ctx context.Context, tw twins.Twin) (string, error) {
	span := createSpan(ctx, trm.tracer, saveTwinOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Save(ctx, tw)
}

func (trm twinRepositoryMiddleware) Update(ctx context.Context, tw twins.Twin) error {
	span := createSpan(ctx, trm.tracer, updateTwinOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, tw)
}

func (trm twinRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (twins.Twin, error) {
	span := createSpan(ctx, trm.tracer, retrieveTwinByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByID(ctx, id)
}

func (trm twinRepositoryMiddleware) RetrieveAll(ctx context.Context, owner string, offset, limit uint64, name string, metadata twins.Metadata) (twins.Page, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllTwinsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveAll(ctx, owner, offset, limit, name, metadata)
}

func (trm twinRepositoryMiddleware) RetrieveByAttribute(ctx context.Context, channel, subtopic string) ([]string, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllTwinsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByAttribute(ctx, channel, subtopic)
}

func (trm twinRepositoryMiddleware) Remove(ctx context.Context, id string) error {
	span := createSpan(ctx, trm.tracer, removeTwinOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Remove(ctx, id)
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
