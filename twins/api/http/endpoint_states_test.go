// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/mainflux/mainflux/twins"
	"github.com/mainflux/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux/twins/mocks"
)

const numRecs = 100

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

func TestListStates(t *testing.T) {
	svc := mocks.NewService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	twin := twins.Twin{
		Owner: email,
	}
	def := mocks.CreateDefinition(channels[0:2], subtopics[0:2])
	tw, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	attr := def.Attributes[0]

	var recs = make([]senml.Record, numRecs)
	mocks.CreateSenML(numRecs, recs)
	message, err := mocks.CreateMessage(attr, recs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	err = svc.SaveStates(message)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var data []stateRes
	for i := 0; i < len(recs); i++ {
		res := createStateResponse(i, tw, recs[i])
		data = append(data, res)
	}

	baseURL := fmt.Sprintf("%s/states/%s", ts.URL, tw.ID)
	queryFmt := "%s?offset=%d&limit=%d"
	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []stateRes
	}{
		{
			desc:   "get a list of states",
			auth:   token,
			status: http.StatusOK,
			url:    baseURL,
			res:    data[0:10],
		},
		{
			desc:   "get a list of states with valid offset and limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf(queryFmt, baseURL, 20, 15),
			res:    data[20:35],
		},
		{
			desc:   "get a list of states with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf(queryFmt, baseURL, 0, 5),
			res:    nil,
		},
		{
			desc:   "get a list of states with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf(queryFmt, baseURL, 0, 5),
			res:    nil,
		},
		{
			desc:   "get a list of states with + limit > total",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf(queryFmt, baseURL, 91, 20),
			res:    data[91:],
		},
		{
			desc:   "get a list of states with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf(queryFmt, baseURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of states with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf(queryFmt, baseURL, 0, -5),
			res:    nil,
		},
		{
			desc:   "get a list of states with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf(queryFmt, baseURL, 0, 0),
			res:    nil,
		},
		{
			desc:   "get a list of states with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf(queryFmt, baseURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of states with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=invalid&limit=%d", baseURL, 15),
			res:    nil,
		},
		{
			desc:   "get a list of states with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=invalid", baseURL, 0),
			res:    nil,
		},
		{
			desc:   "get a list of states without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", baseURL, 15),
			res:    data[0:15],
		},
		{
			desc:   "get a list of states without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", baseURL, 14),
			res:    data[14:24],
		},
		{
			desc:   "get a list of states with invalid number of parameters",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", baseURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of states with redundant query parameters",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", baseURL, 0, 5),
			res:    data[0:5],
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var resData statesPageRes
		if tc.res != nil {
			err = json.NewDecoder(res.Body).Decode(&resData)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		}

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, resData.States, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, resData.States))
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
