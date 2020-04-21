package provision_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/provision"
	"github.com/mainflux/mainflux/provision/mocks"
	sdk "github.com/mainflux/mainflux/provision/sdk"

	logger "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
)

var (
	cfg = provision.Config{
		SDK:              mocks.NewSDK(),
		MFEmail:          "test@example.com",
		MFPass:           "test",
		X509Provision:    true,
		BSProvision:      true,
		BSContent:        "",
		PredefinedThings: []string{"predefined"},
		AutoWhiteList:    true,
	}
	log, _ = logger.New(os.Stdout, "info")
)

func TestProvision(t *testing.T) {
	// Create multiple services with different configurations.
	svc1 := provision.New(cfg, log)

	cfg2 := cfg
	cfg2.PredefinedThings = []string{"invalid"}
	svc2 := provision.New(cfg2, log)

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		svc         provision.Service
		err         error
	}{
		{
			desc:        "Provision successfully",
			externalID:  "id",
			externalKey: "key",
			svc:         svc1,
			err:         nil,
		},
		{
			desc:        "Provision already existing config",
			externalID:  "id",
			externalKey: "key",
			svc:         svc1,
			err:         sdk.ErrConfig,
		},
		{
			desc:        "Provision with invalid proxy ID",
			externalID:  "id",
			externalKey: "key",
			svc:         svc2,
			err:         sdk.ErrConn,
		},
	}

	for _, tc := range cases {
		_, err := tc.svc.Provision(tc.externalID, tc.externalKey)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
