// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total      uint64   `json:"total"`
	Offset     uint64   `json:"offset"`
	Limit      uint64   `json:"limit"`
	Name       string   `json:"name,omitempty"`
	ID         string   `json:"id,omitempty"`
	Path       string   `json:"path,omitempty"`
	DomainID   string   `json:"domain_id,omitempty"`
	Tag        string   `json:"tag,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Status     Status   `json:"status,omitempty"`
	RoleName   string   `json:"role_name,omitempty"`
	RoleID     string   `json:"role_id,omitempty"`
	Actions    []string `json:"actions,omitempty"`
	AccessType string   `json:"access_type,omitempty"`
}
