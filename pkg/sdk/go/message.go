// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
)

const channelParts = 2

func (sdk mgSDK) SendMessage(chanName, msg, key string) errors.SDKError {
	chanNameParts := strings.SplitN(chanName, ".", channelParts)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == channelParts {
		subtopicPart = fmt.Sprintf("/%s", strings.ReplaceAll(chanNameParts[1], ".", "/"))
	}

	reqURL := fmt.Sprintf("%s/channels/%s/messages%s", sdk.httpAdapterURL, chanID, subtopicPart)

	_, _, err := sdk.processRequest(http.MethodPost, reqURL, ClientPrefix+key, []byte(msg), nil, http.StatusAccepted)

	return err
}

func (sdk mgSDK) ReadMessages(pm MessagePageMetadata, chanName, domainID, token string) (MessagesPage, errors.SDKError) {
	chanNameParts := strings.SplitN(chanName, ".", channelParts)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == channelParts {
		subtopicPart = fmt.Sprintf("?subtopic=%s", chanNameParts[1])
	}

	readMessagesEndpoint := fmt.Sprintf("%s/channels/%s/messages%s", domainID, chanID, subtopicPart)
	msgURL, err := sdk.withMessageQueryParams(sdk.readerURL, readMessagesEndpoint, pm)
	if err != nil {
		return MessagesPage{}, errors.NewSDKError(err)
	}

	header := make(map[string]string)
	header["Content-Type"] = string(sdk.msgContentType)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, msgURL, token, nil, header, http.StatusOK)
	if sdkerr != nil {
		return MessagesPage{}, sdkerr
	}

	var mp MessagesPage
	if err := json.Unmarshal(body, &mp); err != nil {
		return MessagesPage{}, errors.NewSDKError(err)
	}

	return mp, nil
}

func (sdk *mgSDK) SetContentType(ct ContentType) errors.SDKError {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return errors.NewSDKError(apiutil.ErrUnsupportedContentType)
	}

	sdk.msgContentType = ct

	return nil
}

func (sdk mgSDK) withMessageQueryParams(baseURL, endpoint string, mpm MessagePageMetadata) (string, error) {
	b, err := json.Marshal(mpm)
	if err != nil {
		return "", err
	}
	q := map[string]interface{}{}
	if err := json.Unmarshal(b, &q); err != nil {
		return "", err
	}
	ret := url.Values{}
	for k, v := range q {
		switch t := v.(type) {
		case string:
			ret.Add(k, t)
		case float64:
			ret.Add(k, strconv.FormatFloat(t, 'f', -1, 64))
		case uint64:
			ret.Add(k, strconv.FormatUint(t, 10))
		case int64:
			ret.Add(k, strconv.FormatInt(t, 10))
		case json.Number:
			ret.Add(k, t.String())
		case bool:
			ret.Add(k, strconv.FormatBool(t))
		}
	}
	qs := ret.Encode()

	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, qs), nil
}
