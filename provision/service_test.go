package provision_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/provision"
	"github.com/mainflux/mainflux/provision/mocks"

	logger "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
)

var (
	cfg = provision.Config{
		Bootstrap: provision.Bootstrap{
			AutoWhiteList: true,
			Provision:     true,
			Content:       "",
			X509Provision: true,
		},
		Server: provision.ServiceConf{
			MfPass: "test",
			MfUser: "test@example.com",
		},
		Channels: []provision.Channel{
			provision.Channel{
				Name:     "control-channel",
				Metadata: map[string]interface{}{"type": "control"},
			},
			provision.Channel{
				Name:     "data-channel",
				Metadata: map[string]interface{}{"type": "data"},
			},
		},
		Things: []provision.Thing{
			provision.Thing{
				Name:     "thing",
				Metadata: map[string]interface{}{"external_id": "xxxxxx"},
			},
		},
	}
	log, _ = logger.New(os.Stdout, "info")
)

func TestProvision(t *testing.T) {
	// Create multiple services with different configurations.
	sdk := mocks.NewSDK()
	svc := provision.New(cfg, sdk, log)

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
			svc:         svc,
			err:         nil,
		},
		{
			desc:        "Provision already existing config",
			externalID:  "id",
			externalKey: "key",
			svc:         svc,
			err:         provision.ErrFailedBootstrap,
		},
	}

	for _, tc := range cases {
		_, err := tc.svc.Provision("", "", tc.externalID, tc.externalKey)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected `%v` got `%v`", tc.desc, tc.err, err))
	}

}
