//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"net/http"

	"github.com/mainflux/mainflux"
)

var _ mainflux.Response = (*listMessagesRes)(nil)

type message struct {
	Channel     string   `json:"channel,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Name        string   `json:"name,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Value       *float64 `json:"value,omitempty"`
	StringValue *string  `json:"stringValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
	DataValue   *string  `json:"dataValue,omitempty"`
	ValueSum    *float64 `json:"valueSum,omitempty"`
	Time        float64  `json:"time,omitempty"`
	UpdateTime  float64  `json:"updateTime,omitempty"`
	Link        string   `json:"link,omitempty"`
}

type listMessagesRes struct {
	Messages []message `json:"messages"`
}

func (res listMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listMessagesRes) Code() int {
	return http.StatusOK
}

func (res listMessagesRes) Empty() bool {
	return false
}
