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

// Based on https://github.com/go-kit/kit/blob/3796a6b25f5c6c545454d3ed7187c4ced258083d/tracing/opencensus/endpoint.go

package otelkit // import "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd/lb"
)

const (
	tracerName = "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"

	// defaultSpanName is the default endpoint span name to use.
	defaultSpanName = "gokit/endpoint"
)

// EndpointMiddleware returns an Endpoint middleware, tracing a Go kit endpoint.
// This endpoint middleware should be used in combination with a Go kit Transport
// tracing middleware, generic OpenTelemetry transport middleware or custom before
// and after transport functions.
func EndpointMiddleware(options ...Option) endpoint.Middleware {
	cfg := &config{}

	for _, o := range options {
		o.apply(cfg)
	}

	if cfg.TracerProvider == nil {
		cfg.TracerProvider = otel.GetTracerProvider()
	}

	tracer := cfg.TracerProvider.Tracer(
		tracerName,
		trace.WithInstrumentationVersion(SemVersion()),
	)

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			operation := cfg.Operation
			if cfg.GetOperation != nil {
				if newOperation := cfg.GetOperation(ctx, operation); newOperation != "" {
					operation = newOperation
				}
			}

			spanName := operation
			if spanName == "" {
				spanName = defaultSpanName
			}

			opts := []trace.SpanStartOption{
				trace.WithAttributes(cfg.Attributes...),
				trace.WithSpanKind(trace.SpanKindServer),
			}

			if cfg.GetAttributes != nil {
				opts = append(opts, trace.WithAttributes(cfg.GetAttributes(ctx)...))
			}

			ctx, span := tracer.Start(ctx, spanName, opts...)
			defer span.End()

			defer func() {
				if err != nil {
					if lberr, ok := err.(lb.RetryError); ok {
						// Handle errors originating from lb.Retry.
						for idx, rawErr := range lberr.RawErrors {
							span.RecordError(rawErr, trace.WithAttributes(
								attribute.Int("gokit.lb.retry.count", idx+1),
							))
						}

						span.RecordError(lberr.Final)
						span.SetStatus(codes.Error, lberr.Error())

						return
					}

					// generic error
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())

					return
				}

				// Test for business error. Business errors are often
				// successful requests carrying a business failure that
				// the client can act upon and therefore do not count
				// as failed requests.
				if res, ok := response.(endpoint.Failer); ok && res.Failed() != nil {
					span.RecordError(res.Failed())

					if !cfg.IgnoreBusinessError {
						span.SetStatus(codes.Error, res.Failed().Error())
					}

					return
				}
				// no errors identified
			}()

			response, err = next(ctx, request)

			return
		}
	}
}
