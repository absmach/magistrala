// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jaeger

import (
	"context"
	"errors"

	jaegerp "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

var (
	errNoURL     = errors.New("URL is empty")
	errNoSvcName = errors.New("Service Name is empty")
)

// NewProvider initializes Jaeger TraceProvider.
func NewProvider(svcName, url string) (*tracesdk.TracerProvider, error) {
	if url == "" {
		return nil, errNoURL
	}

	if svcName == "" {
		return nil, errNoSvcName
	}

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{semconv.ServiceNameKey.String(svcName)}

	hostAttr, err := resource.New(context.TODO(), resource.WithHost(), resource.WithOSDescription(), resource.WithContainer())
	if err != nil {
		return nil, err
	}
	attributes = append(attributes, hostAttr.Attributes()...)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attributes...,
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(jaegerp.Jaeger{})

	return tp, nil
}
