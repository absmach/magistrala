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
	"regexp"
	"strings"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType       = "application/json"
	validEmail        = "user@example.com"
	invalidEmail      = "userexample.com"
	validPass         = "password"
	invalidPass       = "wrong"
	memberRelationKey = "member"
	authoritiesObjKey = "authorities"
)

var (
	user               = users.User{Email: validEmail, Password: validPass}
	notFoundRes        = toJSON(apiutil.ErrorRes{Err: errors.ErrNotFound.Error()})
	unauthRes          = toJSON(apiutil.ErrorRes{Err: errors.ErrAuthentication.Error()})
	malformedRes       = toJSON(apiutil.ErrorRes{Err: errors.ErrMalformedEntity.Error()})
	weakPassword       = toJSON(apiutil.ErrorRes{Err: users.ErrPasswordFormat.Error()})
	unsupportedRes     = toJSON(apiutil.ErrorRes{Err: errors.ErrUnsupportedContentType.Error()})
	missingTokRes      = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error()})
	missingEmailRes    = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingEmail.Error()})
	missingPassRes     = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingPass.Error()})
	invalidRestPassRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrInvalidResetPass.Error()})
	passRegex          = regexp.MustCompile("^.{8,}$")
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
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	req.Header.Set("Referer", "http://localhost")
	return tr.client.Do(req)
}

func newService() users.Service {
	usersRepo := mocks.NewUserRepository()
	hasher := bcrypt.New()

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)
	email := mocks.NewEmailer()
	idProvider := uuid.New()

	return users.New(usersRepo, hasher, auth, email, idProvider, passRegex)
}

