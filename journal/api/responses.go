// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/journal"
)

var (
	_ magistrala.Response = (*pageRes)(nil)
	_ magistrala.Response = (*clientTelemetryRes)(nil)
)

type pageRes struct {
	journal.JournalsPage `json:",inline"`
}

func (res pageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res pageRes) Code() int {
	return http.StatusOK
}

func (res pageRes) Empty() bool {
	return false
}

type clientTelemetryRes struct {
	journal.ClientTelemetry `json:",inline"`
}

func (res clientTelemetryRes) Headers() map[string]string {
	return map[string]string{}
}

func (res clientTelemetryRes) Code() int {
	return http.StatusOK
}

func (res clientTelemetryRes) Empty() bool {
	return false
}
