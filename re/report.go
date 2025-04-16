// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

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

type AggConfig struct {
	AggType  string `json:"agg_type,omitempty"`
	Interval string `json:"interval,omitempty"`
}

type TimeExpression float64

func (te *TimeExpression) UnmarshalJSON(data []byte) error {
	var expr string
	if err := json.Unmarshal(data, &expr); err == nil {
		timestamp, err := parseTimeExpression(expr)
		if err != nil {
			return err
		}
		*te = TimeExpression(timestamp)
		return nil
	}

	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*te = TimeExpression(value)
	return nil
}

func parseTimeExpression(expr string) (float64, error) {
	if expr == "" {
		return 0, nil
	}

	expr = strings.TrimSpace(expr)
	if expr == "now()" || expr == "now" {
		return float64(time.Now().Unix()), nil
	}

	if strings.Contains(expr, "now()") || strings.Contains(expr, "now") {
		expr = strings.ReplaceAll(expr, "+", " + ")
		expr = strings.ReplaceAll(expr, "-", " - ")

		parts := strings.Fields(expr)
		if len(parts) != 3 {
			return 0, fmt.Errorf("invalid time expression format: %s", expr)
		}

		operation := parts[1]
		if operation != "+" && operation != "-" {
			return 0, fmt.Errorf("unsupported operation: %s", operation)
		}

		durStr := parts[2]
		valueStr := ""
		unitStr := ""
		for _, c := range durStr {
			if unicode.IsDigit(c) {
				valueStr += string(c)
			} else {
				unitStr += string(c)
			}
		}

		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration value: %s", valueStr)
		}

		var duration time.Duration
		switch unitStr {
		case "d":
			duration = time.Duration(value) * 24 * time.Hour
		case "h":
			duration = time.Duration(value) * time.Hour
		case "m":
			duration = time.Duration(value) * time.Minute
		case "s":
			duration = time.Duration(value) * time.Second
		default:
			return 0, fmt.Errorf("unsupported time unit: %s", unitStr)
		}

		now := time.Now()
		if operation == "+" {
			targetTime := now.Add(duration)
			return float64(targetTime.Unix()), nil
		} else {
			targetTime := now.Add(-duration)
			return float64(targetTime.Unix()), nil
		}
	}

	timestamp, err := strconv.ParseFloat(expr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp: %s", expr)
	}

	return timestamp, nil
}

type MetricConfig struct {
	From        TimeExpression `json:"from,omitempty"`
	To          TimeExpression `json:"to,omitempty"`
	Subtopic    string         `json:"subtopic,omitempty"`
	Protocol    string         `json:"protocol,omitempty"`
	Format      string         `json:"format,omitempty"`
	Aggregation AggConfig      `json:"aggregation,omitempty"`
}

type ReportConfig struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	DomainID   string       `json:"domain_id"`
	Limit      uint64       `json:"limit"`
	ChannelIDs []string     `json:"channel_ids"`
	ClientIDs  []string     `json:"client_ids"`
	Schedule   Schedule     `json:"schedule,omitempty"`
	Config     *MetricConfig `json:"config,omitempty"`
	Email      Email        `json:"email,omitempty"`
	Metrics    []string     `json:"metrics,omitempty"`
	Status     Status       `json:"status"`
	CreatedAt  time.Time    `json:"created_at,omitempty"`
	CreatedBy  string       `json:"created_by,omitempty"`
	UpdatedAt  time.Time    `json:"updated_at,omitempty"`
	UpdatedBy  string       `json:"updated_by,omitempty"`
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
