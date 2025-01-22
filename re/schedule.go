// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"
	"time"
)

type Schedule struct {
	StartDateTime   time.Time     `json:"start_datetime"`   // When the schedule becomes active
	RecurringTime   time.Time     `json:"recurring_time"`   // Specific time for the rule to run
	RecurringType   ReccuringType `json:"recurring_type"`   // None, Daily, Weekly, Monthly
	RecurringPeriod uint          `json:"recurring_period"` // Controls how many intervals to skip between executions: 1 runs at every interval, 2 runs at every second interval, etc.
}

func (s Schedule) MarshalJSON() ([]byte, error) {
	type Alias Schedule
	jTimes := struct {
		StartDateTime string `json:"start_datetime"`
		RecurringTime string `json:"recurring_time"`
		*Alias
	}{
		StartDateTime: s.StartDateTime.Format(timeFormat),
		RecurringTime: s.RecurringTime.Format(timeFormat),
		Alias:         (*Alias)(&s),
	}
	return json.Marshal(jTimes)
}

func (s *Schedule) UnmarshalJSON(data []byte) error {
	type Alias Schedule
	aux := struct {
		StartDateTime string `json:"start_datetime"`
		RecurringTime string `json:"recurring_time"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.StartDateTime != "" {
		startDateTime, err := time.Parse(timeFormat, aux.StartDateTime)
		if err != nil {
			return err
		}
		s.StartDateTime = startDateTime
	}

	if aux.RecurringTime != "" {
		recurringTime, err := time.Parse(timeFormat, aux.RecurringTime)
		if err != nil {
			return err
		}
		s.RecurringTime = recurringTime
	}
	return nil
}

// Type can be daily, weekly or monthly.
type ReccuringType uint

const (
	None ReccuringType = iota
	Daily
	Weekly
	Monthly
)

func (rt ReccuringType) String() string {
	switch rt {
	case Daily:
		return "daily"
	case Weekly:
		return "weekly"
	case Monthly:
		return "monthly"
	default:
		return "none"
	}
}

func (rt ReccuringType) MarshalJSON() ([]byte, error) {
	return json.Marshal(rt.String())
}

func (rt *ReccuringType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "daily":
		*rt = Daily
	case "weekly":
		*rt = Weekly
	case "monthly":
		*rt = Monthly
	case "none":
		*rt = None
	default:
		return ErrInvalidRecurringType
	}
	return nil
}
