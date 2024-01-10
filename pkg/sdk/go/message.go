// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/internal/apiutil"
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

	url := fmt.Sprintf("%s/channels/%s/messages%s", sdk.httpAdapterURL, chanID, subtopicPart)

	_, _, err := sdk.processRequest(http.MethodPost, url, ThingPrefix+key, []byte(msg), nil, http.StatusAccepted)

	return err
}

func (sdk mgSDK) ReadMessages(pm PageMetadata, chanName, token string) (MessagesPage, errors.SDKError) {
	chanNameParts := strings.SplitN(chanName, ".", channelParts)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == channelParts {
		subtopicPart = fmt.Sprintf("?subtopic=%s", chanNameParts[1])
	}

	readMessagesEndpoint := fmt.Sprintf("channels/%s/messages%s", chanID, subtopicPart)
	url, err := sdk.withQueryParams(sdk.readerURL, readMessagesEndpoint, pm)
	if err != nil {
		return MessagesPage{}, errors.NewSDKError(err)
	}

	header := make(map[string]string)
	header["Content-Type"] = string(sdk.msgContentType)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, header, http.StatusOK)
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
