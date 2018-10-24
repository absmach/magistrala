//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/mainflux/mainflux/things"
	"github.com/stretchr/testify/assert"
)

func TestFromString(t *testing.T) {
	big := fmt.Sprintf("%d", uint64(math.MaxUint64))

	cases := map[string]struct {
		in  string
		out uint64
		err error
	}{
		"from valid number": {
			in:  big,
			out: math.MaxUint64,
			err: nil,
		},
		"from negative number": {
			in:  "-1",
			out: 0,
			err: things.ErrNotFound,
		},
		"from empty string": {
			in:  "",
			out: 0,
			err: things.ErrNotFound,
		},
		"from arbitrary string": {
			in:  "dummy",
			out: 0,
			err: things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		out, err := things.FromString(tc.in)
		assert.Equal(t, tc.out, out, fmt.Sprintf("%s: expected %d got %d", desc, tc.out, out))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s", desc, tc.err, err))
	}
}
