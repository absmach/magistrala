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

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/bootstrap"
	bsapi "github.com/absmach/magistrala/bootstrap/api"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	validToken   = "validToken"
	domainID     = "b4d7d79e-fd99-4c2b-ac09-524e43df6888"
	invalidToken = "invalid"
	email        = "test@example.com"
	unknown      = "unknown"
	contentType  = "application/json"
	wrongID      = "wrong_id"

	addName    = "name"
	addContent = "config"
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey         = []byte("1234567891011121")
	addExternalID  = testsutil.GenerateUUID(&testing.T{})
	addExternalKey = testsutil.GenerateUUID(&testing.T{})
	addID          = testsutil.GenerateUUID(&testing.T{})
	addReq         = struct {
		ExternalID  string `json:"external_id"`
		ExternalKey string `json:"external_key"`
		Name        string `json:"name"`
		Content     string `json:"content"`
	}{
		ExternalID:  addExternalID,
		ExternalKey: addExternalKey,
		Name:        "name",
		Content:     "config",
	}

	updateReq = struct {
		Content    string           `json:"content,omitempty"`
		Status     bootstrap.Status `json:"status,omitempty"`
		ClientCert string           `json:"client_cert,omitempty"`
		CACert     string           `json:"ca_cert,omitempty"`
	}{
		Content:    "config update",
		Status:     1,
		ClientCert: "newcert",
		CACert:     "newca",
	}

	missingIDRes              = toJSON(apiutil.ErrMissingID)
	missingKeyRes             = toJSON(apiutil.ErrBearerKey)
	unknownExternalIDErrorRes = toJSON(svcerr.ErrNotFound)
	extKeyRes                 = toJSON(bootstrap.ErrExternalKey)
	extSecKeyRes              = toJSON(bootstrap.ErrExternalKeySecure)
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
		ID:          addID,
		ExternalID:  addExternalID,
		ExternalKey: addExternalKey,
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
		req.Header.Set("Authorization", apiutil.ClientPrefix+tr.key)
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

func newBootstrapServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	logger := mglog.NewMock()
	svc := new(mocks.Service)
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))
	mux := bsapi.MakeHandler(svc, am, bootstrap.NewConfigReader(encKey), logger, instanceID)
	return httptest.NewServer(mux), svc, authn
}

