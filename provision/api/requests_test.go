package api

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {

	cases := map[string]struct {
		ExternalID  string
		ExternalKey string
		err         error
	}{
		"mac address for device": {
			ExternalID:  "11:22:33:44:55:66",
			ExternalKey: "key12345678",
			err:         nil,
		},
		"external id for device empty": {
			err: errUnauthorized,
		},
	}

	for desc, tc := range cases {
		req := addThingReq{
			ExternalID:  tc.ExternalID,
			ExternalKey: tc.ExternalKey,
		}

		err := req.validate()
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected `%v` got `%v`", desc, err, tc.err))
	}
}
