// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/auth"
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

func (tm *tracingMiddleware) Identify(ctx context.Context, token string) (auth.Key, error) {
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

func (tm *tracingMiddleware) AddPolicies(ctx context.Context, prs []auth.PolicyReq) error {
	ctx, span := tm.tracer.Start(ctx, "add_policies", trace.WithAttributes())
	defer span.End()

	return tm.svc.AddPolicies(ctx, prs)
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

func (tm *tracingMiddleware) DeletePolicies(ctx context.Context, prs []auth.PolicyReq) error {
	ctx, span := tm.tracer.Start(ctx, "delete_policies", trace.WithAttributes())
	defer span.End()

	return tm.svc.DeletePolicies(ctx, prs)
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

func (tm *tracingMiddleware) ListPermissions(ctx context.Context, pr auth.PolicyReq, filterPermissions []string) (auth.Permissions, error) {
	ctx, span := tm.tracer.Start(ctx, "list_permissions", trace.WithAttributes(
		attribute.String("subject", pr.Subject),
		attribute.String("subject_type", pr.SubjectType),
		attribute.String("subject_kind", pr.SubjectKind),
		attribute.String("subject_relation", pr.SubjectRelation),
		attribute.String("object", pr.Object),
		attribute.String("object_type", pr.ObjectType),
		attribute.String("relation", pr.Relation),
		attribute.String("permission", pr.Permission),
		attribute.StringSlice("filter_permissions", filterPermissions),
	))
	defer span.End()

	return tm.svc.ListPermissions(ctx, pr, filterPermissions)
}

func (tm *tracingMiddleware) CreateDomain(ctx context.Context, token string, d auth.Domain) (auth.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "create_domain", trace.WithAttributes(
		attribute.String("name", d.Name),
	))
	defer span.End()
	return tm.svc.CreateDomain(ctx, token, d)
}

func (tm *tracingMiddleware) RetrieveDomain(ctx context.Context, token, id string) (auth.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "view_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.RetrieveDomain(ctx, token, id)
}

func (tm *tracingMiddleware) RetrieveDomainPermissions(ctx context.Context, token, id string) (auth.Permissions, error) {
	ctx, span := tm.tracer.Start(ctx, "view_domain_permissions", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.RetrieveDomainPermissions(ctx, token, id)
}

func (tm *tracingMiddleware) UpdateDomain(ctx context.Context, token, id string, d auth.DomainReq) (auth.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "update_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.UpdateDomain(ctx, token, id, d)
}

func (tm *tracingMiddleware) ChangeDomainStatus(ctx context.Context, token, id string, d auth.DomainReq) (auth.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "change_domain_status", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.ChangeDomainStatus(ctx, token, id, d)
}

func (tm *tracingMiddleware) ListDomains(ctx context.Context, token string, p auth.Page) (auth.DomainsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_domains")
	defer span.End()
	return tm.svc.ListDomains(ctx, token, p)
}

func (tm *tracingMiddleware) AssignUsers(ctx context.Context, token, id string, userIds []string, relation string) error {
	ctx, span := tm.tracer.Start(ctx, "assign_users", trace.WithAttributes(
		attribute.String("id", id),
		attribute.StringSlice("user_ids", userIds),
		attribute.String("relation", relation),
	))
	defer span.End()
	return tm.svc.AssignUsers(ctx, token, id, userIds, relation)
}

func (tm *tracingMiddleware) UnassignUsers(ctx context.Context, token, id string, userIds []string, relation string) error {
	ctx, span := tm.tracer.Start(ctx, "unassign_users", trace.WithAttributes(
		attribute.String("id", id),
		attribute.StringSlice("user_ids", userIds),
		attribute.String("relation", relation),
	))
	defer span.End()
	return tm.svc.UnassignUsers(ctx, token, id, userIds, relation)
}

func (tm *tracingMiddleware) ListUserDomains(ctx context.Context, token, userID string, p auth.Page) (auth.DomainsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_user_domains", trace.WithAttributes(
		attribute.String("user_id", userID),
	))
	defer span.End()
	return tm.svc.ListUserDomains(ctx, token, userID, p)
}
