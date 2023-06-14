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

// Package otelkit instruments the github.com/go-kit/kit package.
//
// Compared to other instrumentation libraries provided by go-kit itself,
// this package only provides instrumentation for the endpoint layer.
// For instrumenting the transport layer,
// look at the instrumentation libraries provided by go.opentelemetry.io/contrib.
// Learn more about go-kit's layers at https://gokit.io/faq/#architecture-and-design.
package otelkit // import "go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
