// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users"
	httpapi "github.com/mainflux/mainflux/users/api/http"
	"github.com/mainflux/mainflux/users/jwt"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/mainflux/mainflux/users/token"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const (
	contentType  = "application/json"
	invalidEmail = "userexample.com"
	wrongID      = "123e4567-e89b-12d3-a456-000000000042"
	id           = "123e4567-e89b-12d3-a456-000000000001"
)

var user = users.User{Email: "user@example.com", Password: "password"}

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.token != "" {
		req.Header.Set("Authorization", tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	req.Header.Set("Referer", "http://localhost")
	return tr.client.Do(req)
}

func newService() users.Service {
	repo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()
	token := mocks.NewTokenizer()
	email := mocks.NewEmailer()

	return users.New(repo, hasher, idp, email, token)
}

func newServer(svc users.Service) *httptest.Server {
	logger, _ := log.New(os.Stdout, log.Info.String())
	mux := httpapi.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestRegister(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(user)
	invalidData := toJSON(users.User{Email: invalidEmail, Password: "password"})
	invalidFieldData := fmt.Sprintf(`{"email": "%s", "pass": "%s"}`, user.Email, user.Password)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
	}{
		{"register new user", data, contentType, http.StatusCreated},
		{"register existing user", data, contentType, http.StatusConflict},
		{"register user with invalid email address", invalidData, contentType, http.StatusBadRequest},
		{"register user with invalid request format", "{", contentType, http.StatusBadRequest},
		{"register user with empty JSON request", "{}", contentType, http.StatusBadRequest},
		{"register user with empty request", "", contentType, http.StatusBadRequest},
		{"register user with invalid field name", invalidFieldData, contentType, http.StatusBadRequest},
		{"register user with missing content type", data, "", http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	j := jwt.New("secret")
	token, _ := j.TemporaryKey(user.Email)
	tokenData := toJSON(map[string]string{"token": token})
	data := toJSON(user)
	invalidEmailData := toJSON(users.User{
		Email:    invalidEmail,
		Password: "password",
	})
	invalidData := toJSON(users.User{
		Email:    "user@example.com",
		Password: "invalid_password",
	})
	nonexistentData := toJSON(users.User{
		Email:    "non-existentuser@example.com",
		Password: "pass",
	})
	svc.Register(context.Background(), user)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"login with valid credentials", data, contentType, http.StatusCreated, tokenData},
		{"login with invalid credentials", invalidData, contentType, http.StatusForbidden, ""},
		{"login with invalid email address", invalidEmailData, contentType, http.StatusBadRequest, ""},
		{"login non-existent user", nonexistentData, contentType, http.StatusForbidden, ""},
		{"login with invalid request format", "{", contentType, http.StatusBadRequest, ""},
		{"login with empty JSON request", "{}", contentType, http.StatusBadRequest, ""},
		{"login with empty request", "", contentType, http.StatusBadRequest, ""},
		{"login with missing content type", data, "", http.StatusUnsupportedMediaType, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/tokens", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestUserInfo(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	svc.Register(context.Background(), user)
	j := jwt.New("secret")
	token, _ := j.TemporaryKey(user.Email)
	invalidToken, _ := j.TemporaryKey("non-exist@example.com")

	cases := []struct {
		desc   string
		token  string
		status int
		res    string
	}{
		{"user info with valid token", token, http.StatusOK, ""},
		{"user info with invalid token", invalidToken, http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/users", ts.URL),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, "", fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestPasswordResetRequest(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	data := toJSON(user)

	nonexistentData := toJSON(users.User{
		Email:    "non-existentuser@example.com",
		Password: "pass",
	})

	expectedNonExistent := toJSON(struct {
		Msg string `json:"msg"`
	}{
		users.ErrUserNotFound.Error(),
	})

	expectedExisting := toJSON(struct {
		Msg string `json:"msg"`
	}{
		httpapi.MailSent,
	})

	svc.Register(context.Background(), user)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"password reset request with valid email", data, contentType, http.StatusCreated, expectedExisting},
		{"password reset request with invalid email", nonexistentData, contentType, http.StatusCreated, expectedNonExistent},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, ""},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, ""},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, ""},
		{"password reset request with missing content type", data, "", http.StatusUnsupportedMediaType, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/password/reset-request", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestPasswordReset(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	tokenizer := token.New([]byte("secret"), 1)
	resData := struct {
		Msg string `json:"msg"`
	}{
		"",
	}
	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		ConfPass string `json:"confirm_password,omitempty"`
	}{}

	expectedSuccess := toJSON(resData)

	resData.Msg = users.ErrUserNotFound.Error()
	expectedNonExUser := toJSON(resData)

	svc.Register(context.Background(), user)
	tok, _ := tokenizer.Generate(user.Email, 0)

	reqData.Password = user.Password
	reqData.ConfPass = user.Password
	reqData.Token = tok
	reqExisting := toJSON(reqData)

	reqData.Token, _ = tokenizer.Generate("non-existentuser@example.com", 0)

	reqNoExist := toJSON(reqData)
	reqData.Token, _ = tokenizer.Generate(user.Email, -5)

	reqData.ConfPass = "wrong"
	reqPassNoMatch := toJSON(reqData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password reset with valid token", reqExisting, contentType, http.StatusCreated, expectedSuccess, tok},
		{"password reset with invalid token", reqNoExist, contentType, http.StatusCreated, expectedNonExUser, tok},
		{"password reset with confirm password not matching", reqPassNoMatch, contentType, http.StatusBadRequest, "", tok},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, "", tok},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, "", tok},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, "", tok},
		{"password reset request with missing content type", reqExisting, "", http.StatusUnsupportedMediaType, "", tok},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/password/reset", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestPasswordChange(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	j := jwt.New("secret")
	resData := struct {
		Msg string `json:"msg"`
	}{
		"",
	}
	expectedSuccess := toJSON(resData)

	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		OldPassw string `json:"old_password,omitempty"`
	}{}
	resData.Msg = users.ErrUnauthorizedAccess.Error()
	expectedNonExUser := toJSON(resData)

	svc.Register(context.Background(), user)
	tok, _ := j.TemporaryKey(user.Email)
	tokNoUser, _ := j.TemporaryKey("non-existentuser@example.com")

	reqData.Password = user.Password
	reqData.OldPassw = user.Password
	reqData.Token = tok
	dataResExisting := toJSON(reqData)

	reqData.Token, _ = j.TemporaryKey(user.Email)

	reqNoExist := toJSON(reqData)
	reqData.Token, _ = j.TemporaryKey(user.Email)

	reqData.OldPassw = "wrong"
	reqWrongPass := toJSON(reqData)

	resData.Msg = users.ErrUnauthorizedAccess.Error()
	expWronPassRes := toJSON(resData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password change with valid token", dataResExisting, contentType, http.StatusCreated, expectedSuccess, tok},
		{"password change with invalid token", reqNoExist, contentType, http.StatusCreated, expectedNonExUser, tokNoUser},
		{"password change with invalid old password", reqWrongPass, contentType, http.StatusCreated, expWronPassRes, tok},
		{"password change with empty JSON request", "{}", contentType, http.StatusBadRequest, "", tok},
		{"password change empty request", "", contentType, http.StatusBadRequest, "", tok},
		{"password change missing content type", dataResExisting, "", http.StatusUnsupportedMediaType, "", tok},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/password", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
			token:       tc.tok,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}
