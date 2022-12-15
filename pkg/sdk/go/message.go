// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
)

func (sdk mfSDK) SendMessage(chanName, msg, key string) errors.SDKError {
	chanNameParts := strings.SplitN(chanName, ".", 2)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == 2 {
		subtopicPart = fmt.Sprintf("/%s", strings.Replace(chanNameParts[1], ".", "/", -1))
	}

	url := fmt.Sprintf("%s/channels/%s/messages/%s", sdk.httpAdapterURL, chanID, subtopicPart)

	_, _, err := sdk.processRequest(http.MethodPost, url, apiutil.ThingPrefix+key, string(CTJSON), []byte(msg), http.StatusAccepted)
	return err
}

func (sdk mfSDK) ReadMessages(chanName, token string) (MessagesPage, errors.SDKError) {
	chanNameParts := strings.SplitN(chanName, ".", 2)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == 2 {
		subtopicPart = fmt.Sprintf("?subtopic=%s", strings.Replace(chanNameParts[1], ".", "/", -1))
	}

	url := fmt.Sprintf("%s/channels/%s/messages%s", sdk.readerURL, chanID, subtopicPart)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(sdk.msgContentType), nil, http.StatusOK)
	if err != nil {
		return MessagesPage{}, err
	}

	var mp MessagesPage
	if err := json.Unmarshal(body, &mp); err != nil {
		return MessagesPage{}, errors.NewSDKError(err)
	}

	return mp, nil
}

func (sdk *mfSDK) SetContentType(ct ContentType) errors.SDKError {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return errors.NewSDKError(errors.ErrUnsupportedContentType)
	}

	sdk.msgContentType = ct
	return nil
}
