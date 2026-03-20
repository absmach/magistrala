// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/absmach/supermq/consumers/notifiers"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	saveOp        = "save_op"
	retrieveOp    = "retrieve_op"
	retrieveAllOp = "retrieve_all_op"
	removeOp      = "remove_op"
)

var _ notifiers.SubscriptionsRepository = (*subRepositoryMiddleware)(nil)

type subRepositoryMiddleware struct {
	tracer trace.Tracer
	repo   notifiers.SubscriptionsRepository
}

// New instantiates a new Subscriptions repository that
// tracks request and their latency, and adds spans to context.
func New(tracer trace.Tracer, repo notifiers.SubscriptionsRepository) notifiers.SubscriptionsRepository {
	return subRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

// Save traces the "Save" operation of the wrapped Subscriptions repository.
func (urm subRepositoryMiddleware) Save(ctx context.Context, sub notifiers.Subscription) (string, error) {
	ctx, span := urm.tracer.Start(ctx, saveOp, trace.WithAttributes(
		attribute.String("id", sub.ID),
		attribute.String("contact", sub.Contact),
		attribute.String("topic", sub.Topic),
	))
	defer span.End()

	return urm.repo.Save(ctx, sub)
}

// Retrieve traces the "Retrieve" operation of the wrapped Subscriptions repository.
func (urm subRepositoryMiddleware) Retrieve(ctx context.Context, id string) (notifiers.Subscription, error) {
	ctx, span := urm.tracer.Start(ctx, retrieveOp, trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return urm.repo.Retrieve(ctx, id)
}

// RetrieveAll traces the "RetrieveAll" operation of the wrapped Subscriptions repository.
func (urm subRepositoryMiddleware) RetrieveAll(ctx context.Context, pm notifiers.PageMetadata) (notifiers.Page, error) {
	ctx, span := urm.tracer.Start(ctx, retrieveAllOp)
	defer span.End()

	return urm.repo.RetrieveAll(ctx, pm)
}

// Remove traces the "Remove" operation of the wrapped Subscriptions repository.
func (urm subRepositoryMiddleware) Remove(ctx context.Context, id string) error {
	ctx, span := urm.tracer.Start(ctx, removeOp, trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return urm.repo.Remove(ctx, id)
}
