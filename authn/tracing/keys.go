// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/authn"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOp     = "save"
	retrieveOp = "retrieve_by_id"
	revokeOp   = "remove"
)

var _ authn.KeyRepository = (*keyRepositoryMiddleware)(nil)

// keyRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
type keyRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   authn.KeyRepository
}

// New tracks request and their latency, and adds spans
// to context.
func New(repo authn.KeyRepository, tracer opentracing.Tracer) authn.KeyRepository {
	return keyRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (krm keyRepositoryMiddleware) Save(ctx context.Context, key authn.Key) (string, error) {
	span := createSpan(ctx, krm.tracer, saveOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return krm.repo.Save(ctx, key)
}

func (krm keyRepositoryMiddleware) Retrieve(ctx context.Context, owner, id string) (authn.Key, error) {
	span := createSpan(ctx, krm.tracer, retrieveOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return krm.repo.Retrieve(ctx, owner, id)
}

func (krm keyRepositoryMiddleware) Remove(ctx context.Context, owner, id string) error {
	span := createSpan(ctx, krm.tracer, revokeOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return krm.repo.Remove(ctx, owner, id)
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
