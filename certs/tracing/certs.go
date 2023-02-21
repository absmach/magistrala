// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/certs"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveCertsOp          = "save_certs"
	updateCertsOp        = "update_certs"
	retrieveCertsOp      = "retrieve_certs"
	removeCertsOp        = "retrieve_certs"
	retrieveThingCertsOp = "retrieve_thing_certs"
	removeThingCertsOp   = "retrieve_thing_certs"
)

var (
	_ certs.Repository = (*certsRepositoryMiddleware)(nil)
)

type certsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   certs.Repository
}

// ChannelRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func CertsRepositoryMiddleware(tracer opentracing.Tracer, repo certs.Repository) certs.Repository {
	return certsRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (crm certsRepositoryMiddleware) Save(ctx context.Context, cert certs.Cert) error {
	span := createSpan(ctx, crm.tracer, saveCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, cert)
}

func (crm certsRepositoryMiddleware) Update(ctx context.Context, ownerID string, cert certs.Cert) error {
	span := createSpan(ctx, crm.tracer, updateCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, ownerID, cert)
}

func (crm certsRepositoryMiddleware) Retrieve(ctx context.Context, ownerID, certID, name, thingID, serial string, status certs.Status, offset uint64, limit int64) (certs.Page, error) {
	span := createSpan(ctx, crm.tracer, retrieveCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Retrieve(ctx, ownerID, certID, name, thingID, serial, status, offset, limit)
}

func (crm certsRepositoryMiddleware) RetrieveCount(ctx context.Context, ownerID, certID, name, thingID, serial string, status certs.Status) (uint64, error) {
	span := createSpan(ctx, crm.tracer, removeThingCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveCount(ctx, ownerID, certID, name, thingID, serial, status)
}

func (crm certsRepositoryMiddleware) Remove(ctx context.Context, ownerID, certID string) error {
	span := createSpan(ctx, crm.tracer, removeCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, ownerID, certID)
}

func (crm certsRepositoryMiddleware) RetrieveThingCerts(ctx context.Context, thingID string) (certs.Page, error) {
	span := createSpan(ctx, crm.tracer, retrieveThingCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveThingCerts(ctx, thingID)
}

func (crm certsRepositoryMiddleware) RemoveThingCerts(ctx context.Context, thingID string) error {
	span := createSpan(ctx, crm.tracer, removeThingCertsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RemoveThingCerts(ctx, thingID)
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
