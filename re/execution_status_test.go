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
			desc:   "Aborted status",
			status: AbortedStatus,
			want:   Aborted,
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
			desc:   "Partial Success status",
			status: PartialSuccessStatus,
			want:   PartialSuccess,
		},
		{
			desc:   "Never Run status",
			status: NeverRunStatus,
			want:   NeverRun,
		},
		{
			desc:   "Unknown status",
			status: ExecutionStatus(99),
			want:   UnknownExec,
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
			desc:   "Aborted status",
			status: Aborted,
			want:   AbortedStatus,
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
			desc:   "Partial Success status",
			status: PartialSuccess,
			want:   PartialSuccessStatus,
		},
		{
			desc:   "Never Run status",
			status: NeverRun,
			want:   NeverRunStatus,
		},
		{
			desc:   "Empty string defaults to Never Run",
			status: "",
			want:   NeverRunStatus,
		},
		{
			desc:    "Invalid status",
			status:  "invalid",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := ToExecutionStatus(tc.status)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
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
			desc:   "Never Run status",
			status: NeverRunStatus,
			want:   `"never_run"`,
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
			desc: "Never Run status",
			data: `"never_run"`,
			want: NeverRunStatus,
		},
		{
			desc:    "Invalid status",
			data:    `"invalid"`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var status ExecutionStatus
			err := status.UnmarshalJSON([]byte(tc.data))
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, status)
		})
	}
}