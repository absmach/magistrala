// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"fmt"
	"net/http"
	"strings"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
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

func (sdk *mgSDK) SetContentType(ct ContentType) errors.SDKError {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return errors.NewSDKError(apiutil.ErrUnsupportedContentType)
	}

	sdk.msgContentType = ct

	return nil
}
