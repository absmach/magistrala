package sdk_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	bsmocks "github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/certs/api"
	"github.com/mainflux/mainflux/certs/mocks"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things"
	thmocks "github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingsNum      = 1
	thingKey       = "thingKey"
	caPath         = "../../../docker/ssl/certs/ca.crt"
	caKeyPath      = "../../../docker/ssl/certs/ca.key"
	cfgAuthTimeout = "1s"
	cfgLogLevel    = "error"
	cfgClientTLS   = false
	cfgServerCert  = ""
	cfgServerKey   = ""
	cfgCertsURL    = "http://localhost"
	cfgJaegerURL   = ""
	cfgAuthURL     = "localhost:8181"
	thingID        = "1"
	ttl            = "24h"
	keyBits        = 2048
	invalidKeyBits = -1
	key            = "rsa"
)

func newCertService(tokens map[string]string) (certs.Service, error) {
	ac := bsmocks.NewAuthClient(map[string]string{token: email})
	server := newThingsServer(newCertThingsService(ac))

	policies := []thmocks.MockSubjectSet{{Object: "users", Relation: "member"}}
	auth := thmocks.NewAuthService(tokens, map[string][]thmocks.MockSubjectSet{email: policies})
	config := sdk.Config{
		ThingsURL: server.URL,
	}

	sdk := sdk.NewSDK(config)
	repo := mocks.NewCertsRepository()

	tlsCert, caCert, err := loadCertificates(caPath, caKeyPath)
	if err != nil {
		return nil, err
	}

	authTimeout, err := time.ParseDuration(cfgAuthTimeout)
	if err != nil {
		return nil, err
	}

	c := certs.Config{
		LogLevel:       cfgLogLevel,
		ClientTLS:      cfgClientTLS,
		ServerCert:     cfgServerCert,
		ServerKey:      cfgServerKey,
		CertsURL:       cfgCertsURL,
		JaegerURL:      cfgJaegerURL,
		AuthURL:        cfgAuthURL,
		SignTLSCert:    tlsCert,
		SignX509Cert:   caCert,
		SignHoursValid: ttl,
		SignRSABits:    keyBits,
	}
	pki := mocks.NewPkiAgent(tlsCert, caCert, keyBits, ttl, authTimeout)

	return certs.New(auth, repo, sdk, c, pki), nil
}

func newCertThingsService(auth mainflux.AuthServiceClient) things.Service {
	ths := make(map[string]things.Thing, thingsNum)
	for i := 0; i < thingsNum; i++ {
		id := strconv.Itoa(i + 1)
		ths[id] = things.Thing{
			ID:    id,
			Key:   thingKey,
			Owner: email,
		}
	}

	return bsmocks.NewThingsService(ths, map[string]things.Channel{}, auth)
}

func newCertServer(svc certs.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, logger)
	return httptest.NewServer(mux)
}

func TestIssueCert(t *testing.T) {
	svc, err := newCertService(map[string]string{token: token})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))
	cs := newCertServer(svc)
	defer cs.Close()
	sdkConf := sdk.Config{
		CertsURL:        cs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc    string
		thingID string
		keyBits int
		keyType string
		ttl     string
		token   string
		err     error
	}{
		{
			desc:    "issue new cert with invalid token",
			thingID: thingID,
			keyBits: keyBits,
			keyType: key,
			ttl:     ttl,
			token:   invalidToken,
			err:     createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:    "issue new cert with empty token",
			thingID: thingID,
			keyBits: keyBits,
			keyType: key,
			ttl:     ttl,
			token:   "",
			err:     createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
		},
		{
			desc:    "issue new cert for non-existing thing",
			thingID: wrongID,
			keyBits: keyBits,
			keyType: key,
			ttl:     ttl,
			token:   token,
			err:     createError(sdk.ErrFailedCreation, http.StatusInternalServerError),
		},
		{
			desc:    "issue new cert with empty thing ID",
			thingID: "",
			keyBits: keyBits,
			keyType: key,
			ttl:     ttl,
			token:   token,
			err:     createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:    "issue new cert with empty time to live",
			thingID: thingID,
			keyBits: keyBits,
			keyType: key,
			ttl:     "",
			token:   token,
			err:     createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:    "issue new cert with zero key bits",
			thingID: thingID,
			keyBits: 0,
			keyType: key,
			ttl:     ttl,
			token:   token,
			err:     createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},
		{
			desc:    "issue new cert with invalid key bits",
			thingID: thingID,
			keyBits: invalidKeyBits,
			keyType: key,
			ttl:     ttl,
			token:   token,
			err:     createError(sdk.ErrFailedCreation, http.StatusInternalServerError),
		},
		{
			desc:    "issue new cert with empty type",
			thingID: thingID,
			keyBits: keyBits,
			keyType: "",
			ttl:     ttl,
			token:   token,
			err:     createError(sdk.ErrFailedCreation, http.StatusBadRequest),
		},

		{
			desc:    "issue new cert",
			thingID: thingID,
			keyBits: keyBits,
			keyType: key,
			ttl:     ttl,
			token:   token,
			err:     nil,
		},
	}
	for _, tc := range cases {
		_, err = mainfluxSDK.IssueCert(tc.thingID, tc.keyBits, tc.keyType, tc.ttl, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
func TestRevokeCert(t *testing.T) {
	svc, err := newCertService(map[string]string{token: token})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))
	cs := newCertServer(svc)
	defer cs.Close()
	sdkConf := sdk.Config{
		CertsURL:        cs.URL,
		MsgContentType:  contentType,
		TLSVerification: true,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	_, err = mainfluxSDK.IssueCert(thingID, keyBits, key, ttl, token)
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	cases := []struct {
		desc    string
		thingID string
		token   string
		err     error
	}{
		{
			desc:    "revoke cert with with invalid token",
			thingID: thingID,
			token:   invalidToken,
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "revoke cert with empty token",
			thingID: thingID,
			token:   "",
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "revoke cert for non existing thing",
			thingID: wrongID,
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusInternalServerError),
		},
		{
			desc:    "revoke cert for an empty thing ID",
			thingID: "",
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:    "revoke cert",
			thingID: thingID,
			token:   token,
			err:     nil,
		},
		{
			desc:    "revoke already revoked cert",
			thingID: thingID,
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusInternalServerError),
		},
	}

	for _, tc := range cases {
		err = mainfluxSDK.RevokeCert(tc.thingID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func loadCertificates(caPath, caKeyPath string) (tls.Certificate, *x509.Certificate, error) {
	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if caPath == "" || caKeyPath == "" {
		return tlsCert, caCert, nil
	}

	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return tlsCert, caCert, err
	}

	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		return tlsCert, caCert, err
	}

	tlsCert, err := tls.LoadX509KeyPair(caPath, caKeyPath)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(err, err)
	}

	b, err := ioutil.ReadFile(caPath)
	if err != nil {
		return tlsCert, caCert, err
	}

	caCert, err = readCert(b)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(err, err)
	}

	return tlsCert, caCert, nil
}

func readCert(b []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode PEM data")
	}

	return x509.ParseCertificate(block.Bytes)
}
