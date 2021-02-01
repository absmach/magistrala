// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	notifiers "github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errInvalidTopic   = errors.New("invalid Subscription topic")
	errInvalidContact = errors.New("invalid Subscription contact")
	errNotFound       = errors.New("invalid or empty Subscription id")
)

type createSubReq struct {
	token   string
	Topic   string `json:"topic,omitempty"`
	Contact string `json:"contact,omitempty"`
}

func (req createSubReq) validate() error {
	if req.token == "" {
		return notifiers.ErrUnauthorizedAccess
	}
	if req.Topic == "" {
		return errInvalidTopic
	}
	if req.Contact == "" {
		return errInvalidContact
	}
	return nil
}

type subReq struct {
	token string
	id    string
}

func (req subReq) validate() error {
	if req.token == "" {
		return notifiers.ErrUnauthorizedAccess
	}
	if req.id == "" {
		return errNotFound
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
		return notifiers.ErrUnauthorizedAccess
	}
	return nil
}
