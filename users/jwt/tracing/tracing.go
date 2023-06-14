// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/users/jwt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ jwt.Repository = (*tokenRepoMiddlware)(nil)

type tokenRepoMiddlware struct {
	repo   jwt.Repository
	tracer trace.Tracer
}

// New returns a new jwt service with tracing capabilities.
func New(repo jwt.Repository, tracer trace.Tracer) jwt.Repository {
	return &tokenRepoMiddlware{
		repo:   repo,
		tracer: tracer,
	}
}

func (trm tokenRepoMiddlware) Issue(ctx context.Context, claim jwt.Claims) (jwt.Token, error) {
	ctx, span := trm.tracer.Start(ctx, "issue_token", trace.WithAttributes(
		attribute.String("client_id", claim.ClientID),
		attribute.String("email", claim.Email),
		attribute.String("type", claim.Type),
	))
	defer span.End()

	return trm.repo.Issue(ctx, claim)
}

func (trm tokenRepoMiddlware) Parse(ctx context.Context, accessToken string) (jwt.Claims, error) {
	ctx, span := trm.tracer.Start(ctx, "parse_token", trace.WithAttributes(
		attribute.String("accesstoken", accessToken),
	))
	defer span.End()

	return trm.repo.Parse(ctx, accessToken)
}
