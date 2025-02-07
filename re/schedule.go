// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"
	"time"
)

type Schedule struct {
	StartDateTime   time.Time `json:"start_datetime"`   // When the schedule becomes active
	Time            time.Time `json:"time"`             // Specific time for the rule to run
	Recurring       Recurring `json:"recurring"`        // None, Daily, Weekly, Monthly
	RecurringPeriod uint      `json:"recurring_period"` // Controls how many intervals to skip between executions: 1 = every interval, 2 = every second interval, etc.
}

func (s Schedule) MarshalJSON() ([]byte, error) {
	type Alias Schedule
	jTimes := struct {
		StartDateTime string `json:"start_datetime"`
		Time          string `json:"time"`
		*Alias
	}{
		StartDateTime: s.StartDateTime.Format(time.RFC3339),
		Time:          s.Time.Format(timeFormat),
		Alias:         (*Alias)(&s),
	}
	return json.Marshal(jTimes)
}

func (s *Schedule) UnmarshalJSON(data []byte) error {
	type Alias Schedule
	aux := struct {
		StartDateTime string `json:"start_datetime"`
		Time          string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	startDateTime, err := time.Parse(time.RFC3339, aux.StartDateTime)
	if err != nil {
		return err
	}
	s.StartDateTime = startDateTime

	if aux.Time != "" {
		time, err := time.Parse(timeFormat, aux.Time)
		if err != nil {
			return err
		}
		s.Time = time
	}
	return nil
}

// Type can be daily, weekly or monthly.
type Recurring uint

const (
	None Recurring = iota
	Daily
	Weekly
	Monthly
)

func (rt Recurring) String() string {
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

func (rt Recurring) MarshalJSON() ([]byte, error) {
	return json.Marshal(rt.String())
}

func (rt *Recurring) UnmarshalJSON(data []byte) error {
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
