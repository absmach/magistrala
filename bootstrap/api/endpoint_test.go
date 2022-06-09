// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	bsapi "github.com/mainflux/mainflux/bootstrap/api"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things"
	thingsapi "github.com/mainflux/mainflux/things/api/things/http"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validToken     = "validToken"
	invalidToken   = "invalidToken"
	email          = "test@example.com"
	unknown        = "unknown"
	channelsNum    = 3
	contentType    = "application/json"
	wrongID        = "wrong_id"
	addExternalID  = "external-id"
	addExternalKey = "external-key"
	addName        = "name"
	addContent     = "config"
)

var (
	encKey   = []byte("1234567891011121")
	metadata = map[string]interface{}{"meta": "data"}
	addReq   = struct {
		ThingID     string   `json:"thing_id"`
		ExternalID  string   `json:"external_id"`
		ExternalKey string   `json:"external_key"`
		Channels    []string `json:"channels"`
		Name        string   `json:"name"`
		Content     string   `json:"content"`
	}{
		ExternalID:  "external-id",
		ExternalKey: "external-key",
		Channels:    []string{"1"},
		Name:        "name",
		Content:     "config",
	}

	updateReq = struct {
		Channels   []string        `json:"channels,omitempty"`
		Content    string          `json:"content,omitempty"`
		State      bootstrap.State `json:"state,omitempty"`
		ClientCert string          `json:"client_cert,omitempty"`
		ClientKey  string          `json:"client_key,omitempty"`
		CACert     string          `json:"ca_cert,omitempty"`
	}{
		Channels:   []string{"2", "3"},
		Content:    "config update",
		State:      1,
		ClientCert: "newcert",
		ClientKey:  "newkey",
		CACert:     "newca",
	}

	bsErrorRes    = toJSON(apiutil.ErrorRes{Err: bootstrap.ErrBootstrap.Error()})
	extKeyRes     = toJSON(apiutil.ErrorRes{Err: bootstrap.ErrExternalKey.Error()})
	extSecKeyRes  = toJSON(apiutil.ErrorRes{Err: bootstrap.ErrExternalKeySecure.Error()})
	missingIDRes  = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingID.Error()})
	missingKeyRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerKey.Error()})
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	key         string
	body        io.Reader
}

func newConfig(channels []bootstrap.Channel) bootstrap.Config {
	return bootstrap.Config{
		ExternalID:  addExternalID,
		ExternalKey: addExternalKey,
		MFChannels:  channels,
		Name:        addName,
		Content:     addContent,
		ClientCert:  "newcert",
		ClientKey:   "newkey",
		CACert:      "newca",
	}
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)

	if err != nil {
		return nil, err
	}

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.key != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.key)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	return tr.client.Do(req)
}

func enc(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return ciphertext, nil
}

