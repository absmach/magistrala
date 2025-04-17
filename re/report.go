// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"time"

	"github.com/absmach/supermq/pkg/transformers/senml"
)

type Report struct {
	Metric   Metric          `json:"metric,omitempty"`
	Messages []senml.Message `json:"messages,omitempty"`
}

type ReportPage struct {
	Total       uint64    `json:"total"`
	From        time.Time `json:"from,omitempty"`
	To          time.Time `json:"to,omitempty"`
	Aggregation AggConfig `json:"aggregation,omitempty"`
	Reports     []Report  `json:"reports"`
	PDF         []byte    `json:"pdf,omitempty"`
	CSV         []byte    `json:"csv,omitempty"`
}

type AggConfig struct {
	AggType  string `json:"agg_type,omitempty"` // Optional field
	Interval string `json:"interval,omitempty"` // Mandatory field if "AggType" field is set MAX, MIN, COUNT, SUM, AVG
}

type MetricConfig struct {
	From string `json:"from,omitempty"` // Mandatory field
	To   string `json:"to,omitempty"`   // Mandatory field

	Aggregation AggConfig `json:"aggregation,omitempty"` // Optional field
}

type Metric struct {
	ChannelID string `json:"channel_id,omitempty"` // Mandatory field
	ClientID  string `json:"client_id,omitempty"`  // Mandatory field
	Name      string `json:"name,omitempty"`       // Mandatory field
	Subtopic  string `json:"subtopic,omitempty"`   // Optional field
	Protocol  string `json:"protocol,omitempty"`   // Optional field
	Format    string `json:"format,omitiempty"`    // Optional field
}

type ReportConfig struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	DomainID    string        `json:"domain_id"`
	Schedule    Schedule      `json:"schedule,omitempty"`
	Config      *MetricConfig `json:"config,omitempty"`
	Email       *Email        `json:"email,omitempty"`
	Metrics     []Metric      `json:"metrics,omitempty"`
	Status      Status        `json:"status"`
	CreatedAt   time.Time     `json:"created_at,omitempty"`
	CreatedBy   string        `json:"created_by,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at,omitempty"`
	UpdatedBy   string        `json:"updated_by,omitempty"`
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
