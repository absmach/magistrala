// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/messaging"
)

type publishReq struct {
	msg   messaging.Message
	token string
}
