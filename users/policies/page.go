// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies

import "github.com/mainflux/mainflux/internal/apiutil"

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

// Page contains page metadata that helps navigation.
type Page struct {
	Total   uint64 `json:"total"`
	Offset  uint64 `json:"offset"`
	Limit   uint64 `json:"limit"`
	OwnerID string
	Subject string
	Object  string
	Action  string
	Tag     string
}

// Validate check page actions.
func (p Page) Validate() error {
	if p.Action != "" {
		if ok := ValidateAction(p.Action); !ok {
			return apiutil.ErrMalformedPolicyAct
		}
	}
	return nil
}
