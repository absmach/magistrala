// Copyright (c) Abstract Machines
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
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/bootstrap"
	bsapi "github.com/absmach/magistrala/bootstrap/api"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	validToken   = "validToken"
	invalidToken = "invalid"
	email        = "test@example.com"
	unknown      = "unknown"
	channelsNum  = 3
	contentType  = "application/json"
	wrongID      = "wrong_id"

	addName    = "name"
	addContent = "config"
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey         = []byte("1234567891011121")
	metadata       = map[string]interface{}{"meta": "data"}
	addExternalID  = testsutil.GenerateUUID(&testing.T{})
	addExternalKey = testsutil.GenerateUUID(&testing.T{})
	addThingID     = testsutil.GenerateUUID(&testing.T{})
	addThingKey    = testsutil.GenerateUUID(&testing.T{})
	addReq         = struct {
		ThingID     string   `json:"thing_id"`
		ThingKey    string   `json:"thing_key"`
		ExternalID  string   `json:"external_id"`
		ExternalKey string   `json:"external_key"`
		Channels    []string `json:"channels"`
		Name        string   `json:"name"`
		Content     string   `json:"content"`
	}{
		ThingID:     addThingID,
		ThingKey:    addThingKey,
		ExternalID:  addExternalID,
		ExternalKey: addExternalKey,
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
		Channels:   []string{"1"},
		Content:    "config update",
		State:      1,
		ClientCert: "newcert",
		ClientKey:  "newkey",
		CACert:     "newca",
	}

	missingIDRes  = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingID.Error(), Msg: apiutil.ErrValidation.Error()})
	missingKeyRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerKey.Error(), Msg: apiutil.ErrValidation.Error()})
	bsErrorRes    = toJSON(apiutil.ErrorRes{Err: errors.ErrNotFound.Error(), Msg: bootstrap.ErrBootstrap.Error()})
	extKeyRes     = toJSON(apiutil.ErrorRes{Msg: bootstrap.ErrExternalKey.Error()})
	extSecKeyRes  = toJSON(apiutil.ErrorRes{Err: "encoding/hex: invalid byte: U+002D '-'", Msg: bootstrap.ErrExternalKeySecure.Error()})
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

func newConfig() bootstrap.Config {
	return bootstrap.Config{
		ThingID:     addThingID,
		ThingKey:    addThingKey,
		ExternalID:  addExternalID,
		ExternalKey: addExternalKey,
		Channels: []bootstrap.Channel{
			{
				ID:       "1",
				Metadata: metadata,
			},
		},
		Name:       addName,
		Content:    addContent,
		ClientCert: "newcert",
		ClientKey:  "newkey",
		CACert:     "newca",
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

func newService() (bootstrap.Service, *authmocks.Service, *sdkmocks.SDK) {
	things := mocks.NewConfigsRepository()
	auth := new(authmocks.Service)
	sdk := new(sdkmocks.SDK)

	return bootstrap.New(auth, things, sdk, encKey), auth, sdk
}

func newBootstrapServer(svc bootstrap.Service) *httptest.Server {
	logger := mglog.NewMock()
	mux := bsapi.MakeHandler(svc, bootstrap.NewConfigReader(encKey), logger, instanceID)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestAdd(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	data := toJSON(addReq)

	neID := addReq
	neID.ThingID = testsutil.GenerateUUID(t)
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
			status:      http.StatusConflict,
			location:    "",
		},
		{
			desc:        "add a config with invalid channels",
			req:         wrongData,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusConflict,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: addThingID, Credentials: mgsdk.Credentials{Secret: addThingKey}}, nil)
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
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
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestView(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)
	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	var channels []channel
	for _, ch := range saved.Channels {
		channels = append(channels, channel{ID: ch.ID, Name: ch.Name, Metadata: ch.Metadata})
	}

	data := config{
		ThingID:     saved.ThingID,
		ThingKey:    saved.ThingKey,
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
			id:     saved.ThingID,
			status: http.StatusUnauthorized,
			res:    config{},
		},
		{
			desc:   "view a config",
			auth:   validToken,
			id:     saved.ThingID,
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
			id:     saved.ThingID,
			status: http.StatusUnauthorized,
			res:    config{},
		},
	}

	for _, tc := range cases {
		repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)
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
		repoCall.Unset()
	}
}

func TestUpdate(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

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
			id:          saved.ThingID,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update with an empty token",
			req:         data,
			id:          saved.ThingID,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update a valid config",
			req:         data,
			id:          saved.ThingID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update a config with wrong content type",
			req:         data,
			id:          saved.ThingID,
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
			id:          saved.ThingID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with an empty request",
			id:          saved.ThingID,
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)
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
		repoCall.Unset()
	}
}

func TestUpdateCert(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

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
			id:          saved.ThingID,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update with an empty token",
			req:         data,
			id:          saved.ThingID,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update a valid config",
			req:         data,
			id:          saved.ThingID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update a config with wrong content type",
			req:         data,
			id:          saved.ThingID,
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
			id:          saved.ThingKey,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with an empty request",
			id:          saved.ThingID,
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)
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
		repoCall.Unset()
	}
}