func dec(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	if len(in) < aes.BlockSize {
		return nil, errors.ErrMalformedEntity
	}
	iv := in[:aes.BlockSize]
	in = in[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(in, in)
	return in, nil
}

func newService(auth mainflux.AuthServiceClient, url string) bootstrap.Service {
	things := mocks.NewConfigsRepository()
	config := mfsdk.Config{
		ThingsURL: url,
	}

	sdk := mfsdk.NewSDK(config)
	return bootstrap.New(auth, things, sdk, encKey)
}

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

func newThingsService(auth mainflux.AuthServiceClient) things.Service {
	return mocks.NewThingsService(map[string]things.Thing{}, generateChannels(), auth)
}

func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := thingsapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func newBootstrapServer(svc bootstrap.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := bsapi.MakeHandler(svc, bootstrap.NewConfigReader(encKey), logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestAdd(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	data := toJSON(addReq)

	neID := addReq
	neID.ThingID = "non-existent"
	neData := toJSON(neID)

	invalidChannels := addReq
	invalidChannels.Channels = []string{wrongID}
	wrongData := toJSON(invalidChannels)

	cases := []struct {
		desc        string
		req         string
		auth        string
		contentType string
		status      int
		location    string
	}{
		{
			desc:        "add a config with invalid token",
			req:         data,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "add a valid config",
			req:         data,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusCreated,
			location:    "/things/configs/1",
		},
		{
			desc:        "add a config with wring content type",
			req:         data,
			auth:        validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
		{
			desc:        "add an existing config",
			req:         data,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusConflict,
			location:    "",
		},
		{
			desc:        "add a config with non-existent ID",
			req:         neData,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
			location:    "",
		},
		{
			desc:        "add a config with invalid channels",
			req:         wrongData,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add a config with wrong JSON",
			req:         "{\"external_id\": 5}",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "add a config with invalid request format",
			req:         "}",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add a config with empty JSON",
			req:         "{}",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add a config with an empty request",
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      bs.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/configs", bs.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location '%s' got '%s'", tc.desc, tc.location, location))
	}
}

func TestView(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	c := newConfig([]bootstrap.Channel{})

	mfChs := generateChannels()
	for id, ch := range mfChs {
		c.MFChannels = append(c.MFChannels, bootstrap.Channel{
			ID:       ch.ID,
			Name:     fmt.Sprintf("%s%s", "name ", id),
			Metadata: map[string]interface{}{"type": fmt.Sprintf("some type %s", id)},
		})
	}

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	var channels []channel
	for _, ch := range saved.MFChannels {
		channels = append(channels, channel{ID: ch.ID, Name: ch.Name, Metadata: ch.Metadata})
	}

	data := config{
		MFThing:     saved.MFThing,
		MFKey:       saved.MFKey,
		State:       saved.State,
		Channels:    channels,
		ExternalID:  saved.ExternalID,
		ExternalKey: saved.ExternalKey,
		Name:        saved.Name,
		Content:     saved.Content,
	}

	cases := []struct {
		desc   string
		auth   string
		id     string
		status int
		res    config
	}{
		{
			desc:   "view a config with invalid token",
			auth:   invalidToken,
			id:     saved.MFThing,
			status: http.StatusUnauthorized,
			res:    config{},
		},
		{
			desc:   "view a config",
			auth:   validToken,
			id:     saved.MFThing,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view a non-existing config",
			auth:   validToken,
			id:     wrongID,
			status: http.StatusNotFound,
			res:    config{},
		},
		{
			desc:   "view a config with an empty token",
			auth:   "",
			id:     saved.MFThing,
			status: http.StatusUnauthorized,
			res:    config{},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: bs.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/configs/%s", bs.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		var view config
		if err := json.NewDecoder(res.Body).Decode(&view); err != io.EOF {
			assert.Nil(t, err, fmt.Sprintf("Decoding expected to succeed %s: %s", tc.desc, err))
		}

		assert.ElementsMatch(t, tc.res.Channels, view.Channels, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res.Channels, view.Channels))
		// Empty channels to prevent order mismatch.
		tc.res.Channels = []channel{}
		view.Channels = []channel{}
		assert.Equal(t, tc.res, view, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res, view))
	}
}

func TestUpdate(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	data := toJSON(updateReq)

	cases := []struct {
		desc        string
		req         string
		id          string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "update with invalid token",
			req:         data,
			id:          saved.MFThing,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update with an empty token",
			req:         data,
			id:          saved.MFThing,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update a valid config",
			req:         data,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update a config with wrong content type",
			req:         data,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update a non-existing config",
			req:         data,
			id:          wrongID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update a config with invalid request format",
			req:         "}",
			id:          saved.MFThing,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with an empty request",
			id:          saved.MFThing,
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      bs.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/configs/%s", bs.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
func TestUpdateCert(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	data := toJSON(updateReq)

	cases := []struct {
		desc        string
		req         string
		id          string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "update with invalid token",
			req:         data,
			id:          saved.MFThing,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update with an empty token",
			req:         data,
			id:          saved.MFThing,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update a valid config",
			req:         data,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update a config with wrong content type",
			req:         data,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update a non-existing config",
			req:         data,
			id:          wrongID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update a config with invalid request format",
			req:         "}",
			id:          saved.MFKey,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with an empty request",
			id:          saved.MFThing,
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      bs.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/configs/certs/%s", bs.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUpdateConnections(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	data := toJSON(updateReq)

	invalidChannels := updateReq
	invalidChannels.Channels = []string{wrongID}

	wrongData := toJSON(invalidChannels)

	cases := []struct {
		desc        string
		req         string
		id          string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "update connections with invalid token",
			req:         data,
			id:          saved.MFThing,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update connections with an empty token",
			req:         data,
			id:          saved.MFThing,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update connections valid config",
			req:         data,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update connections with wrong content type",
			req:         data,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update connections for a non-existing config",
			req:         data,
			id:          wrongID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update connections with invalid channels",
			req:         wrongData,
			id:          saved.MFThing,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with invalid request format",
			req:         "}",
			id:          saved.MFThing,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with an empty request",
			id:          saved.MFThing,
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      bs.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/configs/connections/%s", bs.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestList(t *testing.T) {
	configNum := 101
	changedStateNum := 20
	var active, inactive []config
	list := make([]config, configNum)

	auth := mocks.NewAuthClient(map[string]string{validToken: email})
	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)
	path := fmt.Sprintf("%s/%s", bs.URL, "things/configs")

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	for i := 0; i < configNum; i++ {
		c.ExternalID = strconv.Itoa(i)
		c.MFKey = c.ExternalID
		c.Name = fmt.Sprintf("%s-%d", addName, i)
		c.ExternalKey = fmt.Sprintf("%s%s", addExternalKey, strconv.Itoa(i))

		saved, err := svc.Add(context.Background(), validToken, c)
		require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

		var channels []channel
		for _, ch := range saved.MFChannels {
			channels = append(channels, channel{ID: ch.ID, Name: ch.Name, Metadata: ch.Metadata})
		}
		s := config{
			MFThing:     saved.MFThing,
			MFKey:       saved.MFKey,
			Channels:    channels,
			ExternalID:  saved.ExternalID,
			ExternalKey: saved.ExternalKey,
			Name:        saved.Name,
			Content:     saved.Content,
			State:       saved.State,
		}
		list[i] = s
	}

	// Change state of first 20 elements for filtering tests.
	for i := 0; i < changedStateNum; i++ {
		state := bootstrap.Active
		if i%2 == 0 {
			state = bootstrap.Inactive
		}
		err := svc.ChangeState(context.Background(), validToken, list[i].MFThing, state)
		require.Nil(t, err, fmt.Sprintf("Changing state expected to succeed: %s.\n", err))
		list[i].State = state
		if state == bootstrap.Inactive {
			inactive = append(inactive, list[i])
			continue
		}
		active = append(active, list[i])
	}

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
		res    configPage
	}{
		{
			desc:   "view list with invalid token",
			auth:   invalidToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 10),
			status: http.StatusUnauthorized,
			res:    configPage{},
		},
		{
			desc:   "view list with an empty token",
			auth:   "",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 10),
			status: http.StatusUnauthorized,
			res:    configPage{},
		},
		{
			desc:   "view list",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 1),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  0,
				Limit:   1,
				Configs: list[0:1],
			},
		},
		{
			desc:   "view list searching by name",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", path, 0, 100, "95"),
			status: http.StatusOK,
			res: configPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: list[95:96],
			},
		},
		{
			desc:   "view last page",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 100, 10),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  100,
				Limit:   10,
				Configs: list[100:],
			},
		},
		{
			desc:   "view with limit greater than allowed",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 1000),
			status: http.StatusBadRequest,
			res:    configPage{},
		},
		{
			desc:   "view list with no specified limit and offset",
			auth:   validToken,
			url:    path,
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  0,
				Limit:   10,
				Configs: list[0:10],
			},
		},
		{
			desc:   "view list with no specified limit",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d", path, 10),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  10,
				Limit:   10,
				Configs: list[10:20],
			},
		},
		{
			desc:   "view list with no specified offset",
			auth:   validToken,
			url:    fmt.Sprintf("%s?limit=%d", path, 10),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  0,
				Limit:   10,
				Configs: list[0:10],
			},
		},
		{
			desc:   "view list with limit < 0",
			auth:   validToken,
			url:    fmt.Sprintf("%s?limit=%d", path, -10),
			status: http.StatusBadRequest,
			res:    configPage{},
		},
		{
			desc:   "view list with offset < 0",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d", path, -10),
			status: http.StatusBadRequest,
			res:    configPage{},
		},
		{
			desc:   "view list with invalid query parameters",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&state=%d&key=%%", path, 10, 10, bootstrap.Inactive),
			status: http.StatusBadRequest,
			res:    configPage{},
		},
		{
			desc:   "view first 10 active",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&state=%d", path, 0, 20, bootstrap.Active),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(active)),
				Offset:  0,
				Limit:   20,
				Configs: active,
			},
		},
		{
			desc:   "view first 10 inactive",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&state=%d", path, 0, 20, bootstrap.Inactive),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list) - len(inactive)),
				Offset:  0,
				Limit:   20,
				Configs: inactive,
			},
		},
		{
			desc:   "view first 5 active",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&state=%d", path, 0, 10, bootstrap.Active),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(active)),
				Offset:  0,
				Limit:   10,
				Configs: active[:5],
			},
		},
		{
			desc:   "view last 5 inactive",
			auth:   validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&state=%d", path, 10, 10, bootstrap.Inactive),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list) - len(active)),
				Offset:  10,
				Limit:   10,
				Configs: inactive[5:],
			},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: bs.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		var body configPage

		json.NewDecoder(res.Body).Decode(&body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res.Configs, body.Configs, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res.Configs, body.Configs))
		assert.Equal(t, tc.res.Total, body.Total, fmt.Sprintf("%s: expected response total '%d' got '%d'", tc.desc, tc.res.Total, body.Total))
	}
}

