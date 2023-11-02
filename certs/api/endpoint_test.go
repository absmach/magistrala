package api_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	bsmocks "github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/certs"
	api "github.com/mainflux/mainflux/certs/api"
	"github.com/mainflux/mainflux/certs/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/things/http"
	thmocks "github.com/mainflux/mainflux/things/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token             = "token"
	invalidToken      = "invalidToken"
	email             = "mainflux@email.com"
	thingsNum         = 1
	thingKey          = "thingKey"
	thingID           = "1"
	invalidThingID    = "invalidThingID"
	caPath            = "../../docker/ssl/certs/ca.crt"
	caKeyPath         = "../../docker/ssl/certs/ca.key"
	cfgAuthTimeout    = "1s"
	cfgLogLevel       = "error"
	cfgClientTLS      = false
	cfgServerCert     = ""
	cfgServerKey      = ""
	cfgCertsURL       = "http://localhost"
	cfgJaegerURL      = ""
	cfgAuthURL        = "localhost:8181"
	cfgSignHoursValid = "24h"
	cfgSignRSABits    = 2048
	contentType       = "application/json"
	ttl               = "1h"
	keyBits           = 2048
	key               = "rsa"
	certNum           = 10
	wrongCertID       = "wrongCertID"
)

var (
	addReq = struct {
		ThingID string `json:"thing_id"`
		TTL     string `json:"ttl"`
		KeyBits int    `json:"key_bits"`
		KeyType string `json:"key_type"`
	}{
		ThingID: thingID,
		TTL:     ttl,
		KeyBits: keyBits,
		KeyType: key,
	}
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func newCertService(tokens map[string]string) (certs.Service, error) {
	ac := bsmocks.NewAuthClient(map[string]string{token: email})
	server := newThingsServer(newCertThingsService(ac))

	policies := []thmocks.MockSubjectSet{{Object: "users", Relation: "member"}}
	auth := thmocks.NewAuthService(tokens, map[string][]thmocks.MockSubjectSet{email: policies})
	config := mfsdk.Config{
		ThingsURL: server.URL,
	}

	sdk := mfsdk.NewSDK(config)
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
		SignHoursValid: cfgSignHoursValid,
		SignRSABits:    cfgSignRSABits,
	}

	pki := mocks.NewPkiAgent(tlsCert, caCert, cfgSignRSABits, cfgSignHoursValid, authTimeout)

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
func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func newCertServer(svc certs.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	req.Header.Set("Referer", "http://localhost")
	return tr.client.Do(req)
}

func TestIssueCert(t *testing.T) {
	svc, err := newCertService(map[string]string{token: email})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	cs := newCertServer(svc)
	defer cs.Close()

	data := toJSON(addReq)
	wrongAddReq := addReq
	wrongAddReq.ThingID = "2"
	wrongData := toJSON(wrongAddReq)

	cases := []struct {
		desc        string
		req         string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "issue new cert",
			req:         data,
			auth:        token,
			contentType: contentType,
			status:      http.StatusCreated,
		},
		{
			desc:        "issue new cert for non existing thing ID",
			req:         data,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "issue new cert with empty token",
			req:         data,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "issue new cert with empty content type",
			req:         data,
			auth:        token,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "issue new cert with wrong JSON",
			req:         "{\"thing_id\": 5}",
			auth:        token,
			contentType: contentType,
			status:      http.StatusInternalServerError,
		},
		{
			desc:        "issue new cert with invalid request format",
			req:         "}",
			auth:        token,
			contentType: contentType,
			status:      http.StatusInternalServerError,
		},
		{
			desc:        "issue new cert with an empty request",
			req:         "",
			auth:        token,
			contentType: contentType,
			status:      http.StatusInternalServerError,
		},
		{
			desc:        "issue new cert with wrong thing",
			req:         wrongData,
			auth:        token,
			contentType: contentType,
			status:      http.StatusInternalServerError,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client:      cs.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/certs", cs.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListSerials(t *testing.T) {
	svc, err := newCertService(map[string]string{token: email})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	cs := newCertServer(svc)
	serialURL := fmt.Sprintf("%s/serials/%s", cs.URL, thingID)
	defer cs.Close()

	var issuedCerts []certsRes
	for i := 0; i < certNum; i++ {
		cert, err := svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, key)
		require.Nil(t, err, fmt.Sprintf("Certificate issuing expected to succeed: %s.\n", err))
		crt := certsRes{
			Serial: cert.Serial,
		}
		issuedCerts = append(issuedCerts, crt)
	}

	cases := []struct {
		desc   string
		url    string
		auth   string
		certs  certsPageRes
		status int
	}{
		{
			desc:   "list all cert ID's",
			url:    serialURL,
			auth:   token,
			certs:  certsPageRes{pageRes: pageRes{Total: uint64(certNum), Offset: 0, Limit: certNum}, Certs: issuedCerts},
			status: http.StatusOK,
		},
		{
			desc:   "list cert ID's with invalid thing ID",
			url:    fmt.Sprintf("%s/%s", serialURL, invalidThingID),
			auth:   token,
			certs:  certsPageRes{},
			status: http.StatusNotFound,
		},
		{
			desc:   "list all cert ID's with invalid token",
			url:    serialURL,
			auth:   invalidToken,
			certs:  certsPageRes{},
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list all cert ID's with empty token",
			url:    serialURL,
			auth:   "",
			certs:  certsPageRes{},
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list last cert ID",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", serialURL, 9, 10),
			auth:   token,
			certs:  certsPageRes{pageRes: pageRes{Total: uint64(certNum), Offset: certNum - 1, Limit: certNum}, Certs: issuedCerts[9:10]},
			status: http.StatusOK,
		},
		{
			desc:   "list last five cert ID's",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", serialURL, 5, 10),
			auth:   token,
			certs:  certsPageRes{pageRes: pageRes{Total: uint64(certNum), Offset: certNum - 5, Limit: certNum}, Certs: issuedCerts[5:10]},
			status: http.StatusOK,
		},
		{
			desc:   "list all certs with limit greater then allowed",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", serialURL, 0, 1000),
			auth:   token,
			certs:  certsPageRes{},
			status: http.StatusBadRequest,
		},
		{
			desc:   "list all certs with invalid limit",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", serialURL, 0, -10),
			auth:   token,
			certs:  certsPageRes{},
			status: http.StatusInternalServerError,
		},
		{
			desc:   "list all certs with invalid offset",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", serialURL, -10, 10),
			auth:   token,
			certs:  certsPageRes{},
			status: http.StatusInternalServerError,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: cs.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		if tc.status == http.StatusOK {
			var certs certsPageRes
			err = json.NewDecoder(res.Body).Decode(&certs)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.certs, certs, fmt.Sprintf("%s: expected certs %v got %v", tc.desc, tc.certs, certs))
		}
	}
}

