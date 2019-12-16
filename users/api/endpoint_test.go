// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

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

	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/mocks"
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
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	email := mocks.NewEmailer()

	return users.New(repo, hasher, auth, email)
}

func newServer(svc users.Service) *httptest.Server {
	logger, _ := log.New(os.Stdout, log.Info.String())
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
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
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Issuer: user.Email, Type: 0})
	token := tkn.GetValue()
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
		{"login with invalid credentials", invalidData, contentType, http.StatusForbidden, toJSON(errorRes{users.ErrUnauthorizedAccess.Error()})},
		{"login with invalid email address", invalidEmailData, contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()})},
		{"login non-existent user", nonexistentData, contentType, http.StatusForbidden, toJSON(errorRes{users.ErrUnauthorizedAccess.Error()})},
		{"login with invalid request format", "{", contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()})},
		{"login with empty JSON request", "{}", contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()})},
		{"login with empty request", "", contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()})},
		{"login with missing content type", data, "", http.StatusUnsupportedMediaType, toJSON(errorRes{api.ErrUnsupportedContentType.Error()})},
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

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Issuer: user.Email, Type: 0})
	token := tkn.GetValue()
	cases := []struct {
		desc   string
		token  string
		status int
		res    string
	}{
		{"user info with valid token", token, http.StatusOK, ""},
		{"user info with invalid token", "", http.StatusForbidden, ""},
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

	expectedExisting := toJSON(struct {
		Msg string `json:"msg"`
	}{
		api.MailSent,
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
		{"password reset request with invalid email", nonexistentData, contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrUserNotFound.Error()})},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, toJSON(errorRes{api.ErrFailedDecode.Error()})},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()})},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, toJSON(errorRes{api.ErrFailedDecode.Error()})},
		{"password reset request with missing content type", data, "", http.StatusUnsupportedMediaType, toJSON(errorRes{api.ErrUnsupportedContentType.Error()})},
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

	svc.Register(context.Background(), user)
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Issuer: user.Email, Type: 0})
	token := tkn.GetValue()

	reqData.Password = user.Password
	reqData.ConfPass = user.Password
	reqData.Token = token
	reqExisting := toJSON(reqData)

	reqData.Token = "wrong"

	reqNoExist := toJSON(reqData)

	reqData.Token = token

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
		{"password reset with valid token", reqExisting, contentType, http.StatusCreated, expectedSuccess, token},
		{"password reset with invalid token", reqNoExist, contentType, http.StatusForbidden, toJSON(errorRes{users.ErrUnauthorizedAccess.Error()}), token},
		{"password reset with confirm password not matching", reqPassNoMatch, contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()}), token},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, toJSON(errorRes{api.ErrFailedDecode.Error()}), token},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()}), token},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, toJSON(errorRes{api.ErrFailedDecode.Error()}), token},
		{"password reset request with missing content type", reqExisting, "", http.StatusUnsupportedMediaType, toJSON(errorRes{api.ErrUnsupportedContentType.Error()}), token},
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
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Issuer: user.Email, Type: 0})
	token := tkn.GetValue()
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

	svc.Register(context.Background(), user)

	reqData.Password = user.Password
	reqData.OldPassw = user.Password
	reqData.Token = token
	dataResExisting := toJSON(reqData)

	reqNoExist := toJSON(reqData)

	reqData.OldPassw = "wrong"
	reqWrongPass := toJSON(reqData)

	resData.Msg = users.ErrUnauthorizedAccess.Error()

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password change with valid token", dataResExisting, contentType, http.StatusCreated, expectedSuccess, token},
		{"password change with invalid token", reqNoExist, contentType, http.StatusForbidden, toJSON(errorRes{users.ErrUnauthorizedAccess.Error()}), ""},
		{"password change with invalid old password", reqWrongPass, contentType, http.StatusForbidden, toJSON(errorRes{users.ErrUnauthorizedAccess.Error()}), token},
		{"password change with empty JSON request", "{}", contentType, http.StatusBadRequest, toJSON(errorRes{users.ErrMalformedEntity.Error()}), token},
		{"password change empty request", "", contentType, http.StatusBadRequest, toJSON(errorRes{api.ErrFailedDecode.Error()}), token},
		{"password change missing content type", dataResExisting, "", http.StatusUnsupportedMediaType, toJSON(errorRes{api.ErrUnsupportedContentType.Error()}), token},
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

type errorRes struct {
	Err string `json:"error"`
}
