// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/policies"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ auth.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    auth.Service
}

// NewTracing returns a new auth service with tracing capabilities.
func NewTracing(svc auth.Service, tracer trace.Tracer) auth.Service {
	return &tracingMiddleware{tracer, svc}
}

func (tm *tracingMiddleware) Issue(ctx context.Context, token string, key auth.Key) (auth.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "issue", trace.WithAttributes(
		attribute.String("type", fmt.Sprintf("%d", key.Type)),
		attribute.String("subject", key.Subject),
	))
	defer span.End()

	return tm.svc.Issue(ctx, token, key)
}

func (tm *tracingMiddleware) Revoke(ctx context.Context, token, id string) error {
	ctx, span := tm.tracer.Start(ctx, "revoke", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.Revoke(ctx, token, id)
}

func (tm *tracingMiddleware) RetrieveKey(ctx context.Context, token, id string) (auth.Key, error) {
	ctx, span := tm.tracer.Start(ctx, "retrieve_key", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.RetrieveKey(ctx, token, id)
}

func (tm *tracingMiddleware) Identify(ctx context.Context, token string) (auth.Key, error) {
	ctx, span := tm.tracer.Start(ctx, "identify")
	defer span.End()

	return tm.svc.Identify(ctx, token)
}

func (tm *tracingMiddleware) RetrieveJWKS() []auth.PublicKeyInfo {
	return tm.svc.RetrieveJWKS()
}

func (tm *tracingMiddleware) Authorize(ctx context.Context, pr policies.Policy, patAuthz *auth.PATAuthz) error {
	attributes := []attribute.KeyValue{
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	}

	if patAuthz != nil {
		attributes = append(attributes,
			attribute.String("pat_id", patAuthz.PatID),
			attribute.String("pat_user_id", patAuthz.UserID),
			attribute.String("pat_entity_type", patAuthz.EntityType.String()),
			attribute.String("pat_entity_id", patAuthz.EntityID),
			attribute.String("pat_operation", patAuthz.Operation),
			attribute.String("pat_domain", patAuthz.Domain),
		)
	}

	ctx, span := tm.tracer.Start(ctx, "authorize", trace.WithAttributes(attributes...))
	defer span.End()

	return tm.svc.Authorize(ctx, pr, patAuthz)
}

func (tm *tracingMiddleware) CreatePAT(ctx context.Context, token, name, description string, duration time.Duration) (auth.PAT, error) {
	ctx, span := tm.tracer.Start(ctx, "create_pat", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("description", description),
		attribute.String("duration", duration.String()),
	))
	defer span.End()
	return tm.svc.CreatePAT(ctx, token, name, description, duration)
}

func (tm *tracingMiddleware) UpdatePATName(ctx context.Context, token, patID, name string) (auth.PAT, error) {
	ctx, span := tm.tracer.Start(ctx, "update_pat_name", trace.WithAttributes(
		attribute.String("pat_id", patID),
		attribute.String("name", name),
	))
	defer span.End()
	return tm.svc.UpdatePATName(ctx, token, patID, name)
}

func (tm *tracingMiddleware) UpdatePATDescription(ctx context.Context, token, patID, description string) (auth.PAT, error) {
	ctx, span := tm.tracer.Start(ctx, "update_pat_description", trace.WithAttributes(
		attribute.String("pat_id", patID),
		attribute.String("description", description),
	))
	defer span.End()
	return tm.svc.UpdatePATDescription(ctx, token, patID, description)
}

func (tm *tracingMiddleware) RetrievePAT(ctx context.Context, token, patID string) (auth.PAT, error) {
	ctx, span := tm.tracer.Start(ctx, "retrieve_pat", trace.WithAttributes(
		attribute.String("pat_id", patID),
	))
	defer span.End()
	return tm.svc.RetrievePAT(ctx, token, patID)
}

func (tm *tracingMiddleware) ListPATS(ctx context.Context, token string, pm auth.PATSPageMeta) (auth.PATSPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_pat", trace.WithAttributes(
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.Int64("offset", int64(pm.Offset)),
	))
	defer span.End()
	return tm.svc.ListPATS(ctx, token, pm)
}

