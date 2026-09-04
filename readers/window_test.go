// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package readers

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidTimeRangeRejectsRoundedMaxInt64Boundary(t *testing.T) {
	assert.True(t, ValidTimeRange(0))
	assert.True(t, ValidTimeRange(float64(math.MinInt64)))
	assert.False(t, ValidTimeRange(float64(math.MaxInt64)))
	assert.False(t, ValidTimeRange(9999999999999999999))
}