func TestRemove(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove with invalid token",
			id:     saved.MFThing,
			auth:   invalidToken,
			status: http.StatusUnauthorized,
		}, {
			desc:   "remove with an empty token",
			id:     saved.MFThing,
			auth:   "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove non-existing config",
			id:     "non-existing",
			auth:   validToken,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove config",
			id:     saved.MFThing,
			auth:   validToken,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed config",
			id:     wrongID,
			auth:   validToken,
			status: http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: bs.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/configs/%s", bs.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestBootstrap(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	encExternKey, err := enc([]byte(c.ExternalKey))
	require.Nil(t, err, fmt.Sprintf("Encrypting config expected to succeed: %s.\n", err))

	var channels []channel
	for _, ch := range saved.MFChannels {
		channels = append(channels, channel{ID: ch.ID, Name: ch.Name, Metadata: ch.Metadata})
	}

	s := struct {
		MFThing    string    `json:"mainflux_id"`
		MFKey      string    `json:"mainflux_key"`
		MFChannels []channel `json:"mainflux_channels"`
		Content    string    `json:"content"`
		ClientCert string    `json:"client_cert"`
		ClientKey  string    `json:"client_key"`
		CACert     string    `json:"ca_cert"`
	}{
		MFThing:    saved.MFThing,
		MFKey:      saved.MFKey,
		MFChannels: channels,
		Content:    saved.Content,
		ClientCert: saved.ClientCert,
		ClientKey:  saved.ClientKey,
		CACert:     saved.CACert,
	}

	data := toJSON(s)

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		status      int
		res         string
		secure      bool
	}{
		{
			desc:        "bootstrap a Thing with unknown ID",
			externalID:  unknown,
			externalKey: c.ExternalKey,
			status:      http.StatusNotFound,
			res:         bsErrorRes,
			secure:      false,
		},
		{
			desc:        "bootstrap a Thing with an empty ID",
			externalID:  "",
			externalKey: c.ExternalKey,
			status:      http.StatusBadRequest,
			res:         missingIDRes,
			secure:      false,
		},
		{
			desc:        "bootstrap a Thing with unknown key",
			externalID:  c.ExternalID,
			externalKey: unknown,
			status:      http.StatusForbidden,
			res:         extKeyRes,
			secure:      false,
		},
		{
			desc:        "bootstrap a Thing with an empty key",
			externalID:  c.ExternalID,
			externalKey: "",
			status:      http.StatusUnauthorized,
			res:         missingKeyRes,
			secure:      false,
		},
		{
			desc:        "bootstrap known Thing",
			externalID:  c.ExternalID,
			externalKey: c.ExternalKey,
			status:      http.StatusOK,
			res:         data,
			secure:      false,
		},
		{
			desc:        "bootstrap secure",
			externalID:  fmt.Sprintf("secure/%s", c.ExternalID),
			externalKey: hex.EncodeToString(encExternKey),
			status:      http.StatusOK,
			res:         data,
			secure:      true,
		},
		{
			desc:        "bootstrap secure with unencrypted key",
			externalID:  fmt.Sprintf("secure/%s", c.ExternalID),
			externalKey: c.ExternalKey,
			status:      http.StatusForbidden,
			res:         extSecKeyRes,
			secure:      true,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: bs.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/bootstrap/%s", bs.URL, tc.externalID),
			key:    tc.externalKey,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		if tc.secure && tc.status == http.StatusOK {
			body, err = dec(body)
		}

		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res, data))
	}
}

