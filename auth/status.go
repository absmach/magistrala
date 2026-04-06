// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

type Status uint8

const (
	ActiveStatus Status = iota
	RevokedStatus
	ExpiredStatus
	AllStatus
)

const (
	Active  = "active"
	Revoked = "revoked"
	Expired = "expired"
	All     = "all"
	Unknown = "unknown"
)

func (s Status) String() string {
	switch s {
	case ActiveStatus:
		return Active
	case RevokedStatus:
		return Revoked
	case ExpiredStatus:
		return Expired
	case AllStatus:
		return All
	default:
		return Unknown
	}
}

// ToStatus converts string value to a valid Client status.
func ToStatus(status string) (Status, error) {
	switch status {
	case "", Active:
		return ActiveStatus, nil
	case Revoked:
		return RevokedStatus, nil
	case All:
		return AllStatus, nil
	case Expired:
		return ExpiredStatus, nil
	}
	return Status(0), svcerr.ErrInvalidStatus
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (p PAT) MarshalJSON() ([]byte, error) {
	type Alias PAT
	return json.Marshal(&struct {
		Alias
		Status string `json:"status,omitempty"`
	}{
		Alias:  (Alias)(p),
		Status: p.Status.String(),
	})
}

func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}
