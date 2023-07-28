// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package clients

// Page contains page metadata that helps navigation.
type Page struct {
	Total    uint64   `json:"total"`
	Offset   uint64   `json:"offset"`
	Limit    uint64   `json:"limit"`
	Name     string   `json:"name,omitempty"`
	Order    string   `json:"order,omitempty"`
	Dir      string   `json:"dir,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
	Owner    string   `json:"owner,omitempty"`
	Tag      string   `json:"tag,omitempty"`
	SharedBy string   `json:"shared_by,omitempty"`
	Status   Status   `json:"status,omitempty"`
	Action   string   `json:"action,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	IDs      []string `json:"ids,omitempty"`
	Identity string   `json:"identity,omitempty"`
}
