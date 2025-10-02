// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutionStatusString(t *testing.T) {
	cases := []struct {
		desc   string
		status ExecutionStatus
		want   string
	}{
		{
			desc:   "Never Run status",
			status: NeverRunStatus,
			want:   NeverRun,
		},
		{
			desc:   "Success status",
			status: SuccessStatus,
			want:   Success,
		},
		{
			desc:   "Failure status",
			status: FailureStatus,
			want:   Failure,
		},
		{
			desc:   "Partial Success status",
			status: PartialSuccessStatus,
			want:   PartialSuccess,
		},
		{
			desc:   "Queued status",
			status: QueuedStatus,
			want:   Queued,
		},
		{
			desc:   "In Progress status",
			status: InProgressStatus,
			want:   InProgress,
		},
		{
			desc:   "Aborted status",
			status: AbortedStatus,
			want:   Aborted,
		},
		{
			desc:   "Unknown status",
			status: UnknownExecutionStatus,
			want:   UnknownExecution,
		},
		{
			desc:   "Invalid status (out of range)",
			status: ExecutionStatus(99),
			want:   UnknownExecution,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.status.String()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestToExecutionStatus(t *testing.T) {
	cases := []struct {
		desc    string
		status  string
		want    ExecutionStatus
		wantErr bool
	}{
		{
			desc:   "Never Run status",
			status: NeverRun,
			want:   NeverRunStatus,
		},
		{
			desc:   "Success status",
			status: Success,
			want:   SuccessStatus,
		},
		{
			desc:   "Failure status",
			status: Failure,
			want:   FailureStatus,
		},
		{
			desc:   "Partial Success status",
			status: PartialSuccess,
			want:   PartialSuccessStatus,
		},
		{
			desc:   "Queued status",
			status: Queued,
			want:   QueuedStatus,
		},
		{
			desc:   "In Progress status",
			status: InProgress,
			want:   InProgressStatus,
		},
		{
			desc:   "Aborted status",
			status: Aborted,
			want:   AbortedStatus,
		},
		{
			desc:   "Unknown status string",
			status: UnknownExecution,
			want:   UnknownExecutionStatus,
		},
		{
			desc:   "Empty string defaults to Unknown",
			status: "",
			want:   UnknownExecutionStatus,
		},
		{
			desc:    "Invalid status",
			status:  "invalid",
			want:    UnknownExecutionStatus,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := ToExecutionStatus(tc.status)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExecutionStatusMarshalJSON(t *testing.T) {
	cases := []struct {
		desc   string
		status ExecutionStatus
		want   string
	}{
		{
			desc:   "Never Run status",
			status: NeverRunStatus,
			want:   `"never_run"`,
		},
		{
			desc:   "Success status",
			status: SuccessStatus,
			want:   `"success"`,
		},
		{
			desc:   "Failure status",
			status: FailureStatus,
			want:   `"failure"`,
		},
		{
			desc:   "Partial Success status",
			status: PartialSuccessStatus,
			want:   `"partial_success"`,
		},
		{
			desc:   "Queued status",
			status: QueuedStatus,
			want:   `"queued"`,
		},
		{
			desc:   "In Progress status",
			status: InProgressStatus,
			want:   `"in_progress"`,
		},
		{
			desc:   "Aborted status",
			status: AbortedStatus,
			want:   `"aborted"`,
		},
		{
			desc:   "Unknown status",
			status: UnknownExecutionStatus,
			want:   `"unknown"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := tc.status.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestExecutionStatusUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc    string
		data    string
		want    ExecutionStatus
		wantErr bool
	}{
		{
			desc: "Never Run status",
			data: `"never_run"`,
			want: NeverRunStatus,
		},
		{
			desc: "Success status",
			data: `"success"`,
			want: SuccessStatus,
		},
		{
			desc: "Failure status",
			data: `"failure"`,
			want: FailureStatus,
		},
		{
			desc: "Partial Success status",
			data: `"partial_success"`,
			want: PartialSuccessStatus,
		},
		{
			desc: "Queued status",
			data: `"queued"`,
			want: QueuedStatus,
		},
		{
			desc: "In Progress status",
			data: `"in_progress"`,
			want: InProgressStatus,
		},
		{
			desc: "Aborted status",
			data: `"aborted"`,
			want: AbortedStatus,
		},
		{
			desc: "Unknown status string",
			data: `"unknown"`,
			want: UnknownExecutionStatus,
		},
		{
			desc: "Empty string",
			data: `""`,
			want: UnknownExecutionStatus,
		},
		{
			desc:    "Invalid status",
			data:    `"invalid"`,
			want:    UnknownExecutionStatus,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var status ExecutionStatus
			err := status.UnmarshalJSON([]byte(tc.data))
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, status)
		})
	}
}
