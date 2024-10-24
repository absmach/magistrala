// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	chmocks "github.com/absmach/magistrala/channels/mocks"
	climocks "github.com/absmach/magistrala/clients/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	"github.com/absmach/magistrala/readers/api"
	"github.com/absmach/magistrala/readers/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	svcName       = "test-service"
	clientToken   = "1"
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
	v            float64 = 5
	vs                   = "value"
	vb                   = true
	vd                   = "dataValue"
	sum          float64 = 42
	domainID             = testsutil.GenerateUUID(&testing.T{})
	validSession         = mgauthn.Session{UserID: testsutil.GenerateUUID(&testing.T{})}
)

func newServer(repo *mocks.MessageRepository, authn *authnmocks.Authentication, clients *climocks.ClientsServiceClient, channels *chmocks.ChannelsServiceClient) *httptest.Server {
	mux := api.MakeHandler(repo, authn, clients, channels, svcName, instanceID)
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
		req.Header.Set("Authorization", apiutil.ClientPrefix+tr.key)
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
	authn := new(authnmocks.Authentication)
	clients := new(climocks.ClientsServiceClient)
	channels := new(chmocks.ChannelsServiceClient)
	ts := newServer(repo, authn, clients, channels)
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
		authnErr     error
		err          error
	}{
		{
			desc:         "read page with valid offset and limit",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=10", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=10", ts.URL, domainID, chanID),
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
			desc:   "read page as user without domain id",
			url:    fmt.Sprintf("%s/%s/channels/%s/messages", ts.URL, "", chanID),
			token:  userToken,
			status: http.StatusBadRequest,
		},
		{
			desc:         "read page with negative offset as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=-1&limit=10", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with negative limit as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=-10", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with zero limit as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=0", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer offset as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=abc&limit=10", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer limit as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=abc", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid channel id as client",
			url:          fmt.Sprintf("%s/channels//messages?offset=0&limit=10", ts.URL),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with multiple offset as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&offset=1&limit=10", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with multiple limit as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=20&limit=10", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with empty token as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0&limit=10", ts.URL, chanID),
			token:        "",
			authResponse: false,
			authnErr:     svcerr.ErrAuthentication,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:         "read page with default offset as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?limit=10", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with default limit as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?offset=0", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with senml format as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?format=messages", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Format: "messages"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with subtopic as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, chanID, subtopic, httpProt),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Subtopic: subtopic, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with subtopic and protocol as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, chanID, subtopic, httpProt),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Subtopic: subtopic, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with publisher as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?publisher=%s", ts.URL, chanID, pubID2),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Publisher: pubID2},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with protocol as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?protocol=http", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Protocol: httpProt},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with name as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?name=%s", ts.URL, chanID, msgName),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Name: msgName},
				Total:        uint64(len(queryMsgs)),
				Messages:     queryMsgs[0:10],
			},
		},
		{
			desc:         "read page with value as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f", ts.URL, chanID, v),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and equal comparator as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v, readers.EqualKey),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v, Comparator: readers.EqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and lower-than comparator as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v+1, readers.LowerThanKey),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v + 1, Comparator: readers.LowerThanKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and lower-than-or-equal comparator as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v+1, readers.LowerThanEqualKey),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v + 1, Comparator: readers.LowerThanEqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and greater-than comparator as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v-1, readers.GreaterThanKey),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v - 1, Comparator: readers.GreaterThanKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with value and greater-than-or-equal comparator as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, chanID, v-1, readers.GreaterThanEqualKey),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Value: v - 1, Comparator: readers.GreaterThanEqualKey},
				Total:        uint64(len(valueMsgs)),
				Messages:     valueMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-float value as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=ab01", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with value and wrong comparator as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?v=%f&comparator=wrong", ts.URL, chanID, v-1),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with boolean value as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?vb=true", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", BoolValue: true},
				Total:        uint64(len(boolMsgs)),
				Messages:     boolMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-boolean value as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?vb=yes", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with string value as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?vs=%s", ts.URL, chanID, vs),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", StringValue: vs},
				Total:        uint64(len(stringMsgs)),
				Messages:     stringMsgs[0:10],
			},
		},
		{
			desc:         "read page with data value as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?vd=%s", ts.URL, chanID, vd),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", DataValue: vd},
				Total:        uint64(len(dataMsgs)),
				Messages:     dataMsgs[0:10],
			},
		},
		{
			desc:         "read page with non-float from as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?from=ABCD", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-float to as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?to=ABCD", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with from/to as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", From: messages[19].Time, To: messages[4].Time},
				Total:        uint64(len(messages[5:20])),
				Messages:     messages[5:15],
			},
		},
		{
			desc:         "read page with aggregation as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with interval as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?interval=10h", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Interval: "10h"},
				Total:        uint64(len(messages)),
				Messages:     messages[0:10],
			},
		},
		{
			desc:         "read page with aggregation and interval as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h", ts.URL, chanID),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval, to and from as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusOK,
			res: pageRes{
				PageMetadata: readers.PageMetadata{Limit: 10, Format: "messages", Aggregation: "MAX", Interval: "10h", From: messages[19].Time, To: messages[4].Time},
				Total:        uint64(len(messages[5:20])),
				Messages:     messages[5:15],
			},
		},
		{
			desc:         "read page with invalid aggregation and valid interval, to and from as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=invalid&interval=10h&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid interval and valid aggregation, to and from as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10hrs&from=%f&to=%f", ts.URL, chanID, messages[19].Time, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with missing from as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&to=%f", ts.URL, chanID, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid from as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&to=ABCD&from=%f", ts.URL, chanID, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid to as client",
			url:          fmt.Sprintf("%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=ABCD", ts.URL, chanID, messages[4].Time),
			key:          clientToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with valid offset and limit as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=10", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=-1&limit=10", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with negative limit as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=-10", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with zero limit as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=0", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer offset as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=abc&limit=10", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-integer limit as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=abc", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid channel id as user",
			url:          fmt.Sprintf("%s/%s/channels//messages?offset=0&limit=10", ts.URL, domainID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid token as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=10", ts.URL, domainID, chanID),
			token:        invalidToken,
			authResponse: false,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "read page with multiple offset as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&offset=1&limit=10", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with multiple limit as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=20&limit=10", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with empty token as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0&limit=10", ts.URL, domainID, chanID),
			token:        "",
			authResponse: false,
			status:       http.StatusUnauthorized,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "read page with default offset as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?limit=10", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?offset=0", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?format=messages", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, domainID, chanID, subtopic, httpProt),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?subtopic=%s&protocol=%s", ts.URL, domainID, chanID, subtopic, httpProt),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?publisher=%s", ts.URL, domainID, chanID, pubID2),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?protocol=http", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?name=%s", ts.URL, domainID, chanID, msgName),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f", ts.URL, domainID, chanID, v),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, domainID, chanID, v, readers.EqualKey),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, domainID, chanID, v+1, readers.LowerThanKey),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, domainID, chanID, v+1, readers.LowerThanEqualKey),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, domainID, chanID, v-1, readers.GreaterThanKey),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f&comparator=%s", ts.URL, domainID, chanID, v-1, readers.GreaterThanEqualKey),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=ab01", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with value and wrong comparator as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?v=%f&comparator=wrong", ts.URL, domainID, chanID, v-1),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with boolean value as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?vb=true", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?vb=yes", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with string value as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?vs=%s", ts.URL, domainID, chanID, vs),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?vd=%s", ts.URL, domainID, chanID, vd),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?from=ABCD", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with non-float to as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?to=ABCD", ts.URL, domainID, chanID),
			token:        userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with from/to as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?from=%f&to=%f", ts.URL, domainID, chanID, messages[19].Time, messages[4].Time),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX", ts.URL, domainID, chanID),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with interval as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?interval=10h", ts.URL, domainID, chanID),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX&interval=10h", ts.URL, domainID, chanID),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval, to and from as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=%f", ts.URL, domainID, chanID, messages[19].Time, messages[4].Time),
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
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=invalid&interval=10h&from=%f&to=%f", ts.URL, domainID, chanID, messages[19].Time, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with invalid interval and valid aggregation, to and from as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX&interval=10hrs&from=%f&to=%f", ts.URL, domainID, chanID, messages[19].Time, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with missing from as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX&interval=10h&to=%f", ts.URL, domainID, chanID, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid from as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX&interval=10h&to=ABCD&from=%f", ts.URL, domainID, chanID, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
		{
			desc:         "read page with aggregation, interval and to with invalid to as user",
			url:          fmt.Sprintf("%s/%s/channels/%s/messages?aggregation=MAX&interval=10h&from=%f&to=ABCD", ts.URL, domainID, chanID, messages[4].Time),
			key:          userToken,
			authResponse: true,
			status:       http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		repo.On("ReadAll", chanID, tc.res.PageMetadata).Return(readers.MessagesPage{Total: tc.res.Total, Messages: fromSenml(tc.res.Messages)}, nil)
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
