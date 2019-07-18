//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOp         = "save_op"
	retrieveByIDOp = "retrieve_by_id"
)

var _ users.UserRepository = (*userRepositoryMiddleware)(nil)

type userRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   users.UserRepository
}

// UserRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func UserRepositoryMiddleware(repo users.UserRepository, tracer opentracing.Tracer) users.UserRepository {
	return userRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (urm userRepositoryMiddleware) Save(ctx context.Context, user users.User) error {
	span := createSpan(ctx, urm.tracer, saveOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Save(ctx, user)
}

func (urm userRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	span := createSpan(ctx, urm.tracer, retrieveByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByID(ctx, id)
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
