// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/twins"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveStateOp         = "save_state"
	updateStateOp       = "update_state"
	countStatesOp       = "count_states"
	retrieveAllStatesOp = "retrieve_all_states"
)

var _ twins.StateRepository = (*stateRepositoryMiddleware)(nil)

type stateRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   twins.StateRepository
}

// StateRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func StateRepositoryMiddleware(tracer opentracing.Tracer, repo twins.StateRepository) twins.StateRepository {
	return stateRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm stateRepositoryMiddleware) Save(ctx context.Context, st twins.State) error {
	span := createSpan(ctx, trm.tracer, saveStateOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Save(ctx, st)
}

func (trm stateRepositoryMiddleware) Update(ctx context.Context, st twins.State) error {
	span := createSpan(ctx, trm.tracer, updateStateOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, st)
}

func (trm stateRepositoryMiddleware) Count(ctx context.Context, tw twins.Twin) (int64, error) {
	span := createSpan(ctx, trm.tracer, countStatesOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Count(ctx, tw)
}

func (trm stateRepositoryMiddleware) RetrieveAll(ctx context.Context, offset, limit uint64, twinID string) (twins.StatesPage, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllStatesOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveAll(ctx, offset, limit, twinID)
}

func (trm stateRepositoryMiddleware) RetrieveLast(ctx context.Context, twinID string) (twins.State, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllStatesOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveLast(ctx, twinID)
}
