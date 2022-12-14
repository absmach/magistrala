package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	bsapi "github.com/mainflux/mainflux/bootstrap/api"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/logger"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	unknown    = "unknown"
	wrongID    = "wrong_id"
	clientCert = "newCert"
	clientKey  = "newKey"
	caCert     = "newCert"
	invalidKey = "invalidKey"
	invalidID  = "invalidID"
)

var (
	channelsNum = 3
	encKey      = []byte("1234567891011121")
	channel     = sdk.Channel{
		ID:       "1",
		Name:     "name",
		Metadata: map[string]interface{}{"name": "value"},
	}
	config = sdk.BootstrapConfig{
		ExternalID:  "external_id",
		ExternalKey: "external_key",
		MFChannels:  []sdk.Channel{channel},
		Content:     "config",
		MFKey:       "mfKey",
		State:       0,
	}
)

func generateChannels() map[string]things.Channel {
	channels := make(map[string]things.Channel, channelsNum)
	for i := 0; i < channelsNum; i++ {
		id := strconv.Itoa(i + 1)
		channels[id] = things.Channel{
			ID:       id,
			Owner:    email,
			Metadata: metadata,
		}
	}
	return channels
}

func newBootstrapThingsService(auth mainflux.AuthServiceClient) things.Service {
	return mocks.NewThingsService(map[string]things.Thing{}, generateChannels(), auth)
}

func newBootstrapService(auth mainflux.AuthServiceClient, url string) bootstrap.Service {
	things := mocks.NewConfigsRepository()
	config := sdk.Config{
		ThingsURL: url,
	}
	sdk := sdk.NewSDK(config)
	return bootstrap.New(auth, things, sdk, encKey)
}

