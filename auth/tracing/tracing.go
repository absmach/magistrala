// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux/auth"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ auth.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    auth.Service
}

// New returns a new group service with tracing capabilities.
func New(svc auth.Service, tracer trace.Tracer) auth.Service {
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

func (tm *tracingMiddleware) Identify(ctx context.Context, token string) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "identify")
	defer span.End()

	return tm.svc.Identify(ctx, token)
}

func (tm *tracingMiddleware) Authorize(ctx context.Context, pr auth.PolicyReq) error {
	ctx, span := tm.tracer.Start(ctx, "authorize", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.Authorize(ctx, pr)
}

func (tm *tracingMiddleware) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	ctx, span := tm.tracer.Start(ctx, "add_policy", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.AddPolicy(ctx, pr)
}

func (tm *tracingMiddleware) AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	ctx, span := tm.tracer.Start(ctx, "add_policies", trace.WithAttributes(
		attribute.String("object", object),
		attribute.StringSlice("subject_ids", subjectIDs),
		attribute.StringSlice("relations", relations),
	))
	defer span.End()

	return tm.svc.AddPolicies(ctx, token, object, subjectIDs, relations)
}

func (tm *tracingMiddleware) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	ctx, span := tm.tracer.Start(ctx, "delete_policy", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.DeletePolicy(ctx, pr)
}

func (tm *tracingMiddleware) DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	ctx, span := tm.tracer.Start(ctx, "delete_policies", trace.WithAttributes(
		attribute.String("object", object),
		attribute.StringSlice("subject_ids", subjectIDs),
		attribute.StringSlice("relations", relations),
	))
	defer span.End()

	return tm.svc.DeletePolicies(ctx, token, object, subjectIDs, relations)
}

func (tm *tracingMiddleware) ListObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (auth.PolicyPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_objects", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.ListObjects(ctx, pr, nextPageToken, limit)
}

func (tm *tracingMiddleware) ListAllObjects(ctx context.Context, pr auth.PolicyReq) (auth.PolicyPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_all_objects", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.ListAllObjects(ctx, pr)
}

func (tm *tracingMiddleware) CountObjects(ctx context.Context, pr auth.PolicyReq) (int, error) {
	ctx, span := tm.tracer.Start(ctx, "count_objects", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.CountObjects(ctx, pr)
}

func (tm *tracingMiddleware) ListSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (auth.PolicyPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_subjects", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.ListSubjects(ctx, pr, nextPageToken, limit)
}

func (tm *tracingMiddleware) ListAllSubjects(ctx context.Context, pr auth.PolicyReq) (auth.PolicyPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_all_subjects", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.ListAllSubjects(ctx, pr)
}

func (tm *tracingMiddleware) CountSubjects(ctx context.Context, pr auth.PolicyReq) (int, error) {
	ctx, span := tm.tracer.Start(ctx, "count_subjects", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
	))
	defer span.End()

	return tm.svc.CountSubjects(ctx, pr)
}
