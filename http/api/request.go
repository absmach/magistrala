// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/messaging"
)

type publishReq struct {
	msg   *messaging.Message
	token string
}

func (req publishReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerKey
	}
	if len(req.msg.Payload) == 0 {
		return apiutil.ErrEmptyMessage
	}

	return nil
}
