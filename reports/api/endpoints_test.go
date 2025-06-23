// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/internal/testsutil"
	pkgSch "github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/api"
	"github.com/absmach/magistrala/reports/mocks"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const contentType = "application/json"

var (
	namegen      = namegenerator.NewGenerator()
	domainID     = testsutil.GenerateUUID(&testing.T{})
	userID       = testsutil.GenerateUUID(&testing.T{})
	validID      = testsutil.GenerateUUID(&testing.T{})
	validToken   = "valid"
	invalidToken = "invalid"
	now          = time.Now().UTC().Truncate(time.Minute)
	schedule     = pkgSch.Schedule{
		StartDateTime:   now.Add(1 * time.Hour),
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            now,
	}
	reportConfig = reports.ReportConfig{
		ID:       validID,
		Name:     namegen.Generate(),
		DomainID: domainID,
		Schedule: schedule,
		Status:   reports.EnabledStatus,
		Metrics: []reports.ReqMetric{
			{
				ChannelID: "channel1",
				ClientIDs: []string{"client1"},
				Name:      "metric_name",
			},
		},
		Config: &reports.MetricConfig{
			From:        "now()-1h",
			To:          "now()",
			Title:       title,
			Aggregation: reports.AggConfig{AggType: reports.AggregationAVG, Interval: "1h"},
		},
		Email: &reports.EmailSetting{
			To:      []string{"test@example.com"},
			Subject: "Test Report",
		},
	}
	title = "test_title"
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

func newReportsServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	authn := new(authnmocks.Authentication)

	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	api.MakeHandler(svc, authn, mux, logger, "")

	return httptest.NewServer(mux), svc, authn
}

func toJSON(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestAddReportConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	scheduleInPast := pkgSch.Schedule{
		StartDateTime:   now.Add(-1 * time.Hour),
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            now,
	}

	reportInPast := reportConfig
	reportInPast.Schedule = scheduleInPast

	cases := []struct {
		desc        string
		cfg         reports.ReportConfig
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcRes      reports.ReportConfig
		svcErr      error
		err         error
	}{
		{
			desc:        "add report config successfully",
			cfg:         reportConfig,
			token:       validToken,
			contentType: contentType,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			status:      http.StatusCreated,
			svcRes:      reportConfig,
		},
		{
			desc:        "add report config with invalid token",
			cfg:         reportConfig,
			token:       invalidToken,
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "add report config with empty token",
			token:       "",
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			cfg:         reportConfig,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "add report config with empty domainID",
			token:       validToken,
			cfg:         reportConfig,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "add report config with invalid content type",
			token:       validToken,
			domainID:    domainID,
			cfg:         reportConfig,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "add report config with startdatetime in past",
			token:       validToken,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			cfg:         reportInPast,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "add report config with service error",
			token:       validToken,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			cfg:         reportConfig,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.cfg)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/reports/configs", ts.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("AddReportConfig", mock.Anything, tc.authnRes, mock.Anything).Return(tc.svcRes, tc.svcErr)
			res, err := req.make()

			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewReportConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		id          string
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcRes      reports.ReportConfig
		svcErr      error
		err         error
	}{
		{
			desc:        "view report config successfully",
			id:          validID,
			token:       validToken,
			contentType: contentType,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			status:      http.StatusOK,
			svcRes:      reportConfig,
		},
		{
			desc:        "view report config with invalid token",
			id:          validID,
			token:       invalidToken,
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "view report config with empty token",
			token:       "",
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "view report config with empty domainID",
			token:       validToken,
			id:          validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "view report config with service error",
			token:       validToken,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			id:          validID,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodGet,
				url:         fmt.Sprintf("%s/%s/reports/configs/%s", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ViewReportConfig", mock.Anything, tc.authnRes, tc.id).Return(tc.svcRes, tc.svcErr)
			res, err := req.make()

			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListReportsConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	cases := []struct {
		desc                string
		query               string
		domainID            string
		token               string
		session             smqauthn.Session
		listReportsResponse reports.ReportConfigPage
		status              int
		authnErr            error
		err                 error
	}{
		{
			desc:     "list reports config successfully",
			domainID: domainID,
			token:    validToken,
			status:   http.StatusOK,
			listReportsResponse: reports.ReportConfigPage{
				ReportConfigs: []reports.ReportConfig{reportConfig},
				PageMeta:      reports.PageMeta{Total: 1},
			},
			err: nil,
		},
		{
			desc:     "list reports config with empty token",
			domainID: domainID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list reports config with invalid token",
			domainID: domainID,
			token:    invalidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodGet,
				url:         ts.URL + "/" + tc.domainID + "/reports/configs?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListReportsConfig", mock.Anything, tc.session, mock.Anything).Return(tc.listReportsResponse, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var bodyRes respBody
			err = json.NewDecoder(res.Body).Decode(&bodyRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if bodyRes.Err != "" || bodyRes.Message != "" {
				err = errors.Wrap(errors.New(bodyRes.Err), errors.New(bodyRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateReportConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		updateReq   reports.ReportConfig
		contentType string
		session     smqauthn.Session
		svcResp     reports.ReportConfig
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update report config successfully",
			token:       validToken,
			domainID:    domainID,
			id:          validID,
			updateReq:   reportConfig,
			contentType: contentType,
			svcResp:     reportConfig,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update report config with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			updateReq:   reportConfig,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update report config with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			updateReq:   reportConfig,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update report config with empty domainID",
			token:       validToken,
			id:          validID,
			updateReq:   reportConfig,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "update report config with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			updateReq:   reportConfig,
			contentType: "application/xml",
			svcResp:     reportConfig,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update report config with service error",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			updateReq:   reportConfig,
			contentType: contentType,
			svcResp:     reports.ReportConfig{},
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.updateReq)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/reports/configs/%s", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateReportConfig", mock.Anything, tc.session, mock.Anything).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteReportConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "delete report config successfully",
			token:    validToken,
			domainID: domainID,
			id:       validID,
			svcErr:   nil,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "delete report config with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "delete report config with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "delete report config with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "delete report config with service error",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/reports/configs/%s", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveReportConfig", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableReportConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  reports.ReportConfig
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "enable report config successfully",
			token:    validToken,
			domainID: domainID,
			id:       validID,
			svcResp:  reportConfig,
			svcErr:   nil,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "enable report config with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "enable report config with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "enable report config with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "enable report config with service error",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			svcResp:  reports.ReportConfig{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "enable report config with empty id",
			token:    validToken,
			id:       "",
			domainID: domainID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/reports/configs/%s/enable", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("EnableReportConfig", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableReportConfigEndpoint(t *testing.T) {
	ts, svc, authn := newReportsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  reports.ReportConfig
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "disable report config successfully",
			token:    validToken,
			domainID: domainID,
			id:       validID,
			svcResp:  reportConfig,
			svcErr:   nil,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "disable report config with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "disable report config with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "disable report config with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "disable report config with service error",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			svcResp:  reports.ReportConfig{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "disable report config with empty id",
			token:    validToken,
			id:       "",
			domainID: domainID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/reports/configs/%s/disable", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DisableReportConfig", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

type respBody struct {
	Err     string         `json:"error"`
	Message string         `json:"message"`
	Total   uint64         `json:"total"`
	ID      string         `json:"id"`
	Status  reports.Status `json:"status"`
}