func newServer(svc users.Service) *httptest.Server {
	logger := logger.NewMock()
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
	userNew := toJSON(users.User{Email: "user2@example.com", Password: "password"})
	invalidData := toJSON(users.User{Email: invalidEmail, Password: validPass})
	invalidPasswordData := toJSON(users.User{Email: validEmail, Password: invalidPass})
	invalidFieldData := fmt.Sprintf(`{"email": "%s", "pass": "%s"}`, user.Email, user.Password)
	emptyEmailData := `{"email": ""}`
	emptyHostData := fmt.Sprintf(`{"email": "%s"}`, user.Email)

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})
	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	token := tkn.GetValue()

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		token       string
	}{
		{"register new user", data, contentType, http.StatusCreated, token},
		{"register user with empty token", data, contentType, http.StatusUnauthorized, ""},
		{"register existing user", data, contentType, http.StatusConflict, token},
		{"register user with invalid email address", invalidData, contentType, http.StatusBadRequest, token},
		{"register user with weak password", invalidPasswordData, contentType, http.StatusBadRequest, token},
		{"register new user with unauthenticated access", userNew, contentType, http.StatusUnauthorized, "wrong"},
		{"register existing user with unauthenticated access", data, contentType, http.StatusUnauthorized, "wrong"},
		{"register user with invalid request format", "{", contentType, http.StatusBadRequest, token},
		{"register user with empty email request", emptyEmailData, contentType, http.StatusBadRequest, token},
		{"register user with empty host request", emptyHostData, contentType, http.StatusBadRequest, token},
		{"register user with empty request", "", contentType, http.StatusBadRequest, token},
		{"register user with invalid field name", invalidFieldData, contentType, http.StatusBadRequest, token},
		{"register user with missing content type", data, "", http.StatusUnsupportedMediaType, token},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
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

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	token := tkn.GetValue()
	tokenData := toJSON(map[string]string{"token": token})
	data := toJSON(user)
	invalidEmailData := toJSON(users.User{
		Email:    invalidEmail,
		Password: validPass,
	})
	invalidData := toJSON(users.User{
		Email:    validEmail,
		Password: "invalid_password",
	})
	nonexistentData := toJSON(users.User{
		Email:    "non-existentuser@example.com",
		Password: validPass,
	})
	_, err := svc.Register(context.Background(), token, user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"login with valid credentials", data, contentType, http.StatusCreated, tokenData},
		{"login with invalid credentials", invalidData, contentType, http.StatusUnauthorized, unauthRes},
		{"login with invalid email address", invalidEmailData, contentType, http.StatusBadRequest, malformedRes},
		{"login non-existent user", nonexistentData, contentType, http.StatusUnauthorized, unauthRes},
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

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	token := tkn.GetValue()

	userID, err := svc.Register(context.Background(), token, user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		status int
		res    string
	}{
		{"user info with valid token", token, http.StatusOK, ""},
		{"user info with invalid token", "", http.StatusUnauthorized, ""},
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
		Password: validPass,
	})

	expectedExisting := toJSON(struct {
		Msg string `json:"msg"`
	}{
		api.MailSent,
	})

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)
	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	token := tkn.GetValue()

	_, err := svc.Register(context.Background(), token, user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"password reset request with valid email", data, contentType, http.StatusCreated, expectedExisting},
		{"password reset request with invalid email", nonexistentData, contentType, http.StatusNotFound, notFoundRes},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, malformedRes},
		{"password reset request with empty email request", "{}", contentType, http.StatusBadRequest, missingEmailRes},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, malformedRes},
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
	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		ConfPass string `json:"confirm_password,omitempty"`
	}{}

	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, err := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue user token error: %s", err))

	token := tkn.GetValue()

	_, err = svc.Register(context.Background(), token, user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	reqData.Password = user.Password
	reqData.ConfPass = user.Password
	reqData.Token = token
	reqExisting := toJSON(reqData)

	reqData.Token = "wrong"

	reqNoExist := toJSON(reqData)

	reqData.Token = token

	reqData.ConfPass = invalidPass
	reqPassNoMatch := toJSON(reqData)

	reqData.Password = invalidPass
	reqPassWeak := toJSON(reqData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password reset with valid token", reqExisting, contentType, http.StatusCreated, "{}", token},
		{"password reset with invalid token", reqNoExist, contentType, http.StatusUnauthorized, unauthRes, token},
		{"password reset with confirm password not matching", reqPassNoMatch, contentType, http.StatusBadRequest, invalidRestPassRes, token},
		{"password reset request with invalid request format", "{", contentType, http.StatusBadRequest, malformedRes, token},
		{"password reset request with empty JSON request", "{}", contentType, http.StatusBadRequest, missingPassRes, token},
		{"password reset request with empty request", "", contentType, http.StatusBadRequest, malformedRes, token},
		{"password reset request with missing content type", reqExisting, "", http.StatusUnsupportedMediaType, unsupportedRes, token},
		{"password reset with weak password", reqPassWeak, contentType, http.StatusBadRequest, weakPassword, token},
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
	mockAuthzDB := map[string][]mocks.SubjectSet{}
	mockAuthzDB[user.Email] = append(mockAuthzDB[user.Email], mocks.SubjectSet{Object: authoritiesObjKey, Relation: memberRelationKey})

	auth := mocks.NewAuthService(map[string]string{user.Email: user.Email}, mockAuthzDB)

	tkn, _ := auth.Issue(context.Background(), &mainflux.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	token := tkn.GetValue()

	reqData := struct {
		Token    string `json:"token,omitempty"`
		Password string `json:"password,omitempty"`
		OldPassw string `json:"old_password,omitempty"`
	}{}

	_, err := svc.Register(context.Background(), token, user)
	require.Nil(t, err, fmt.Sprintf("register user got unexpected error: %s", err))

	reqData.Password = user.Password
	reqData.OldPassw = user.Password
	reqData.Token = token
	dataResExisting := toJSON(reqData)

	reqNoExist := toJSON(reqData)

	reqData.OldPassw = invalidPass
	reqWrongPass := toJSON(reqData)

	reqData.OldPassw = user.Password
	reqData.Password = invalidPass
	reqWeakPass := toJSON(reqData)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
		tok         string
	}{
		{"password change with valid token", dataResExisting, contentType, http.StatusCreated, "{}", token},
		{"password change with empty token", reqNoExist, contentType, http.StatusUnauthorized, missingTokRes, ""},
		{"password change with invalid old password", reqWrongPass, contentType, http.StatusUnauthorized, unauthRes, token},
		{"password change with invalid new password", reqWeakPass, contentType, http.StatusBadRequest, weakPassword, token},
		{"password change with empty JSON request", "{}", contentType, http.StatusBadRequest, missingPassRes, token},
		{"password change empty request", "", contentType, http.StatusBadRequest, malformedRes, token},
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
