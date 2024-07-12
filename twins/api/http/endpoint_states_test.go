// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/mocks"
	"github.com/absmach/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	numRecs      = 100
	publisher    = "twins"
	validToken   = "validToken"
	invalidToken = "invalidToken"
)

var (
	subtopics = []string{"engine", "chassis", "wheel_2"}
	channels  = []string{"01ec3c3e-0e66-4e69-9751-a0545b44e08f", "48061e4f-7c23-4f5c-9012-0f9b7cd9d18d", "5b2180e4-e96b-4469-9dc1-b6745078d0b6"}
)

type stateRes struct {
	TwinID     string                 `json:"twin_id"`
	ID         int64                  `json:"id"`
	Definition int                    `json:"definition"`
	Payload    map[string]interface{} `json:"payload"`
}

type statesPageRes struct {
	pageRes
	States []stateRes `json:"states"`
}

func NewService() (twins.Service, *authmocks.AuthClient, *mocks.TwinRepository, *mocks.TwinCache, *mocks.StateRepository) {
	auth := new(authmocks.AuthClient)
	twinsRepo := new(mocks.TwinRepository)
	twinCache := new(mocks.TwinCache)
	statesRepo := new(mocks.StateRepository)
	idProvider := uuid.NewMock()
	subs := map[string]string{"chanID": "chanID"}
	broker := mocks.NewBroker(subs)

	return twins.New(broker, auth, twinsRepo, twinCache, statesRepo, idProvider, "chanID", mglog.NewMock()), auth, twinsRepo, twinCache, statesRepo
}

func TestListStates(t *testing.T) {
	svc, auth, _, _, stateRepo := NewService()
	ts := newServer(svc)
	defer ts.Close()

	def := mocks.CreateDefinition(channels[0:2], subtopics[0:2])
	twin := twins.Twin{
		Owner:       email,
		Definitions: []twins.Definition{def},
		ID:          testsutil.GenerateUUID(t),
		Created:     time.Now(),
	}
	recs := make([]senml.Record, numRecs)

	var data []stateRes
	for i := 0; i < len(recs); i++ {
		res := createStateResponse(i, twin, recs[i])
		data = append(data, res)
	}

	baseURL := fmt.Sprintf("%s/states/%s", ts.URL, twin.ID)
	queryFmt := "%s?offset=%d&limit=%d"
	cases := []struct {
		desc        string
		token       string
		status      int
		url         string
		res         []stateRes
		err         error
		page        twins.StatesPage
		identifyErr error
		userID      string
	}{
		{
			desc:   "get a list of states",
			token:  validToken,
			status: http.StatusOK,
			url:    baseURL,
			res:    data[0:10],
			err:    nil,
			page: twins.StatesPage{
				States: convState(data[0:10]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of states with valid offset and limit",
			token:  validToken,
			status: http.StatusOK,
			url:    fmt.Sprintf(queryFmt, baseURL, 20, 15),
			res:    data[20:35],
			page: twins.StatesPage{
				States: convState(data[20:35]),
			},
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with invalid token",
			token:       invalidToken,
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, 5),
			res:         nil,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "get a list of states with empty token",
			token:       "",
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, 5),
			res:         nil,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:   "get a list of states with + limit > total",
			token:  validToken,
			status: http.StatusOK,
			url:    fmt.Sprintf(queryFmt, baseURL, 91, 20),
			res:    data[91:],
			page: twins.StatesPage{
				States: convState(data[91:]),
			},
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with negative offset",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, -1, 5),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with negative limit",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, -5),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with zero limit",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, 0),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with limit greater than max",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, 110),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with invalid offset",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s?offset=invalid&limit=%d", baseURL, 15),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with invalid limit",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s?offset=%d&limit=invalid", baseURL, 0),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of states without offset",
			token:  validToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", baseURL, 15),
			res:    data[0:15],
			page: twins.StatesPage{
				States: convState(data[0:15]),
			},
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of states without limit",
			token:  validToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", baseURL, 14),
			res:    data[14:24],
			page: twins.StatesPage{
				States: convState(data[14:24]),
			},
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of states with invalid number of parameters",
			token:       validToken,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s%s", baseURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of states with redundant query parameters",
			token:  validToken,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", baseURL, 0, 5),
			res:    data[0:5],
			page: twins.StatesPage{
				States: convState(data[0:5]),
			},
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := stateRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.err)
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var resData statesPageRes
		if tc.res != nil {
			err = json.NewDecoder(res.Body).Decode(&resData)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		}

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, resData.States, fmt.Sprintf("%s: got incorrect body from response", tc.desc))
		authCall.Unset()
		repoCall.Unset()
	}
}

func createStateResponse(id int, tw twins.Twin, rec senml.Record) stateRes {
	return stateRes{
		TwinID:     tw.ID,
		ID:         int64(id),
		Definition: tw.Definitions[len(tw.Definitions)-1].ID,
		Payload:    map[string]interface{}{rec.BaseName: nil},
	}
}

func convState(data []stateRes) []twins.State {
	states := make([]twins.State, len(data))
	for i, d := range data {
		states[i] = twins.State{
			TwinID:     d.TwinID,
			ID:         d.ID,
			Definition: d.Definition,
			Payload:    d.Payload,
		}
	}
	return states
}
