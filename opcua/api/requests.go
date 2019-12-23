// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/mainflux/mainflux/opcua"

type browseReq struct {
	ServerURI  string
	Namespace  string
	Identifier string
}

func (req *browseReq) validate() error {
	if req.ServerURI == "" || req.Namespace == "" || req.Identifier == "" {
		return opcua.ErrMalformedEntity
	}

	return nil
}
