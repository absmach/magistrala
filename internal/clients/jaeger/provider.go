// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package jaeger

import (
	"context"
	"errors"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var (
	errNoURL                     = errors.New("URL is empty")
	errNoSvcName                 = errors.New("service Name is empty")
	errUnsupportedTraceURLScheme = errors.New("unsupported tracing url scheme")
)

// NewProvider initializes Jaeger TraceProvider.
func NewProvider(ctx context.Context, svcName, jaegerUrl, instanceID string, fraction float64) (*trace.TracerProvider, error) {
	if jaegerUrl == "" {
		return nil, errNoURL
	}

	if svcName == "" {
		return nil, errNoSvcName
	}

	url, err := url.Parse(jaegerUrl)
	if err != nil {
		return nil, err
	}

	var exporter *otlptrace.Exporter
	switch url.Scheme {
	case "http":
		exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(url.Host), otlptracehttp.WithURLPath(url.Path), otlptracehttp.WithInsecure())
		if err != nil {
			return nil, err
		}
	case "https":
		exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(url.Host), otlptracehttp.WithURLPath(url.Path))
		if err != nil {
			return nil, err
		}
	default:
		return nil, errUnsupportedTraceURLScheme
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(svcName),
		attribute.String("host.id", instanceID),
	}

	hostAttr, err := resource.New(ctx, resource.WithHost(), resource.WithOSDescription(), resource.WithContainer())
	if err != nil {
		return nil, err
	}
	attributes = append(attributes, hostAttr.Attributes()...)

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(fraction)),
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attributes...,
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}