func TestViewCert(t *testing.T) {
	svc, err := newCertService(map[string]string{token: email})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	cs := newCertServer(svc)
	defer cs.Close()

	saved, err := svc.IssueCert(context.Background(), token, addReq.ThingID, addReq.TTL, addReq.KeyBits, addReq.KeyType)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "list cert data",
			id:     saved.Serial,
			auth:   token,
			status: http.StatusOK,
		},
		{
			desc:   "list cert data with invalid token",
			id:     saved.Serial,
			auth:   invalidToken,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list cert data with empty token",
			id:     saved.Serial,
			auth:   "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "list cert data with invalid certs ID",
			id:     wrongCertID,
			auth:   token,
			status: http.StatusInternalServerError,
		},
		{
			desc:   "list cert data with empty certs ID",
			id:     "",
			auth:   token,
			status: http.StatusBadRequest,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: cs.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/certs/%s", cs.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRevokeCert(t *testing.T) {
	svc, err := newCertService(map[string]string{token: email})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	cs := newCertServer(svc)
	defer cs.Close()

	saved, err := svc.IssueCert(context.Background(), token, addReq.ThingID, addReq.TTL, addReq.KeyBits, addReq.KeyType)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "revoke cert with invalid token",
			id:     saved.ThingID,
			auth:   invalidToken,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "revoke cert with empty token",
			id:     saved.ThingID,
			auth:   "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "revoke cert with non-existing thing",
			id:     invalidThingID,
			auth:   token,
			status: http.StatusInternalServerError,
		},
		{
			desc:   "revoke cert",
			id:     saved.ThingID,
			auth:   token,
			status: http.StatusOK,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: cs.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/certs/%s", cs.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
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

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type certsPageRes struct {
	pageRes
	Certs []certsRes `json:"certs"`
}

type certsRes struct {
	Serial string `json:"cert_serial"`
}
