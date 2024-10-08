// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	"github.com/absmach/magistrala/readers/api"
	"github.com/absmach/magistrala/readers/mocks"
	thmocks "github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	svcName       = "test-service"
	thingToken    = "1"
	userToken     = "token"
	invalidToken  = "invalid"
	email         = "user@example.com"
	invalid       = "invalid"
	numOfMessages = 100
	valueFields   = 5
	subtopic      = "topic"
	mqttProt      = "mqtt"
	httpProt      = "http"
	msgName       = "temperature"
	instanceID    = "5de9b29a-feb9-11ed-be56-0242ac120002"
)

var (
	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42
)

func newServer(repo *mocks.MessageRepository, authClient *authmocks.AuthClient, thingsAuthzClient *thmocks.AuthzServiceClient) *httptest.Server {
	mux := api.MakeHandler(repo, authClient, thingsAuthzClient, svcName, instanceID)
	return httptest.NewServer(mux)
}

type testRequest struct {
	client *http.Client
	method string
	url    string
	token  string
	key    string
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, http.NoBody)
	if err != nil {
		return nil, err
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.key != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.key)
	}

	return tr.client.Do(req)
}

func TestReadAll(t *testing.T) {
	chanID := testsutil.GenerateUUID(t)
	pubID := testsutil.GenerateUUID(t)
	pubID2 := testsutil.GenerateUUID(t)

	now := time.Now().Unix()

	var messages []senml.Message
	var queryMsgs []senml.Message
	var valueMsgs []senml.Message
	var boolMsgs []senml.Message
	var stringMsgs []senml.Message
	var dataMsgs []senml.Message

	for i := 0; i < numOfMessages; i++ {
		// Mix possible values as well as value sum.
		msg := senml.Message{
			Channel:   chanID,
			Publisher: pubID,
			Protocol:  mqttProt,
			Time:      float64(now - int64(i)),
			Name:      "name",
		}

		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &v
			valueMsgs = append(valueMsgs, msg)
		case 1:
			msg.BoolValue = &vb
			boolMsgs = append(boolMsgs, msg)
		case 2:
			msg.StringValue = &vs
			stringMsgs = append(stringMsgs, msg)
		case 3:
			msg.DataValue = &vd
			dataMsgs = append(dataMsgs, msg)
		case 4:
			msg.Sum = &sum
			msg.Subtopic = subtopic
			msg.Protocol = httpProt
			msg.Publisher = pubID2
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}

		messages = append(messages, msg)
	}

	repo := new(mocks.MessageRepository)
	auth := new(authmocks.AuthClient)
	things := new(thmocks.AuthzServiceClient)
	ts := newServer(repo, auth, things)
	defer ts.Close()

	cases := []struct {
		desc         string
		req          string
		url          string
		token        string
		key          string
		authResponse bool
		status       int
		res          pageRes
		err          error
	}{
		{
			desc:         "read page with valid offset and limit",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with valid offset and limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with negative offset as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=-1&limit=10", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with negative limit as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=-10", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with zero limit as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=0", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer offset as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=abc&limit=10", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer limit as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=abc", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid channel id as thing",
			url:          fmt.Sprintf("%s/channels//messages?offset=0&limit=10", ts.URL),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid token as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        invalidToken,
			authResponse: false,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "read page with multiple offset as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&offset=1&limit=10", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with multiple limit as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=20&limit=10", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with empty token as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        "",
			authResponse: false,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "read page with default offset as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?limit=10", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with default limit as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with senml format as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?format=messages", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Format: "messages"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with subtopic as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, chanID, subtopic, httpProt),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Subtopic: subtopic, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with subtopic and protocol as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, chanID, subtopic, httpProt),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Subtopic: subtopic, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with publisher as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?publisher=%s", ts.URL, chanID, pubID2),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Publisher: pubID2},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},

		{
			desc:         "read page with protocol as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?protocol=http", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with name as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?name=%s", ts.URL, chanID, msgName),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Name: msgName},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with value as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f", ts.URL, chanID, v),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and equal comparator as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v, readers.EqualKey),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v, Comparator: readers.EqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and lower-than comparator as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v+1, readers.LowerThanKey),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v + 1, Comparator: readers.LowerThanKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and lower-than-or-equal comparator as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v+1, readers.LowerThanEqualKey),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v + 1, Comparator: readers.LowerThanEqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and greater-than comparator as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v-1, readers.GreaterThanKey),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v - 1, Comparator: readers.GreaterThanKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and greater-than-or-equal comparator as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v-1, readers.GreaterThanEqualKey),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v - 1, Comparator: readers.GreaterThanEqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-float value as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=ab01", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with value and wrong comparator as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=wrong", ts.URL, chanID, v-1),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with boolean value as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?vb=true", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", BoolValue: true},
				Total:        uint64(len(boolMsgs)),
				Messages:     boolMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-boolean value as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?vb=yes", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with string value as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?vs=%s", ts.URL, chanID, vs),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", StringValue: vs},
				Total:        uint64(len(stringMsgs)),
				Messages:     stringMsgs[0:10],
			},
		},
		{
			desc:         "read page with data value as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?vd=%s", ts.URL, chanID, vd),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", DataValue: vd},
				Total:        uint64(len(dataMsgs)),
				Messages:     dataMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-float from as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?from=ABCD", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-float to as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?to=ABCD", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with from/to as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", From: messages[19].Time, To: messages[4].Time},
				Total:        uint64(len(messages[5:20])),
				Messages:     messages[5:15],
			},
		},
		{
			desc:         "read page with aggregation as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with interval as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?interval=10h", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Interval: "10h"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with aggregation and interval as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h", ts.URL, chanID),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval, to and from as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Aggregation: "MAX", Interval: "10h", From: messages[19].Time, To: messages[4].Time},
				Total:        uint64(len(messages[5:20])),
				Messages:     messages[5:15],
			},
		},
		{
			desc:         "read page with invalid aggregation and valid interval, to and from as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=invalid&interval=10h&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid interval and valid aggregation, to and from as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10hrs&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with missing from as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&to=%f", ts.URL, chanID, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid from as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&to=ABCD&from=%f", ts.URL, chanID, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid to as thing",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=ABCD", ts.URL, chanID, messages[4].Time),
			key:          thingToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with valid offset and limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with negative offset as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=-1&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with negative limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=-10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with zero limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=0", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer offset as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=abc&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=abc", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid channel id as user",
			url:          fmt.Sprintf("%s/channels//messages?offset=0&limit=10", ts.URL),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid token as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        invalidToken,
			authResponse: false,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "read page with multiple offset as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&offset=1&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with multiple limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=20&limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with empty token as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        "",
			authResponse: false,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "read page with default offset as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?limit=10", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with default limit as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with senml format as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?format=messages", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Format: "messages"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with subtopic as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, chanID, subtopic, httpProt),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Subtopic: subtopic, Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with subtopic and protocol as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, chanID, subtopic, httpProt),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Subtopic: subtopic, Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with publisher as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?publisher=%s", ts.URL, chanID, pubID2),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Publisher: pubID2},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with protocol as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?protocol=http", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with name as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?name=%s", ts.URL, chanID, msgName),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Name: msgName},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with value as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f", ts.URL, chanID, v),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and equal comparator as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v, readers.EqualKey),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v, Comparator: readers.EqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and lower-than comparator as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v+1, readers.LowerThanKey),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v + 1, Comparator: readers.LowerThanKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and lower-than-or-equal comparator as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v+1, readers.LowerThanEqualKey),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v + 1, Comparator: readers.LowerThanEqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and greater-than comparator as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v-1, readers.GreaterThanKey),
			token:        userToken,
			status:       http.StatusOK,
			authResponse: true,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v - 1, Comparator: readers.GreaterThanKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and greater-than-or-equal comparator as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v-1, readers.GreaterThanEqualKey),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v - 1, Comparator: readers.GreaterThanEqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-float value as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=ab01", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with value and wrong comparator as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=wrong", ts.URL, chanID, v-1),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with boolean value as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?vb=true", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", BoolValue: true},
				Total:        uint64(len(boolMsgs)),
				Messages:     boolMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-boolean value as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?vb=yes", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with string value as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?vs=%s", ts.URL, chanID, vs),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", StringValue: vs},
				Total:        uint64(len(stringMsgs)),
				Messages:     stringMsgs[0:10],
			},
		},
		{
			desc:         "read page with data value as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?vd=%s", ts.URL, chanID, vd),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", DataValue: vd},
				Total:        uint64(len(dataMsgs)),
				Messages:     dataMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-float from as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?from=ABCD", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-float to as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?to=ABCD", ts.URL, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with from/to as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			token:        userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", From: messages[19].Time, To: messages[4].Time},
				Total:        uint64(len(messages[5:20])),
				Messages:     messages[5:15],
			},
		},
		{
			desc:         "read page with aggregation as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX", ts.URL, chanID),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with interval as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?interval=10h", ts.URL, chanID),
			key:          userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Interval: "10h"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with aggregation and interval as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h", ts.URL, chanID),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval, to and from as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Aggregation: "MAX", Interval: "10h", From: messages[19].Time, To: messages[4].Time},
				Total:        uint64(len(messages[5:20])),
				Messages:     messages[5:15],
			},
		},
		{
			desc:         "read page with invalid aggregation and valid interval, to and from as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=invalid&interval=10h&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid interval and valid aggregation, to and from as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10hrs&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with missing from as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&to=%f", ts.URL, chanID, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid from as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&to=ABCD&from=%f", ts.URL, chanID, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid to as user",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=ABCD", ts.URL, chanID, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", context.Background(), mock.Anything).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		authCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: tc.authResponse}, tc.err)
		repo.On("ReadAll", chanID, tc.res.PageMetadata).Return(readers.MessagesPage{Total: tc.res.Total, Messages: fromSenml(tc.res.Messages)}, nil)
		if tc.key != "" {
			repoCall = things.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: tc.authResponse}, tc.err)
		}
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
			key:    tc.key,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var page pageRes
		err = json.NewDecoder(res.Body).Decode(&page)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))

		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res.Total, page.Total, fmt.Sprintf("%s: expected %d got %d", tc.desc, tc.res.Total, page.Total))
		assert.ElementsMatch(t, tc.res.Messages, page.Messages, fmt.Sprintf("%s: got incorrect body from response", tc.desc))
		repoCall.Unset()
		authCall.Unset()
	}
}

type pageRes struct {
	readers.PageMetadata
	Total    uint64          `json:"total"`
	Messages []senml.Message `json:"messages,omitempty"`
}

func fromSenml(in []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range in {
		ret = append(ret, m)
	}
	return ret
}
