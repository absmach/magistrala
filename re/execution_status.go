// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"encoding/json"
	"strings"

	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

// ExecutionStatus represents the last run status of a rule execution.
type ExecutionStatus uint8

// Possible execution status values.
const (
	// NeverRunStatus represents a rule that has never been executed.
	NeverRunStatus ExecutionStatus = iota
	// SuccessStatus represents a successful rule execution.
	SuccessStatus
	// FailureStatus represents a failed rule execution.
	FailureStatus
	// PartialSuccessStatus represents a rule execution with partial success.
	PartialSuccessStatus
	// QueuedStatus represents a rule that is queued for execution.
	QueuedStatus
	// InProgressStatus represents a rule that is currently being executed.
	InProgressStatus
	// AbortedStatus represents a rule execution that was aborted.
	AbortedStatus
	// UnknownExecutionStatus represents an unknown execution status.
	UnknownExecutionStatus
)

// String representation of the possible execution status values.
const (
	NeverRun         = "never_run"
	Success          = "success"
	Failure          = "failure"
	PartialSuccess   = "partial_success"
	Queued           = "queued"
	InProgress       = "in_progress"
	Aborted          = "aborted"
	UnknownExecution = "unknown"
)

func (es ExecutionStatus) String() string {
	switch es {
	case NeverRunStatus:
		return NeverRun
	case SuccessStatus:
		return Success
	case FailureStatus:
		return Failure
	case PartialSuccessStatus:
		return PartialSuccess
	case QueuedStatus:
		return Queued
	case InProgressStatus:
		return InProgress
	case AbortedStatus:
		return Aborted
	default:
		return UnknownExecution
	}
}

// ToExecutionStatus converts string value to a valid execution status.
func ToExecutionStatus(status string) (ExecutionStatus, error) {
	switch status {
	case NeverRun:
		return NeverRunStatus, nil
	case Success:
		return SuccessStatus, nil
	case Failure:
		return FailureStatus, nil
	case PartialSuccess:
		return PartialSuccessStatus, nil
	case Queued:
		return QueuedStatus, nil
	case InProgress:
		return InProgressStatus, nil
	case Aborted:
		return AbortedStatus, nil
	case "", UnknownExecution:
		return UnknownExecutionStatus, nil
	}
	return UnknownExecutionStatus, svcerr.ErrInvalidStatus
}

func (es ExecutionStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(es.String())
}

func (es *ExecutionStatus) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToExecutionStatus(str)
	*es = val
	return err
}
