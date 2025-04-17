// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reltime

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	now := time.Now()

	tests := []struct {
		desc     string
		expr     string
		expected time.Time
		err      error
	}{
		{
			desc:     "testing expression now()-5d",
			expr:     "now()-5d",
			expected: now.Add(-5 * 24 * time.Hour),
			err:      nil,
		},
		{
			desc:     "testing expression now()+2h30m",
			expr:     "now()+2h30m",
			expected: now.Add(2*time.Hour + 30*time.Minute),
			err:      nil,
		},
		{
			desc:     "testing expression now()-1w3d10h40m",
			expr:     "now()-1w3d10h40m",
			expected: now.Add(-(7*24+3*24+10)*time.Hour - 40*time.Minute),
			err:      nil,
		},
		{
			desc: "testing expression yesterday",
			expr: "yesterday",
			err:  ErrInvalidExpression,
		},
		{
			desc: "testing expression now()--5d",
			expr: "now()--5d",
			err:  ErrInvalidExpression,
		},
		{
			desc: "testing expression now()+",
			expr: "now()+",
			err:  ErrInvalidExpression,
		},
		{
			desc: "testing expression now()+5r",
			expr: "now()+5r",
			err:  ErrInvalidDuration,
		},
		{
			desc: "testing expression now()+5M",
			expr: "now()+5M",
			err:  ErrUnsupportedUnit,
		},
	}

	for _, tc := range tests {
		got, err := Parse(tc.expr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v and response time %v\n", tc.desc, tc.err, err, got))
		if err == nil {
			assert.WithinDuration(t, tc.expected, got, time.Duration(10*time.Second))
		}
	}
}
