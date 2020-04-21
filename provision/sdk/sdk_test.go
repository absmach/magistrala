package sdk_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/errors"
	provsdk "github.com/mainflux/mainflux/provision/sdk"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/stretchr/testify/assert"
)

const (
	contentType = "application/json"
	invalid     = "invalid"
	exists      = "exists"
	valid       = "valid"
)

type handler func(http.ResponseWriter, *http.Request)

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h(rw, r)
}

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

type certReq struct {
	ThingID  string `json:"id,omitempty"`
	ThingKey string `json:"key,omitempty"`
}

func auth(rw http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Authorization") != valid {
		rw.WriteHeader(http.StatusForbidden)
		return false
	}
	return true
}

func ct(rw http.ResponseWriter, r *http.Request) bool {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		rw.WriteHeader(http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

func delete(rw http.ResponseWriter, r *http.Request) {
	if !auth(rw, r) {
		return
	}
	id := bone.GetValue(r, "id")
	if id == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}

func createToken(rw http.ResponseWriter, r *http.Request) {
	u := mfsdk.User{}
	if !ct(rw, r) {
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if u.Email == invalid || u.Password == invalid {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusCreated)
	data, _ := json.Marshal(tokenRes{valid})
	rw.Write(data)
}

func createThing(rw http.ResponseWriter, r *http.Request) {
	t := mfsdk.Thing{}
	if !ct(rw, r) || !auth(rw, r) {
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if t.Name == invalid {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusCreated)
	rw.Header().Add("Location", t.ID)
}

func thing(rw http.ResponseWriter, r *http.Request) {
	if !auth(rw, r) {
		return
	}
	id := bone.GetValue(r, "id")
	if id == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if id != exists {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	data, _ := json.Marshal(mfsdk.Thing{})
	rw.WriteHeader(http.StatusOK)
	rw.Write(data)
}

func createChannel(rw http.ResponseWriter, r *http.Request) {
	c := mfsdk.Channel{}
	if !ct(rw, r) || !auth(rw, r) {
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if c.Name == invalid {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusCreated)
	rw.Header().Add("Location", c.ID)
}

func connect(rw http.ResponseWriter, r *http.Request) {
	if !auth(rw, r) {
		return
	}
	var conn mfsdk.ConnectionIDs
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(conn.ChannelIDs) == 0 || len(conn.ThingIDs) == 0 {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if conn.ChannelIDs[0] != exists || conn.ThingIDs[0] != exists {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func cert(rw http.ResponseWriter, r *http.Request) {
	if !auth(rw, r) || !ct(rw, r) {
		return
	}
	crt := certReq{}
	if err := json.NewDecoder(r.Body).Decode(&crt); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if crt.ThingID == "" || crt.ThingKey == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if crt.ThingID == exists || crt.ThingKey == exists {
		rw.WriteHeader(http.StatusConflict)
		return
	}

	data, _ := json.Marshal(provsdk.Cert{})
	rw.WriteHeader(http.StatusCreated)
	rw.Write(data)
}

func saveToBootstrap(rw http.ResponseWriter, r *http.Request) {
	if !auth(rw, r) || !ct(rw, r) {
		return
	}
	cfg := provsdk.BSConfig{}
	json.NewDecoder(r.Body).Decode(&cfg)
	defer r.Body.Close()

	if cfg.ThingID == exists {
		rw.WriteHeader(http.StatusConflict)
		return
	}
	if cfg.Channels[0] == invalid || cfg.Channels[1] == invalid {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusCreated)
}

func whitelist(rw http.ResponseWriter, r *http.Request) {
	if !auth(rw, r) || !ct(rw, r) {
		return
	}
	id := bone.GetValue(r, "id")
	if id == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if id != exists {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	d := make(map[string]int)
	json.NewDecoder(r.Body).Decode(&d)
	defer r.Body.Close()

	if s, ok := d["state"]; ok {
		if s != 0 && s != 1 {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusOK)
		return
	}
	rw.WriteHeader(http.StatusBadRequest)
}

func removeCert(rw http.ResponseWriter, r *http.Request) {}

func newSDK() provsdk.SDK {
	r := bone.New()
	r.Post("/tokens", handler(createToken))
	r.Post("/things", handler(createThing))
	r.Get("/things/:id", handler(thing))
	r.Delete("/things/:id", handler(delete))
	r.Post("/channels", handler(createChannel))
	r.Delete("/channels/:id", handler(delete))
	r.Post("/connect", handler(connect))
	r.Post("/certs", handler(cert))
	r.Post("/things/configs", handler(saveToBootstrap))
	r.Put("/things/state/:id", handler(whitelist))
	r.Delete("/things/configs/:id", handler(delete))
	r.Delete("/certs/:id", handler(delete))
	svc := httptest.NewServer(r)
	crt := fmt.Sprintf("%s/certs", svc.URL)
	bs := fmt.Sprintf("%s/things/configs", svc.URL)
	whl := fmt.Sprintf("%s/things/state", svc.URL)

	thingSdkCfg := mfsdk.Config{
		BaseURL:         svc.URL,
		MsgContentType:  "application/json",
		TLSVerification: false,
	}
	things := mfsdk.NewSDK(thingSdkCfg)

	userSdkCfg := mfsdk.Config{
		BaseURL:         svc.URL,
		MsgContentType:  "application/json",
		TLSVerification: false,
	}
	users := mfsdk.NewSDK(userSdkCfg)
	return provsdk.New(crt, bs, whl, things, users)
}

func TestCreateToken(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		email string
		pass  string
		err   error
	}{
		{
			desc:  "Create token successfully",
			email: "test@email.com",
			pass:  valid,
			err:   nil,
		},
		{
			desc:  "Create an invalid token",
			email: invalid,
			pass:  valid,
			err:   mfsdk.ErrFailedCreation,
		},
	}
	for _, tc := range cases {
		_, err := sdk.CreateToken(tc.email, tc.pass)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, err, tc.err))
	}
}

func TestCreateThing(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		token string
		err   error
	}{
		{
			desc:  "Create thing successfully",
			token: valid,
			err:   nil,
		},
		{
			desc:  "Create thing unauthorized",
			token: invalid,
			err:   status(http.StatusForbidden),
		},
	}
	for _, tc := range cases {
		_, err := sdk.CreateThing("external", "name", tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, err, tc.err))
	}
}

func TestThing(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc    string
		thingID string
		token   string
		err     error
	}{
		{
			desc:    "Fetch thing successfully",
			thingID: exists,
			token:   valid,
			err:     nil,
		},
		{
			desc:    "Fetch non existent thing",
			thingID: invalid,
			token:   valid,
			err:     mfsdk.ErrFailedFetch,
		},
		{
			desc:    "Fetch thing wrong id",
			thingID: "",
			token:   valid,
			err:     mfsdk.ErrFailedFetch,
		},
		{
			desc:    "Fetch thing unauthorized",
			thingID: exists,
			token:   invalid,
			err:     status(http.StatusForbidden),
		},
	}
	for _, tc := range cases {
		_, err := sdk.Thing(tc.thingID, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, err, tc.err))
	}
}

func TestDeleteThing(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "Delete thing successfully",
			id:    valid,
			token: valid,
			err:   nil,
		},
		{
			desc:  "Delete thing unauthorized",
			id:    valid,
			token: invalid,
			err:   status(http.StatusForbidden),
		},
		{
			desc:  "Delete thing wrong ID",
			id:    "",
			token: valid,
			err:   mfsdk.ErrFailedRemoval,
		},
	}
	for _, tc := range cases {
		err := sdk.DeleteThing(tc.id, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, err, tc.err))
	}
}