func (tm *tracingMiddleware) ListScopes(ctx context.Context, token string, pm auth.ScopesPageMeta) (auth.ScopesPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_scopes", trace.WithAttributes(
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.Int64("offset", int64(pm.Offset)),
	))
	defer span.End()
	return tm.svc.ListScopes(ctx, token, pm)
}

func (tm *tracingMiddleware) DeletePAT(ctx context.Context, token, patID string) error {
	ctx, span := tm.tracer.Start(ctx, "delete_pat", trace.WithAttributes(
		attribute.String("pat_id", patID),
	))
	defer span.End()
	return tm.svc.DeletePAT(ctx, token, patID)
}

func (tm *tracingMiddleware) ResetPATSecret(ctx context.Context, token, patID string, duration time.Duration) (auth.PAT, error) {
	ctx, span := tm.tracer.Start(ctx, "reset_pat_secret", trace.WithAttributes(
		attribute.String("pat_id", patID),
		attribute.String("duration", duration.String()),
	))
	defer span.End()
	return tm.svc.ResetPATSecret(ctx, token, patID, duration)
}

func (tm *tracingMiddleware) RevokePATSecret(ctx context.Context, token, patID string) error {
	ctx, span := tm.tracer.Start(ctx, "revoke_pat_secret", trace.WithAttributes(
		attribute.String("pat_id", patID),
	))
	defer span.End()
	return tm.svc.RevokePATSecret(ctx, token, patID)
}

func (tm *tracingMiddleware) RemoveAllPAT(ctx context.Context, token string) error {
	ctx, span := tm.tracer.Start(ctx, "clear_all_pat")
	defer span.End()
	return tm.svc.RemoveAllPAT(ctx, token)
}

func (tm *tracingMiddleware) AddScope(ctx context.Context, token, patID string, scopes []auth.Scope) error {
	var attributes []attribute.KeyValue
	for _, s := range scopes {
		attributes = append(attributes, attribute.String("entity_type", s.EntityType.String()))
		attributes = append(attributes, attribute.String("domain_id", s.DomainID))
		attributes = append(attributes, attribute.String("operation", s.Operation))
		attributes = append(attributes, attribute.String("entity_id", s.EntityID))
	}

	attributes = append(attributes, attribute.String("pat_id", patID))

	ctx, span := tm.tracer.Start(ctx, "add_pat_scope", trace.WithAttributes(attributes...))
	defer span.End()
	return tm.svc.AddScope(ctx, token, patID, scopes)
}

func (tm *tracingMiddleware) RemoveScope(ctx context.Context, token, patID string, scopesID ...string) error {
	var attributes []attribute.KeyValue
	for _, s := range scopesID {
		attributes = append(attributes, attribute.String("scope_id", s))
	}

	attributes = append(attributes, attribute.String("pat_id", patID))

	ctx, span := tm.tracer.Start(ctx, "remove_pat_scope", trace.WithAttributes(attributes...))
	defer span.End()
	return tm.svc.RemoveScope(ctx, token, patID, scopesID...)
}

func (tm *tracingMiddleware) RemovePATAllScope(ctx context.Context, token, patID string) error {
	ctx, span := tm.tracer.Start(ctx, "clear_pat_all_scope", trace.WithAttributes(
		attribute.String("pat_id", patID),
	))
	defer span.End()
	return tm.svc.RemovePATAllScope(ctx, token, patID)
}

func (tm *tracingMiddleware) IdentifyPAT(ctx context.Context, paToken string) (auth.PAT, error) {
	ctx, span := tm.tracer.Start(ctx, "identity_pat")
	defer span.End()
	return tm.svc.IdentifyPAT(ctx, paToken)
}

func (tm *tracingMiddleware) AuthorizePAT(ctx context.Context, userID, patID string, entityType auth.EntityType, domainID string, operation string, entityID string) error {
	ctx, span := tm.tracer.Start(ctx, "authorize_pat", trace.WithAttributes(
		attribute.String("pat_id", patID),
		attribute.String("entity_type", entityType.String()),
		attribute.String("domain_id", domainID),
		attribute.String("operation", operation),
		attribute.String("entities", entityID),
	))
	defer span.End()
	return tm.svc.AuthorizePAT(ctx, userID, patID, entityType, domainID, operation, entityID)
}
