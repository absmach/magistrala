// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/mainflux/mainflux/readers"

type apiReq interface {
	validate() error
}

type listMessagesReq struct {
	chanID   string
	pageMeta readers.PageMetadata
}

func (req listMessagesReq) validate() error {
	if req.pageMeta.Limit < 1 {
		return errInvalidRequest
	}

	return nil
}
