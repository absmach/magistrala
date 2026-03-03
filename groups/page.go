// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import "strings"

type Operator uint8

const (
	OrOp Operator = iota
	AndOp
)

type TagsQuery struct {
	Elements []string
	Operator Operator
}

func ToTagsQuery(s string) TagsQuery {
	switch {
	case strings.Contains(s, "+"):
		elements := strings.Split(s, "+")
		for i := range elements {
			elements[i] = strings.TrimSpace(elements[i])
		}
		return TagsQuery{Elements: elements, Operator: AndOp}
	case strings.Contains(s, ","):
		elements := strings.Split(s, ",")
		for i := range elements {
			elements[i] = strings.TrimSpace(elements[i])
		}
		return TagsQuery{Elements: elements, Operator: OrOp}
	default:
		return TagsQuery{Elements: []string{s}, Operator: OrOp}
	}
}

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total      uint64    `json:"total"`
	Offset     uint64    `json:"offset"`
	Limit      uint64    `json:"limit"`
	OnlyTotal  bool      `json:"only_total"`
	Name       string    `json:"name,omitempty"`
	ID         string    `json:"id,omitempty"`
	Dir        string    `json:"dir,omitempty"`
	Order      string    `json:"order,omitempty"`
	Path       string    `json:"path,omitempty"`
	DomainID   string    `json:"domain_id,omitempty"`
	Tags       TagsQuery `json:"tags,omitempty"`
	Metadata   Metadata  `json:"metadata,omitempty"`
	Status     Status    `json:"status,omitempty"`
	RoleName   string    `json:"role_name,omitempty"`
	RoleID     string    `json:"role_id,omitempty"`
	Actions    []string  `json:"actions,omitempty"`
	AccessType string    `json:"access_type,omitempty"`
	RootGroup  bool      `json:"root_group,omitempty"`
}