func TestChangeState(t *testing.T) {
	auth := mocks.NewAuthClient(map[string]string{validToken: email})

	ts := newThingsServer(newThingsService(auth))
	svc := newService(auth, ts.URL)
	bs := newBootstrapServer(svc)

	c := newConfig([]bootstrap.Channel{{ID: "1"}})

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	inactive := fmt.Sprintf("{\"state\": %d}", bootstrap.Inactive)
	active := fmt.Sprintf("{\"state\": %d}", bootstrap.Active)

	cases := []struct {
		desc        string
		id          string
		auth        string
		state       string
		contentType string
		status      int
	}{
		{
			desc:        "change state with invalid token",
			id:          saved.MFThing,
			auth:        invalidToken,
			state:       active,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "change state with an empty token",
			id:          saved.MFThing,
			auth:        "",
			state:       active,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "change state with invalid content type",
			id:          saved.MFThing,
			auth:        validToken,
			state:       active,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "change state to active",
			id:          saved.MFThing,
			auth:        validToken,
			state:       active,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "change state to inactive",
			id:          saved.MFThing,
			auth:        validToken,
			state:       inactive,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "change state of non-existing config",
			id:          wrongID,
			auth:        validToken,
			state:       active,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "change state to invalid value",
			id:          saved.MFThing,
			auth:        validToken,
			state:       fmt.Sprintf("{\"state\": %d}", -3),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "change state with invalid data",
			id:          saved.MFThing,
			auth:        validToken,
			state:       "",
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      bs.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/state/%s", bs.URL, tc.id),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(tc.state),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type channel struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type config struct {
	MFThing     string          `json:"mainflux_id,omitempty"`
	MFKey       string          `json:"mainflux_key,omitempty"`
	Channels    []channel       `json:"mainflux_channels,omitempty"`
	ExternalID  string          `json:"external_id"`
	ExternalKey string          `json:"external_key,omitempty"`
	Content     string          `json:"content,omitempty"`
	Name        string          `json:"name"`
	State       bootstrap.State `json:"state"`
}

type configPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Configs []config `json:"configs"`
}
