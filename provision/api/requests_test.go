package api

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
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
			err: apiutil.ErrMissingID,
		},
	}

	for desc, tc := range cases {
		req := provisionReq{
			ExternalID:  tc.ExternalID,
			ExternalKey: tc.ExternalKey,
		}

		err := req.validate()
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected `%v` got `%v`", desc, tc.err, err))
	}
}
