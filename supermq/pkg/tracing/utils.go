// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	separator   = "-"
	emptyString = ""
	formater    = "%032s"
)

func StartSpan(ctx context.Context, tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	reqID := middleware.GetReqID(ctx)
	if reqID != "" {
		cleanID := strings.ReplaceAll(reqID, separator, emptyString)
		final := fmt.Sprintf(formater, cleanID)
		if traceID, err := trace.TraceIDFromHex(final); err == nil {
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     trace.SpanID{},
				TraceFlags: trace.FlagsSampled,
			})
			ctx = trace.ContextWithSpanContext(ctx, spanCtx)
		}
	}

	opts = append(opts, trace.WithAttributes(attribute.String("request_id", reqID)))
	return tracer.Start(ctx, name, opts...)
}
