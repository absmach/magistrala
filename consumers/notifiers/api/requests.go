// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/absmach/magistrala/internal/apiutil"

type createSubReq struct {
	token   string
	Topic   string `json:"topic,omitempty"`
	Contact string `json:"contact,omitempty"`
}

func (req createSubReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.Topic == "" {
		return apiutil.ErrInvalidTopic
	}
	if req.Contact == "" {
		return apiutil.ErrInvalidContact
	}
	return nil
}

type subReq struct {
	token string
	id    string
}

func (req subReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listSubsReq struct {
	token   string
	topic   string
	contact string
	offset  uint
	limit   uint
}

func (req listSubsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}
