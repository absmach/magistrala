// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"time"

	"github.com/absmach/supermq/pkg/transformers/senml"
)

type Report struct {
	ClientMessages map[string][]senml.Message `json:"client_messages"`
}

type ReportPage struct {
	Total   uint64   `json:"total"`
	Reports []Report `json:"reports"`
	PDF     []byte   `json:"pdf,omitempty"`
	CSV     []byte   `json:"csv,omitempty"`
}

type ReportConfig struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DomainID    string    `json:"domain_id"`
	Limit       uint64    `json:"limit"`
	ChannelIDs  []string  `json:"channel_ids"`
	ClientIDs   []string  `json:"client_ids"`
	Schedule    Schedule  `json:"schedule,omitempty"`
	Aggregation string    `json:"aggregation,omitempty"`
	Email       Email     `json:"email,omitempty"`
	Metrics     []string  `json:"metrics,omitempty"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
}

type ReportConfigPage struct {
	PageMeta
	ReportConfigs []ReportConfig `json:"report_configs"`
}

type Email struct {
	To      []string `json:"to,omitempty"`
	From    string   `json:"from,omitempty"`
	Subject string   `json:"subject,omitempty"`
}
