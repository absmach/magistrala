// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jaeger

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/opentracing/opentracing-go"
	jconfig "github.com/uber/jaeger-client-go/config"
)

var (
	errNoUrl     = errors.New("URL is empty")
	errNoSvcName = errors.New("Service Name is empty")
)

// NewTracer initializes Jaeger
func NewTracer(svcName, url string) (opentracing.Tracer, io.Closer, error) {
	if url == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil), errNoUrl
	}

	if svcName == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil), errNoSvcName
	}

	return jconfig.Configuration{
		ServiceName: svcName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LocalAgentHostPort: url,
			LogSpans:           true,
		},
	}.NewTracer()
}
