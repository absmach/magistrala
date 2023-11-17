// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/twins"
	"go.opentelemetry.io/otel/trace"
)

const (
	saveStateOp         = "save_state"
	updateStateOp       = "update_state"
	countStatesOp       = "count_states"
	retrieveAllStatesOp = "retrieve_all_states"
)

var _ twins.StateRepository = (*stateRepositoryMiddleware)(nil)

type stateRepositoryMiddleware struct {
	tracer trace.Tracer
	repo   twins.StateRepository
}

// StateRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func StateRepositoryMiddleware(tracer trace.Tracer, repo twins.StateRepository) twins.StateRepository {
	return stateRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm stateRepositoryMiddleware) Save(ctx context.Context, st twins.State) error {
	ctx, span := createSpan(ctx, trm.tracer, saveStateOp)
	defer span.End()

	return trm.repo.Save(ctx, st)
}

func (trm stateRepositoryMiddleware) Update(ctx context.Context, st twins.State) error {
	ctx, span := createSpan(ctx, trm.tracer, updateStateOp)
	defer span.End()

	return trm.repo.Update(ctx, st)
}

func (trm stateRepositoryMiddleware) Count(ctx context.Context, tw twins.Twin) (int64, error) {
	ctx, span := createSpan(ctx, trm.tracer, countStatesOp)
	defer span.End()

	return trm.repo.Count(ctx, tw)
}

func (trm stateRepositoryMiddleware) RetrieveAll(ctx context.Context, offset, limit uint64, twinID string) (twins.StatesPage, error) {
	ctx, span := createSpan(ctx, trm.tracer, retrieveAllStatesOp)
	defer span.End()

	return trm.repo.RetrieveAll(ctx, offset, limit, twinID)
}

func (trm stateRepositoryMiddleware) RetrieveLast(ctx context.Context, twinID string) (twins.State, error) {
	ctx, span := createSpan(ctx, trm.tracer, retrieveAllStatesOp)
	defer span.End()

	return trm.repo.RetrieveLast(ctx, twinID)
}
