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
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Reports []Report `json:"reports"`
	PDF     []byte
	CSV     []byte
}

type ReportConfig struct {
	Name            string    `json:"name"`
	DomainID        string    `json:"domain_id"`
	ChannelIDs      []string  `json:"channel_ids"`
	ClientIDs       []string  `json:"client_ids"`
	StartDateTime   time.Time `json:"start_datetime,omitempty"`
	Time            time.Time `json:"time,omitempty"`
	Recurring       Recurring `json:"recurring,omitempty"`
	RecurringPeriod uint      `json:"recurring_period,omitempty"`
	Aggregation     string    `json:"aggregation,omitempty"`
	Email           *Email    `json:"email,omitempty"`
	Metrics         []string    `json:"metrics,omitempty"`
}

type Email struct {
	To      []string `json:"to,omitempty"`
	From    string   `json:"from,omitempty"`
	Subject string   `json:"subject,omitempty"`
}
