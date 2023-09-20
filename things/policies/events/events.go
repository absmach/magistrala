// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"strings"

	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/things/policies"
)

const (
	policyPrefix = "policies."
	authorize    = policyPrefix + "authorize"
	policyAdd    = policyPrefix + "add"
	policyUpdate = policyPrefix + "update"
	policyList   = policyPrefix + "list"
	policyDelete = policyPrefix + "delete"
)

var (
	_ events.Event = (*policyEvent)(nil)
	_ events.Event = (*authorizeEvent)(nil)
	_ events.Event = (*listPoliciesEvent)(nil)
)

type policyEvent struct {
	policies.Policy
	operation string
}

func (pe policyEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": pe.operation,
	}
	if pe.OwnerID != "" {
		val["owner_id"] = pe.OwnerID
	}
	if pe.Subject != "" {
		val["subject"] = pe.Subject
	}
	if pe.Object != "" {
		val["object"] = pe.Object
	}
	if len(pe.Actions) > 0 {
		actions := fmt.Sprintf("[%s]", strings.Join(pe.Actions, ","))
		val["actions"] = actions
	}
	if !pe.CreatedAt.IsZero() {
		val["created_at"] = pe.CreatedAt
	}
	if !pe.UpdatedAt.IsZero() {
		val["updated_at"] = pe.UpdatedAt
	}
	if pe.UpdatedBy != "" {
		val["updated_by"] = pe.UpdatedBy
	}
	return val, nil
}

type authorizeEvent struct {
	policies.AccessRequest
}

func (ae authorizeEvent) Encode() (map[string]interface{}, error) {
	// We don't want to send the key over the stream, so we don't send the subject.
	val := map[string]interface{}{
		"operation":   authorize,
		"entity_type": ae.Entity,
	}

	if ae.Object != "" {
		val["object"] = ae.Object
	}
	if ae.Action != "" {
		val["actions"] = ae.Action
	}
	return val, nil
}

type listPoliciesEvent struct {
	policies.Page
}

func (ae listPoliciesEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": policyList,
		"total":     ae.Total,
		"limit":     ae.Limit,
		"offset":    ae.Offset,
	}
	if ae.OwnerID != "" {
		val["owner_id"] = ae.OwnerID
	}
	if ae.Subject != "" {
		val["subject"] = ae.Subject
	}
	if ae.Object != "" {
		val["object"] = ae.Object
	}
	if ae.Action != "" {
		val["action"] = ae.Action
	}
	return val, nil
}
