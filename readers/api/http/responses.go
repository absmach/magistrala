// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/readers"
)

var _ magistrala.Response = (*pageRes)(nil)

type pageRes struct {
	readers.PageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages"`
}

func (res pageRes) MarshalJSON() ([]byte, error) {
	pm := res.PageMetadata
	pm.DeviceScope = nil

	type pageResponse pageRes
	return json.Marshal(pageResponse{
		PageMetadata: pm,
		Total:        res.Total,
		Messages:     res.Messages,
	})
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