func TestUpdateConnections(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

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
			id:          saved.ThingID,
			auth:        invalidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update connections with an empty token",
			req:         data,
			id:          saved.ThingID,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update connections valid config",
			req:         data,
			id:          saved.ThingID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update connections with wrong content type",
			req:         data,
			id:          saved.ThingID,
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
			id:          saved.ThingID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update a config with invalid request format",
			req:         "}",
			id:          saved.ThingID,
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update a config with an empty request",
			id:          saved.ThingID,
			req:         "",
			auth:        validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
		repoCall3 := auth.On("DeletePolicy", mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
		repoCall4 := auth.On("AddPolicy", mock.Anything, mock.Anything).Return(&magistrala.AddPolicyRes{Authorized: true}, nil)
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
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestList(t *testing.T) {
	configNum := 101
	changedStateNum := 20
	var active, inactive []config
	list := make([]config, configNum)

	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)
	path := fmt.Sprintf("%s/%s", bs.URL, "things/configs")

	c := newConfig()

	for i := 0; i < configNum; i++ {
		c.ExternalID = strconv.Itoa(i)
		c.ThingKey = c.ExternalID
		c.Name = fmt.Sprintf("%s-%d", addName, i)
		c.ExternalKey = fmt.Sprintf("%s%s", addExternalKey, strconv.Itoa(i))

		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
		saved, err := svc.Add(context.Background(), validToken, c)
		assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()

		var channels []channel
		for _, ch := range saved.Channels {
			channels = append(channels, channel{ID: ch.ID, Name: ch.Name, Metadata: ch.Metadata})
		}
		s := config{
			ThingID:     saved.ThingID,
			ThingKey:    saved.ThingKey,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := sdk.On("Connect", mock.Anything, mock.Anything).Return(nil)

		err := svc.ChangeState(context.Background(), validToken, list[i].ThingID, state)
		assert.Nil(t, err, fmt.Sprintf("Changing state expected to succeed: %s.\n", err))

		repoCall.Unset()
		repoCall1.Unset()

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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)

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

		err = json.NewDecoder(res.Body).Decode(&body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.ElementsMatch(t, tc.res.Configs, body.Configs, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res.Configs, body.Configs))
		assert.Equal(t, tc.res.Total, body.Total, fmt.Sprintf("%s: expected response total '%d' got '%d'", tc.desc, tc.res.Total, body.Total))

		repoCall.Unset()
	}
}

func TestRemove(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove with invalid token",
			id:     saved.ThingID,
			auth:   invalidToken,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove with an empty token",
			id:     saved.ThingID,
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
			id:     saved.ThingID,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		req := testRequest{
			client: bs.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/configs/%s", bs.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestBootstrap(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	encExternKey, err := enc([]byte(c.ExternalKey))
	assert.Nil(t, err, fmt.Sprintf("Encrypting config expected to succeed: %s.\n", err))

	var channels []channel
	for _, ch := range saved.Channels {
		channels = append(channels, channel{ID: ch.ID, Name: ch.Name, Metadata: ch.Metadata})
	}

	s := struct {
		ThingID    string    `json:"thing_id"`
		ThingKey   string    `json:"thing_key"`
		Channels   []channel `json:"channels"`
		Content    string    `json:"content"`
		ClientCert string    `json:"client_cert"`
		ClientKey  string    `json:"client_key"`
		CACert     string    `json:"ca_cert"`
	}{
		ThingID:    saved.ThingID,
		ThingKey:   saved.ThingKey,
		Channels:   channels,
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
		repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.externalKey}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		req := testRequest{
			client: bs.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/bootstrap/%s", bs.URL, tc.externalID),
			key:    tc.externalKey,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		body, err := io.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		if tc.secure && tc.status == http.StatusOK {
			body, err = dec(body)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding body: %s", tc.desc, err))
		}

		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res, data))
		repoCall.Unset()
	}
}

func TestChangeState(t *testing.T) {
	svc, auth, sdk := newService()
	bs := newBootstrapServer(svc)

	c := newConfig()

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

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
			id:          saved.ThingID,
			auth:        invalidToken,
			state:       active,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "change state with an empty token",
			id:          saved.ThingID,
			auth:        "",
			state:       active,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "change state with invalid content type",
			id:          saved.ThingID,
			auth:        validToken,
			state:       active,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "change state to active",
			id:          saved.ThingID,
			auth:        validToken,
			state:       active,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "change state to inactive",
			id:          saved.ThingID,
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
			id:          saved.ThingID,
			auth:        validToken,
			state:       fmt.Sprintf("{\"state\": %d}", -3),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "change state with invalid data",
			id:          saved.ThingID,
			auth:        validToken,
			state:       "",
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 = sdk.On("Connect", mock.Anything, mock.Anything).Return(nil)
		repoCall3 := sdk.On("DisconnectThing", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

type channel struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type config struct {
	ThingID     string          `json:"thing_id,omitempty"`
	ThingKey    string          `json:"thing_key,omitempty"`
	Channels    []channel       `json:"channels,omitempty"`
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

func toGroup(ch bootstrap.Channel) mgsdk.Channel {
	return mgsdk.Channel{
		ID:       ch.ID,
		Name:     ch.Name,
		Metadata: ch.Metadata,
	}
}