func toJSON(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestAdd(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	data := toJSON(addReq)

	cases := []struct {
		desc            string
		req             string
		domainID        string
		token           string
		session         smqauthn.Session
		contentType     string
		status          int
		location        string
		authenticateErr error
		err             error
	}{
		{
			desc:            "add a config with invalid token",
			req:             data,
			domainID:        domainID,
			token:           invalidToken,
			contentType:     contentType,
			status:          http.StatusUnauthorized,
			location:        "",
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:        "add a valid config",
			req:         data,
			domainID:    domainID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusCreated,
			location:    "/clients/configs/" + c.ID,
			err:         nil,
		},
		{
			desc:        "add a config with wrong content type",
			req:         data,
			domainID:    domainID,
			token:       validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
			location:    "",
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "add an existing config",
			req:         data,
			domainID:    domainID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
			err:         svcerr.ErrConflict,
		},
		{
			desc:        "add a config with wrong JSON",
			req:         "{\"external_id\": 5}",
			domainID:    domainID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
		},
		{
			desc:        "add a config with invalid request format",
			req:         "}",
			domainID:    domainID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
			err:         svcerr.ErrMalformedEntity,
		},
		{
			desc:        "add a config with empty JSON",
			req:         "{}",
			domainID:    domainID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:        "add a config with an empty request",
			req:         "",
			domainID:    domainID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			location:    "",
			err:         svcerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)

			svcCall := svc.On("Add", mock.Anything, tc.session, tc.token, mock.Anything).Return(c, tc.err)
			req := testRequest{
				client:      bs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/configs", bs.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.req),
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			location := res.Header.Get("Location")
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location '%s' got '%s'", tc.desc, tc.location, location))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestView(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	data := config{
		ID:         c.ID,
		Status:     c.Status,
		ExternalID: c.ExternalID,
		Name:       c.Name,
		Content:    c.Content,
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		id              string
		status          int
		res             config
		authenticateErr error
		err             error
	}{
		{
			desc:            "view a config with invalid token",
			token:           invalidToken,
			id:              c.ID,
			status:          http.StatusUnauthorized,
			res:             config{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:   "view a config",
			token:  validToken,
			id:     c.ID,
			status: http.StatusOK,
			res:    data,
			err:    nil,
		},
		{
			desc:   "view a non-existing config",
			token:  validToken,
			id:     wrongID,
			status: http.StatusNotFound,
			res:    config{},
			err:    svcerr.ErrNotFound,
		},
		{
			desc:   "view a config with an empty token",
			token:  "",
			id:     c.ID,
			status: http.StatusUnauthorized,
			res:    config{},
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "view config without authorization",
			token:  validToken,
			id:     c.ID,
			status: http.StatusForbidden,
			res:    config{},
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("View", mock.Anything, tc.session, tc.id).Return(c, tc.err)
			req := testRequest{
				client: bs.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/clients/configs/%s", bs.URL, domainID, tc.id),
				token:  tc.token,
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			var view config
			if err := json.NewDecoder(res.Body).Decode(&view); err != io.EOF {
				assert.Nil(t, err, fmt.Sprintf("Decoding expected to succeed %s: %s", tc.desc, err))
			}

			assert.Equal(t, tc.res, view, fmt.Sprintf("%s: expected response '%s' got '%s'", tc.desc, tc.res, view))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdate(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	data := toJSON(updateReq)

	cases := []struct {
		desc            string
		req             string
		id              string
		token           string
		session         smqauthn.Session
		contentType     string
		status          int
		authenticateErr error
		err             error
	}{
		{
			desc:            "update with invalid token",
			req:             data,
			id:              c.ID,
			token:           invalidToken,
			contentType:     contentType,
			status:          http.StatusUnauthorized,
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:        "update with an empty token",
			req:         data,
			id:          c.ID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update a valid config",
			req:         data,
			id:          c.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update a config with wrong content type",
			req:         data,
			id:          c.ID,
			token:       validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update a non-existing config",
			req:         data,
			id:          wrongID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "update a config with invalid request format",
			req:         "}",
			id:          c.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
		},
		{
			desc:        "update a config with an empty request",
			id:          c.ID,
			req:         "",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("Update", mock.Anything, tc.session, mock.Anything).Return(tc.err)
			req := testRequest{
				client:      bs.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/clients/configs/%s", bs.URL, domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.req),
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateCert(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	data := toJSON(updateReq)

	cases := []struct {
		desc            string
		req             string
		id              string
		token           string
		session         smqauthn.Session
		contentType     string
		status          int
		authenticateErr error
		err             error
	}{
		{
			desc:            "update with invalid token",
			req:             data,
			id:              c.ID,
			token:           invalidToken,
			contentType:     contentType,
			status:          http.StatusUnauthorized,
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:        "update with an empty token",
			req:         data,
			id:          c.ID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update a valid config",
			req:         data,
			id:          c.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update a config with wrong content type",
			req:         data,
			id:          c.ID,
			token:       validToken,
			contentType: "",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update a non-existing config",
			req:         data,
			id:          wrongID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "update a config with invalid request format",
			req:         "}",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
		},
		{
			desc:        "update a config with an empty request",
			id:          c.ID,
			req:         "",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateCert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(c, tc.err)
			req := testRequest{
				client:      bs.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/clients/configs/certs/%s", bs.URL, domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.req),
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestList(t *testing.T) {
	configNum := 101
	changedStatusNum := 20
	var active, inactive []config
	list := make([]config, configNum)

	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	path := fmt.Sprintf("%s/%s/%s", bs.URL, domainID, "clients/configs")

	c := newConfig()

	for i := 0; i < configNum; i++ {
		c.ExternalID = strconv.Itoa(i)
		c.Name = fmt.Sprintf("%s-%d", addName, i)
		c.ExternalKey = fmt.Sprintf("%s%s", addExternalKey, strconv.Itoa(i))

		s := config{
			ID:         c.ID,
			ExternalID: c.ExternalID,
			Name:       c.Name,
			Content:    c.Content,
			Status:     c.Status,
		}
		list[i] = s
	}
	// Change status of first 20 elements for filtering tests.
	for i := 0; i < changedStatusNum; i++ {
		if i%2 == 0 {
			// Even elements remain inactive (default status).
			inactive = append(inactive, list[i])
			continue
		}
		// Odd elements are enabled (active).
		enabledCfg := bootstrap.Config{ID: list[i].ID, Status: bootstrap.Active}
		svcCall := svc.On("EnableConfig", context.Background(), mock.Anything, mock.Anything).Return(enabledCfg, nil)
		_, err := svc.EnableConfig(context.Background(), smqauthn.Session{}, list[i].ID)
		assert.Nil(t, err, fmt.Sprintf("Enabling config expected to succeed: %s.\n", err))
		svcCall.Unset()
		list[i].Status = bootstrap.Active
		active = append(active, list[i])
	}

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		url             string
		status          int
		res             configPage
		authenticateErr error
		err             error
	}{
		{
			desc:            "view list with invalid token",
			token:           invalidToken,
			url:             fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 10),
			status:          http.StatusUnauthorized,
			res:             configPage{},
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:   "view list with an empty token",
			token:  "",
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 10),
			status: http.StatusUnauthorized,
			res:    configPage{},
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "view list",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 1),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  0,
				Limit:   1,
				Configs: list[0:1],
			},
			err: nil,
		},
		{
			desc:   "view list searching by name",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", path, 0, 100, "95"),
			status: http.StatusOK,
			res: configPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: list[95:96],
			},
			err: nil,
		},
		{
			desc:   "view last page",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 100, 10),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  100,
				Limit:   10,
				Configs: list[100:],
			},
			err: nil,
		},
		{
			desc:   "view with limit greater than allowed",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", path, 0, 1000),
			status: http.StatusBadRequest,
			res:    configPage{},
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:   "view list with no specified limit and offset",
			token:  validToken,
			url:    path,
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  0,
				Limit:   10,
				Configs: list[0:10],
			},
			err: nil,
		},
		{
			desc:   "view list with no specified limit",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d", path, 10),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  10,
				Limit:   10,
				Configs: list[10:20],
			},
			err: nil,
		},
		{
			desc:   "view list with no specified offset",
			token:  validToken,
			url:    fmt.Sprintf("%s?limit=%d", path, 10),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list)),
				Offset:  0,
				Limit:   10,
				Configs: list[0:10],
			},
			err: nil,
		},
		{
			desc:   "view list with limit < 0",
			token:  validToken,
			url:    fmt.Sprintf("%s?limit=%d", path, -10),
			status: http.StatusBadRequest,
			res:    configPage{},
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:   "view list with offset < 0",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d", path, -10),
			status: http.StatusBadRequest,
			res:    configPage{},
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:   "view list with invalid query parameters",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&status=%s&key=%%", path, 10, 10, bootstrap.Disabled),
			status: http.StatusBadRequest,
			res:    configPage{},
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:   "view first 10 active",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&status=%s", path, 0, 20, bootstrap.Enabled),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(active)),
				Offset:  0,
				Limit:   20,
				Configs: active,
			},
			err: nil,
		},
		{
			desc:   "view first 10 inactive",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&status=%s", path, 0, 20, bootstrap.Disabled),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list) - len(inactive)),
				Offset:  0,
				Limit:   20,
				Configs: inactive,
			},
			err: nil,
		},
		{
			desc:   "view first 5 active",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&status=%s", path, 0, 10, bootstrap.Enabled),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(active)),
				Offset:  0,
				Limit:   10,
				Configs: active[:5],
			},
			err: nil,
		},
		{
			desc:   "view last 5 inactive",
			token:  validToken,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&status=%s", path, 10, 10, bootstrap.Disabled),
			status: http.StatusOK,
			res: configPage{
				Total:   uint64(len(list) - len(active)),
				Offset:  10,
				Limit:   10,
				Configs: inactive[5:],
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(bootstrap.ConfigsPage{Total: tc.res.Total, Offset: tc.res.Offset, Limit: tc.res.Limit}, tc.err)
			req := testRequest{
				client: bs.Client(),
				method: http.MethodGet,
				url:    tc.url,
				token:  tc.token,
			}

			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			var body configPage

			err = json.NewDecoder(res.Body).Decode(&body)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

			assert.Equal(t, tc.res.Total, body.Total, fmt.Sprintf("%s: expected response total '%d' got '%d'", tc.desc, tc.res.Total, body.Total))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemove(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		status          int
		authenticateErr error
		err             error
	}{
		{
			desc:            "remove with invalid token",
			id:              c.ID,
			token:           invalidToken,
			status:          http.StatusUnauthorized,
			authenticateErr: svcerr.ErrAuthentication,
			err:             svcerr.ErrAuthentication,
		},
		{
			desc:   "remove with an empty token",
			id:     c.ID,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "remove non-existing config",
			id:     "non-existing",
			token:  validToken,
			status: http.StatusNoContent,
			err:    nil,
		},
		{
			desc:   "remove config",
			id:     c.ID,
			token:  validToken,
			status: http.StatusNoContent,
			err:    nil,
		},
		{
			desc:   "remove removed config",
			id:     wrongID,
			token:  validToken,
			status: http.StatusNoContent,
			err:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			req := testRequest{
				client: bs.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/clients/configs/%s", bs.URL, domainID, tc.id),
				token:  tc.token,
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestBootstrap(t *testing.T) {
	bs, svc, _ := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	encExternKey, err := enc([]byte(c.ExternalKey))
	assert.Nil(t, err, fmt.Sprintf("Encrypting config expected to succeed: %s.\n", err))

	s := struct {
		ID         string `json:"id"`
		Content    string `json:"content"`
		ClientCert string `json:"client_cert"`
		ClientKey  string `json:"client_key"`
		CACert     string `json:"ca_cert"`
	}{
		ID:         c.ID,
		Content:    c.Content,
		ClientCert: c.ClientCert,
		ClientKey:  c.ClientKey,
		CACert:     c.CACert,
	}

	data := toJSON(s)

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		status      int
		res         string
		secure      bool
		err         error
	}{
		{
			desc:        "bootstrap a Client with unknown ID",
			externalID:  unknown,
			externalKey: c.ExternalKey,
			status:      http.StatusNotFound,
			res:         unknownExternalIDErrorRes,
			secure:      false,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "bootstrap a Client with an empty ID",
			externalID:  "",
			externalKey: c.ExternalKey,
			status:      http.StatusBadRequest,
			res:         missingIDRes,
			secure:      false,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "bootstrap a Client with unknown key",
			externalID:  c.ExternalID,
			externalKey: unknown,
			status:      http.StatusForbidden,
			res:         extKeyRes,
			secure:      false,
			err:         bootstrap.ErrExternalKey,
		},
		{
			desc:        "bootstrap a Client with an empty key",
			externalID:  c.ExternalID,
			externalKey: "",
			status:      http.StatusUnauthorized,
			res:         missingKeyRes,
			secure:      false,
			err:         apiutil.ErrBearerKey,
		},
		{
			desc:        "bootstrap known Client",
			externalID:  c.ExternalID,
			externalKey: c.ExternalKey,
			status:      http.StatusOK,
			res:         data,
			secure:      false,
			err:         nil,
		},
		{
			desc:        "bootstrap secure",
			externalID:  fmt.Sprintf("secure/%s", c.ExternalID),
			externalKey: hex.EncodeToString(encExternKey),
			status:      http.StatusOK,
			res:         data,
			secure:      true,
			err:         nil,
		},
		{
			desc:        "bootstrap secure with unencrypted key",
			externalID:  fmt.Sprintf("secure/%s", c.ExternalID),
			externalKey: c.ExternalKey,
			status:      http.StatusForbidden,
			res:         extSecKeyRes,
			secure:      true,
			err:         bootstrap.ErrExternalKeySecure,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Bootstrap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(c, tc.err)
			req := testRequest{
				client: bs.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/clients/bootstrap/%s", bs.URL, tc.externalID),
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
			svcCall.Unset()
		})
	}
}

func TestChangeStatus(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()
	c := newConfig()

	activeCfg := bootstrap.Config{ID: c.ID, Status: bootstrap.Active}
	inactiveCfg := bootstrap.Config{ID: c.ID, Status: bootstrap.Inactive}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		action          string
		status          int
		authenticateErr error
		svcCfg          bootstrap.Config
		svcErr          error
	}{
		{
			desc:            "enable with invalid token",
			id:              c.ID,
			token:           invalidToken,
			action:          "enable",
			status:          http.StatusUnauthorized,
			authenticateErr: svcerr.ErrAuthentication,
		},
		{
			desc:   "enable with empty token",
			id:     c.ID,
			token:  "",
			action: "enable",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "enable config",
			id:     c.ID,
			token:  validToken,
			action: "enable",
			status: http.StatusOK,
			svcCfg: activeCfg,
		},
		{
			desc:   "disable config",
			id:     c.ID,
			token:  validToken,
			action: "disable",
			status: http.StatusOK,
			svcCfg: inactiveCfg,
		},
		{
			desc:   "enable non-existing config",
			id:     wrongID,
			token:  validToken,
			action: "enable",
			status: http.StatusNotFound,
			svcErr: svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			methodName := "EnableConfig"
			if tc.action == "disable" {
				methodName = "DisableConfig"
			}
			svcCall := svc.On(methodName, mock.Anything, tc.session, mock.Anything).Return(tc.svcCfg, tc.svcErr)
			req := testRequest{
				client: bs.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/clients/configs/%s/%s", bs.URL, domainID, tc.id, tc.action),
				token:  tc.token,
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUploadProfile(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()

	session := smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
	saved := bootstrap.Profile{
		ID:              testsutil.GenerateUUID(t),
		Name:            "gateway",
		TemplateFormat:  bootstrap.TemplateFormatGoTemplate,
		ContentTemplate: "{{ .Device.ID }}",
	}

	cases := []struct {
		desc        string
		contentType string
		body        string
		profile     bootstrap.Profile
	}{
		{
			desc:        "upload JSON profile",
			contentType: "application/json",
			body:        `{"name":"gateway","template_format":"go-template","content_template":"{{ .Device.ID }}"}`,
			profile: bootstrap.Profile{
				Name:            "gateway",
				TemplateFormat:  bootstrap.TemplateFormatGoTemplate,
				ContentTemplate: "{{ .Device.ID }}",
			},
		},
		{
			desc:        "upload YAML profile",
			contentType: "application/yaml",
			body:        "name: gateway\ntemplate_format: go-template\ncontent_template: '{{ .Device.ID }}'\n",
			profile: bootstrap.Profile{
				Name:            "gateway",
				TemplateFormat:  bootstrap.TemplateFormatGoTemplate,
				ContentTemplate: "{{ .Device.ID }}",
			},
		},
		{
			desc:        "upload TOML profile",
			contentType: "application/toml",
			body:        "name = 'gateway'\ntemplate_format = 'go-template'\ncontent_template = '{{ .Device.ID }}'\n",
			profile: bootstrap.Profile{
				Name:            "gateway",
				TemplateFormat:  bootstrap.TemplateFormatGoTemplate,
				ContentTemplate: "{{ .Device.ID }}",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", mock.Anything, validToken).Return(session, nil)
			svcCall := svc.On("CreateProfile", mock.Anything, session, tc.profile).Return(saved, nil)
			req := testRequest{
				client:      bs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/bootstrap/profiles/upload", bs.URL, domainID),
				contentType: tc.contentType,
				token:       validToken,
				body:        strings.NewReader(tc.body),
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, http.StatusCreated, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, http.StatusCreated, res.StatusCode))
			assert.Equal(t, "/bootstrap/profiles/"+saved.ID, res.Header.Get("Location"))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestProfileSlots(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()

	session := smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
	profileID := testsutil.GenerateUUID(t)
	slots := []bootstrap.BindingSlot{
		{Name: "mqtt_client", Type: "client", Required: true, Fields: []string{"id", "secret"}},
		{Name: "telemetry", Type: "channel", Required: true, Fields: []string{"id", "topic"}},
	}
	profile := bootstrap.Profile{
		ID:           profileID,
		Name:         "gateway",
		BindingSlots: slots,
	}
	authCall := auth.On("Authenticate", mock.Anything, validToken).Return(session, nil)
	svcCall := svc.On("ViewProfile", mock.Anything, session, profileID).Return(profile, nil)

	req := testRequest{
		client: bs.Client(),
		method: http.MethodGet,
		url:    fmt.Sprintf("%s/%s/clients/bootstrap/profiles/%s/slots", bs.URL, domainID, profileID),
		token:  validToken,
	}
	res, err := req.make()
	assert.Nil(t, err, fmt.Sprintf("profile slots unexpected error %s", err))
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected status code %d got %d", http.StatusOK, res.StatusCode))

	var got struct {
		BindingSlots []bootstrap.BindingSlot `json:"binding_slots"`
	}
	err = json.NewDecoder(res.Body).Decode(&got)
	assert.Nil(t, err, fmt.Sprintf("decoding profile slots expected to succeed: %s", err))
	assert.ElementsMatch(t, slots, got.BindingSlots)

	svcCall.Unset()
	authCall.Unset()
}

func TestRenderPreview(t *testing.T) {
	bs, svc, auth := newBootstrapServer()
	defer bs.Close()

	session := smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
	profileID := testsutil.GenerateUUID(t)
	configID := testsutil.GenerateUUID(t)
	profile := bootstrap.Profile{
		ID:              profileID,
		Name:            "gateway",
		TemplateFormat:  bootstrap.TemplateFormatGoTemplate,
		ContentTemplate: `device={{ .Device.ID }} site={{ .Vars.site }} topic={{ index (index .Bindings "telemetry").Snapshot "topic" }}`,
	}
	authCall := auth.On("Authenticate", mock.Anything, validToken).Return(session, nil)
	svcCall := svc.On("ViewProfile", mock.Anything, session, profileID).Return(profile, nil)

	reqBody := struct {
		Config   bootstrap.Config            `json:"config"`
		Bindings []bootstrap.BindingSnapshot `json:"bindings"`
	}{
		Config: bootstrap.Config{
			ID:         configID,
			ExternalID: "gw-001",
			RenderContext: map[string]any{
				"site": "warehouse-1",
			},
		},
		Bindings: []bootstrap.BindingSnapshot{
			{
				Slot:       "telemetry",
				Type:       "channel",
				ResourceID: "ch-1",
				Snapshot: map[string]any{
					"topic": "devices/gw-001/telemetry",
				},
			},
		},
	}

	req := testRequest{
		client:      bs.Client(),
		method:      http.MethodPost,
		url:         fmt.Sprintf("%s/%s/clients/bootstrap/profiles/%s/render-preview", bs.URL, domainID, profileID),
		contentType: contentType,
		token:       validToken,
		body:        strings.NewReader(toJSON(reqBody)),
	}
	res, err := req.make()
	assert.Nil(t, err, fmt.Sprintf("render preview unexpected error %s", err))
	assert.Equal(t, http.StatusOK, res.StatusCode, fmt.Sprintf("expected status code %d got %d", http.StatusOK, res.StatusCode))

	var got struct {
		Content string `json:"content"`
	}
	err = json.NewDecoder(res.Body).Decode(&got)
	assert.Nil(t, err, fmt.Sprintf("decoding render preview expected to succeed: %s", err))
	assert.Equal(t, "device="+configID+" site=warehouse-1 topic=devices/gw-001/telemetry", got.Content)

	svcCall.Unset()
	authCall.Unset()
}

type config struct {
	ID         string           `json:"id,omitempty"`
	ExternalID string           `json:"external_id"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name"`
	Status     bootstrap.Status `json:"status"`
}

type configPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Configs []config `json:"configs"`
}
