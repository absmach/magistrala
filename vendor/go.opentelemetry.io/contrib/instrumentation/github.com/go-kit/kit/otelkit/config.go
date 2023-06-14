// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Based on https://github.com/go-kit/kit/blob/3796a6b25f5c6c545454d3ed7187c4ced258083d/tracing/opencensus/endpoint_options.go

package otelkit // import "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// config holds the options for tracing an endpoint.
type config struct {
	// TracerProvider provides access to instrumentation Tracers.
	TracerProvider trace.TracerProvider

	// IgnoreBusinessError if set to true will not treat a business error
	// identified through the endpoint.Failer interface as a span error.
	IgnoreBusinessError bool

	// Operation identifies the current operation and serves as a span name.
	Operation string

	// GetOperation is an optional function that can set the span name based on the existing operation
	// for the endpoint and information in the context.
	//
	// If the function is nil, or the returned operation is empty, the existing operation for the endpoint is used.
	GetOperation func(ctx context.Context, operation string) string

	// Attributes holds the default attributes for each span created by this middleware.
	Attributes []attribute.KeyValue

	// GetAttributes is an optional function that can extract trace attributes
	// from the context and add them to the span.
	GetAttributes func(ctx context.Context) []attribute.KeyValue
}

// Option configures an EndpointMiddleware.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(o *config) {
		if provider != nil {
			o.TracerProvider = provider
		}
	})
}

// WithIgnoreBusinessError if set to true will not treat a business error
// identified through the endpoint.Failer interface as a span error.
func WithIgnoreBusinessError(val bool) Option {
	return optionFunc(func(o *config) {
		o.IgnoreBusinessError = val
	})
}

// WithOperation sets an operation name for an endpoint.
// Use this when you register a middleware for each endpoint.
func WithOperation(operation string) Option {
	return optionFunc(func(o *config) {
		o.Operation = operation
	})
}

// WithOperationGetter sets an operation name getter function in config.
func WithOperationGetter(fn func(ctx context.Context, name string) string) Option {
	return optionFunc(func(o *config) {
		o.GetOperation = fn
	})
}

// WithAttributes sets the default attributes for the spans created by the Endpoint tracer.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return optionFunc(func(o *config) {
		o.Attributes = attrs
	})
}

// WithAttributeGetter extracts additional attributes from the context.
func WithAttributeGetter(fn func(ctx context.Context) []attribute.KeyValue) Option {
	return optionFunc(func(o *config) {
		o.GetAttributes = fn
	})
}
