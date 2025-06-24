// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"encoding/json"
	"time"

	"github.com/absmach/supermq/pkg/errors"
)

var (
	ErrInvalidRecurringType = errors.New("invalid recurring type")
	ErrStartDateTimeInPast  = errors.New("start_datetime must be greater than or equal to current time")
)

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

type Schedule struct {
	StartDateTime   *time.Time `json:"start_datetime"`   // When the schedule becomes active
	Time            time.Time  `json:"time"`             // Specific time for the rule to run
	Recurring       Recurring  `json:"recurring"`        // None, Daily, Weekly, Monthly
	RecurringPeriod uint       `json:"recurring_period"` // Controls how many intervals to skip between executions: 1 = every interval, 2 = every second interval, etc.
}

func (s Schedule) Validate() error {
	if s.StartDateTime != nil {
		now := time.Now().UTC()
		if s.StartDateTime.Before(now) {
			return ErrStartDateTimeInPast
		}
	}
	return nil
}

func (s Schedule) MarshalJSON() ([]byte, error) {
	type Alias Schedule
	jTimes := struct {
		StartDateTime *string `json:"start_datetime"`
		Time          string  `json:"time"`
		*Alias
	}{
		Time:  s.Time.Format(time.RFC3339),
		Alias: (*Alias)(&s),
	}
	if s.StartDateTime != nil {
		formatted := s.StartDateTime.Format(time.RFC3339)
		jTimes.StartDateTime = &formatted
	}

	return json.Marshal(jTimes)
}

func (s *Schedule) UnmarshalJSON(data []byte) error {
	type Alias Schedule
	temp := struct {
		StartDateTime string `json:"start_datetime"`
		Time          string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	s.StartDateTime = nil
	if temp.StartDateTime != "" {
		startDateTime, err := time.Parse(time.RFC3339, temp.StartDateTime)
		if err != nil {
			return err
		}
		s.StartDateTime = &startDateTime
	}
	if temp.Time != "" {
		parsedTime, err := time.Parse(time.RFC3339, temp.Time)
		if err != nil {
			return err
		}
		s.Time = parsedTime
	}
	return nil
}

func (s Schedule) NextDue() time.Time {
	switch s.Recurring {
	case Daily:
		return s.Time.AddDate(0, 0, int(s.RecurringPeriod))
	case Weekly:
		return s.Time.AddDate(0, 0, int(s.RecurringPeriod)*7)
	case Monthly:
		return s.Time.AddDate(0, int(s.RecurringPeriod), 0)
	default:
		return time.Time{}
	}
}