func TestCreateChannel(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		token string
		err   error
	}{
		{
			desc:  "Create channel successfully",
			token: valid,
			err:   nil,
		},
		{
			desc:  "Create channel unauthorized",
			token: invalid,
			err:   mfsdk.ErrFailedCreation,
		},
	}
	for _, tc := range cases {
		_, err := sdk.CreateChannel("external", "ctrl", tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, err, tc.err))
	}
}

func TestDeleteChannel(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "Delete channel successfully",
			id:    valid,
			token: valid,
			err:   nil,
		},
		{
			desc:  "Delete channel unauthorized",
			id:    valid,
			token: invalid,
			err:   status(http.StatusForbidden),
		},
		{
			desc:  "Delete channel wrong ID",
			id:    "",
			token: valid,
			err:   mfsdk.ErrFailedRemoval,
		},
	}
	for _, tc := range cases {
		err := sdk.DeleteChannel(tc.id, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc      string
		thingID   string
		channelID string
		token     string
		err       error
	}{
		{
			desc:      "Connect successfully",
			thingID:   exists,
			channelID: exists,
			token:     valid,
			err:       nil,
		},
		{
			desc:      "Connect unauthorized",
			thingID:   exists,
			channelID: exists,
			token:     invalid,
			err:       status(http.StatusForbidden),
		},
		{
			desc:      "Connect bad data",
			thingID:   "",
			channelID: exists,
			token:     valid,
			err:       mfsdk.ErrFailedConnect,
		},
		{
			desc:      "Connect non existent data",
			thingID:   valid,
			channelID: exists,
			token:     valid,
			err:       mfsdk.ErrFailedConnect,
		},
	}
	for _, tc := range cases {
		err := sdk.Connect(tc.thingID, tc.channelID, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCert(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		id    string
		key   string
		token string
		err   error
	}{
		{
			desc:  "Create cert successfully",
			id:    valid,
			key:   valid,
			token: valid,
			err:   nil,
		},
		{
			desc:  "Create cert unauthorized",
			id:    valid,
			key:   valid,
			token: invalid,
			err:   provsdk.ErrCerts,
		},
		{
			desc:  "Create cert with an existing id",
			id:    exists,
			key:   valid,
			token: valid,
			err:   provsdk.ErrCerts,
		},
		{
			desc:  "Create cert with an existing key",
			id:    valid,
			key:   exists,
			token: valid,
			err:   provsdk.ErrCerts,
		},
	}
	for _, tc := range cases {
		_, err := sdk.Cert(tc.id, tc.key, tc.token)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestBootstrap(t *testing.T) {
	sdk := newSDK()

	cfg1 := provsdk.BSConfig{
		ThingID:     valid,
		ExternalID:  valid,
		ExternalKey: valid,
		Channels:    []string{valid, valid},
		Content:     valid,
		ClientCert:  valid,
		ClientKey:   valid,
		CACert:      valid,
	}

	cfg2 := cfg1
	cfg2.Channels = []string{invalid, valid}

	cfg3 := cfg1
	cfg3.ThingID = exists

	cases := []struct {
		desc   string
		config provsdk.BSConfig
		token  string
		err    error
	}{
		{
			desc:   "Save config successfully",
			config: cfg1,
			token:  valid,
			err:    nil,
		},
		{
			desc:   "Save config unauthorized",
			config: cfg1,
			token:  invalid,
			err:    provsdk.ErrUnauthorized,
		},
		{
			desc:   "Save malformed config",
			config: cfg2,
			token:  valid,
			err:    provsdk.ErrMalformedEntity,
		},
		{
			desc:   "Save existing config",
			config: cfg3,
			token:  valid,
			err:    provsdk.ErrConflict,
		},
	}
	for _, tc := range cases {
		err := sdk.SaveConfig(tc.config, tc.token)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestWhitelist(t *testing.T) {
	sdk := newSDK()

	sValid := map[string]int{"state": 1}
	sInvalid := map[string]int{"state": 42}

	cases := []struct {
		desc  string
		id    string
		state map[string]int
		token string
		err   error
	}{
		{
			desc:  "Whitelist successfully",
			id:    exists,
			state: sValid,
			token: valid,
			err:   nil,
		},
		{
			desc:  "Whitelist unauthorized",
			id:    exists,
			state: sValid,
			token: invalid,
			err:   provsdk.ErrUnauthorized,
		},
		{
			desc:  "Whitelist invalid state",
			id:    exists,
			state: sInvalid,
			token: valid,
			err:   provsdk.ErrMalformedEntity,
		},
		{
			desc:  "Whitelist not found",
			id:    valid,
			state: sValid,
			token: valid,
			err:   provsdk.ErrNotFound,
		},
	}
	for _, tc := range cases {
		err := sdk.Whitelist(tc.id, tc.state, tc.token)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveBootstrap(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "Delete config successfully",
			id:    valid,
			token: valid,
			err:   nil,
		},
		{
			desc:  "Delete config unauthorized",
			id:    valid,
			token: invalid,
			err:   provsdk.ErrUnauthorized,
		},
		{
			desc:  "Delete config wrong ID",
			id:    "",
			token: valid,
			err:   provsdk.ErrConfigRemove,
		},
	}
	for _, tc := range cases {
		err := sdk.RemoveConfig(tc.id, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveCert(t *testing.T) {
	sdk := newSDK()

	cases := []struct {
		desc  string
		key   string
		token string
		err   error
	}{
		{
			desc:  "Delete cert successfully",
			key:   valid,
			token: valid,
			err:   nil,
		},
		{
			desc:  "Delete cert unauthorized",
			key:   valid,
			token: invalid,
			err:   provsdk.ErrUnauthorized,
		},
		{
			desc:  "Delete cert wrong ID",
			key:   "",
			token: valid,
			err:   provsdk.ErrCertsRemove,
		},
	}
	for _, tc := range cases {
		err := sdk.RemoveCert(tc.key, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func status(status int) error {
	return errors.New(fmt.Sprintf("%d %s", status, http.StatusText(status)))
}
