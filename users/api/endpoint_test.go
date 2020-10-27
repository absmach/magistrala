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
	"strings"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType  = "application/json"
	invalidEmail = "userexample.com"
)

var (
	user           = users.User{Email: "user@example.com", Password: "password"}
	notFoundRes    = toJSON(errorRes{users.ErrUserNotFound.Error()})
	unauthRes      = toJSON(errorRes{users.ErrUnauthorizedAccess.Error()})
	malformedRes   = toJSON(errorRes{users.ErrMalformedEntity.Error()})
	unsupportedRes = toJSON(errorRes{api.ErrUnsupportedContentType.Error()})
	failDecodeRes  = toJSON(errorRes{api.ErrFailedDecode.Error()})
	groupExists    = toJSON(errorRes{users.ErrGroupConflict.Error()})
)

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
	usersRepo := mocks.NewUserRepository()
	groupRepo := mocks.NewGroupRepository()
	hasher := bcrypt.New()
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	email := mocks.NewEmailer()

	return users.New(usersRepo, groupRepo, hasher, auth, email)
}

func newServer(svc users.Service) *httptest.Server {
	mux := api.MakeHandler(svc, mocktracer.New())
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
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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
		Password: "password",
	})
	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"login with valid credentials", data, contentType, http.StatusCreated, tokenData},
		{"login with invalid credentials", invalidData, contentType, http.StatusForbidden, unauthRes},
		{"login with invalid email address", invalidEmailData, contentType, http.StatusBadRequest, malformedRes},
		{"login non-existent user", nonexistentData, contentType, http.StatusForbidden, unauthRes},
		{"login with invalid request format", "{", contentType, http.StatusBadRequest, malformedRes},
		{"login with empty JSON request", "{}", contentType, http.StatusBadRequest, malformedRes},
		{"login with empty request", "", contentType, http.StatusBadRequest, malformedRes},
		{"login with missing content type", data, "", http.StatusUnsupportedMediaType, unsupportedRes},
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

func TestUser(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	userID, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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
			url:    fmt.Sprintf("%s/users/%s", ts.URL, userID),
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
		Password: "password",
	})

	expectedExisting := toJSON(struct {
		Msg string `json:"msg"`
	}{
		api.MailSent,
	})

	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"password reset request with valid email", data, contentType, http.StatusCreated, expectedExisting},
		{"password reset request with invalid email", nonexistentData, contentType, http.StatusBadRequest, notFoundRes},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, failDecodeRes},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, malformedRes},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, failDecodeRes},
		{"password reset request with missing content type", data, "", http.StatusUnsupportedMediaType, unsupportedRes},
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

	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})

	tkn, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue user token error: %s", err))

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
		{"password reset with invalid token", reqNoExist, contentType, http.StatusForbidden, unauthRes, token},
		{"password reset with confirm password not matching", reqPassNoMatch, contentType, http.StatusBadRequest, malformedRes, token},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, failDecodeRes, token},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, malformedRes, token},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, failDecodeRes, token},
		{"password reset request with missing content type", reqExisting, "", http.StatusUnsupportedMediaType, unsupportedRes, token},
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

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
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

	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

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
		{"password change with invalid token", reqNoExist, contentType, http.StatusForbidden, unauthRes, ""},
		{"password change with invalid old password", reqWrongPass, contentType, http.StatusForbidden, unauthRes, token},
		{"password change with empty JSON request", "{}", contentType, http.StatusBadRequest, malformedRes, token},
		{"password change empty request", "", contentType, http.StatusBadRequest, failDecodeRes, token},
		{"password change missing content type", dataResExisting, "", http.StatusUnsupportedMediaType, unsupportedRes, token},
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

func TestGroupCreate(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email})

	_, err := svc.Register(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	token := tkn.GetValue()

	expectedSuccess := ""

	groupData := struct {
		Token       string `json:"token,omitempty"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}{}

	groupData.Token = token
	groupData.Name = "Mainflux"
	createValidTokenRequest := toJSON(groupData)

	groupData.Token = "invalid"
	createInvalidTokenRequest := toJSON(groupData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"group create with valid token", createValidTokenRequest, contentType, http.StatusCreated, expectedSuccess, token},
		{"group create with existing name", createValidTokenRequest, contentType, http.StatusConflict, groupExists, token},
		{"group create with invalid token", createInvalidTokenRequest, contentType, http.StatusForbidden, unauthRes, ""},
		{"group create with empty JSON request", "{}", contentType, http.StatusBadRequest, malformedRes, token},
		{"group create empty request", "", contentType, http.StatusBadRequest, malformedRes, token},
		{"group create missing content type", createValidTokenRequest, "", http.StatusUnsupportedMediaType, unsupportedRes, token},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups", ts.URL),
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