func newBootstrapServer(svc bootstrap.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := bsapi.MakeHandler(svc, bootstrap.NewConfigReader(encKey), logger)
	return httptest.NewServer(mux)
}
func TestAddBootstrap(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: token})
	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc   string
		config sdk.BootstrapConfig
		auth   string
		err    error
	}{

		{
			desc:   "add a config with invalid token",
			config: config,
			auth:   invalidToken,
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:   "add a config with empty token",
			config: config,
			auth:   "",
			err:    createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:   "add a config with invalid config",
			config: sdk.BootstrapConfig{},
			auth:   token,
			err:    createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:   "add a valid config",
			config: config,
			auth:   token,
			err:    nil,
		},
		{
			desc:   "add an existing config",
			config: config,
			auth:   token,
			err:    createError(sdk.ErrFailedCreation, http.StatusConflict),
		},
	}
	for _, tc := range cases {
		_, err := mainfluxSDK.AddBootstrap(tc.auth, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestWhitelist(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: token})
	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	mfThingID, err := mainfluxSDK.AddBootstrap(token, config)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	updtConfig := config
	updtConfig.MFThing = mfThingID
	wrongConfig := config
	wrongConfig.MFThing = wrongID

	cases := []struct {
		desc   string
		config sdk.BootstrapConfig
		auth   string
		err    error
	}{
		{
			desc:   "change state with invalid token",
			config: updtConfig,
			auth:   invalidToken,
			err:    createError(sdk.ErrFailedWhitelist, http.StatusUnauthorized),
		},
		{
			desc:   "change state with empty token",
			config: updtConfig,
			auth:   "",
			err:    createError(sdk.ErrFailedWhitelist, http.StatusUnauthorized),
		},
		{
			desc:   "change state of non-existing config",
			config: wrongConfig,
			auth:   token,
			err:    createError(sdk.ErrFailedWhitelist, http.StatusNotFound),
		},
		{
			desc:   "change state to active",
			config: updtConfig,
			auth:   token,
			err:    nil,
		},
		{
			desc:   "change state to current state",
			config: updtConfig,
			auth:   token,
			err:    nil,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.Whitelist(tc.auth, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestViewBootstrap(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: token})
	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	thingID, err := mainfluxSDK.AddBootstrap(token, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc string
		id   string
		auth string
		err  error
	}{
		{
			desc: "view a non-existing config",
			id:   unknown,
			auth: token,
			err:  createError(sdk.ErrFailedFetch, http.StatusNotFound),
		},
		{
			desc: "view a config with invalid token",
			id:   thingID,
			auth: invalidToken,
			err:  createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
		{
			desc: "view a config with empty token",
			id:   thingID,
			auth: "",
			err:  createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
		{
			desc: "view an existing config",
			id:   thingID,
			auth: token,
			err:  nil,
		},
	}
	for _, tc := range cases {
		_, err := mainfluxSDK.ViewBootstrap(tc.auth, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestUpdateBootstrap(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: email})
	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	thingID, err := mainfluxSDK.AddBootstrap(token, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	updatedConfig := config
	updatedConfig.MFThing = thingID
	ch := channel
	ch.ID = "2"
	updatedConfig.MFChannels = append(updatedConfig.MFChannels, ch)
	nonExisting := config
	nonExisting.MFThing = unknown

	cases := []struct {
		desc   string
		auth   string
		config sdk.BootstrapConfig
		err    error
	}{
		{
			desc:   "update config with invalid token",
			auth:   invalidToken,
			config: updatedConfig,
			err:    createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc:   "update config with empty token",
			auth:   "",
			config: updatedConfig,
			err:    createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc:   "update a non-existing config",
			auth:   token,
			config: nonExisting,
			err:    createError(sdk.ErrFailedUpdate, http.StatusNotFound),
		},
		{
			desc:   "update a config with state created",
			auth:   token,
			config: updatedConfig,
			err:    nil,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.UpdateBootstrap(tc.auth, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestUpdateBootstrapCerts(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: email})
	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	_, err := mainfluxSDK.AddBootstrap(token, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	updatedConfig := config
	updatedConfig.MFKey = thingID
	ch := channel
	ch.ID = "2"
	updatedConfig.MFChannels = append(updatedConfig.MFChannels, ch)

	cases := []struct {
		desc       string
		id         string
		clientCert string
		clientKey  string
		caCert     string
		auth       string
		err        error
	}{
		{
			desc:       "update cert for a non-existing config",
			id:         wrongID,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			auth:       token,
			err:        createError(sdk.ErrFailedCertUpdate, http.StatusNotFound),
		},
		{
			desc:       "update cert with with invalid token",
			id:         updatedConfig.MFKey,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			auth:       invalidToken,
			err:        createError(sdk.ErrFailedCertUpdate, http.StatusUnauthorized),
		},
		{
			desc:       "update cert with an empty token",
			id:         updatedConfig.MFKey,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			auth:       "",
			err:        createError(sdk.ErrFailedCertUpdate, http.StatusUnauthorized),
		},
		{
			desc:       "update cert for the valid config",
			id:         updatedConfig.MFKey,
			clientCert: clientCert,
			clientKey:  clientKey,
			caCert:     caCert,
			auth:       token,
			err:        nil,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.UpdateBootstrapCerts(tc.auth, tc.id, tc.clientCert, tc.clientKey, tc.caCert)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveBootstrap(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: email})

	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	mfThingID, err := mainfluxSDK.AddBootstrap(token, config)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove config with invalid token",
			id:    mfThingID,
			token: invalidToken,
			err:   createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:  "remove config with empty token",
			id:    mfThingID,
			token: "",
			err:   createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:  "remove non-existing config",
			id:    unknown,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove an existing config",
			id:    mfThingID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove already removed config",
			id:    mfThingID,
			token: token,
			err:   nil,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.RemoveBootstrap(tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestBootstrap(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{token: email})

	ts := newThingsServer(newBootstrapThingsService(auth))
	svc := newBootstrapService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	defer bs.Close()

	sdkConf := sdk.Config{
		BootstrapURL:    bs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	mfThingID, err := mainfluxSDK.AddBootstrap(token, config)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	updtConfig := config
	updtConfig.MFThing = mfThingID

	cases := []struct {
		desc        string
		config      sdk.BootstrapConfig
		externalKey string
		externalID  string
		err         error
	}{
		{
			desc:        "bootstrap an existing config",
			config:      updtConfig,
			externalID:  updtConfig.ExternalID,
			externalKey: updtConfig.ExternalKey,
			err:         nil,
		},
		{
			desc:        "bootstrap config with empty external ID",
			config:      sdk.BootstrapConfig{},
			externalID:  "",
			externalKey: updtConfig.ExternalKey,
			err:         createError(sdk.ErrFailedFetch, http.StatusBadRequest),
		},
		{
			desc:        "bootstrap config with invalid ID",
			config:      sdk.BootstrapConfig{},
			externalID:  invalidID,
			externalKey: updtConfig.ExternalKey,
			err:         createError(sdk.ErrFailedFetch, http.StatusNotFound),
		},
		{
			desc:        "bootstrap config with empty extrnal key",
			config:      sdk.BootstrapConfig{},
			externalID:  updtConfig.ExternalID,
			externalKey: "",
			err:         createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
		},
		{
			desc:        "bootstrap config with invalid key",
			config:      sdk.BootstrapConfig{},
			externalID:  updtConfig.ExternalID,
			externalKey: invalidKey,
			err:         createError(sdk.ErrFailedFetch, http.StatusForbidden),
		},
	}
	for _, tc := range cases {
		_, err := mainfluxSDK.Bootstrap(tc.externalKey, tc.externalID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
