// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/readers"
)

type readMessagesReq struct {
	chanID   string
	token    string
	domain   string
	pageMeta readers.PageMetadata
}

func (req readMessagesReq) validate() error {
	if req.chanID == "" {
		return apiutil.ErrMissingID
	}
	if req.domain == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
