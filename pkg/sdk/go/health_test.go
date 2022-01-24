package sdk_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
)

const (
	thingsDescription = "things service"
	thingsStatus      = "pass"
)

func TestHealth(t *testing.T) {
	svc := newThingsService(map[string]string{token: email})
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	cases := map[string]struct {
		empty bool
		err   error
	}{
		"get things service health check": {
			empty: false,
			err:   nil,
		},
	}
	for desc, tc := range cases {
		h, err := mainfluxSDK.Health()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
		assert.Equal(t, thingsStatus, h.Status, fmt.Sprintf("%s: expected %s status, got %s", desc, thingsStatus, h.Status))
		assert.Equal(t, tc.empty, h.Version == "", fmt.Sprintf("%s: expected non-empty version", desc))
		assert.Equal(t, mainflux.Commit, h.Commit, fmt.Sprintf("%s: expected non-empty commit", desc))
		assert.Equal(t, thingsDescription, h.Description, fmt.Sprintf("%s: expected proper description, got %s", desc, h.Description))
		assert.Equal(t, mainflux.BuildTime, h.BuildTime, fmt.Sprintf("%s: expected default epoch date, got %s", desc, h.BuildTime))
	}
}
