//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"github.com/mainflux/mainflux"
)

type publishReq struct {
	msg   mainflux.RawMessage
	token string
}
