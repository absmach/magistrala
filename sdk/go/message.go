//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"fmt"
	"net/http"
	"strings"
)

func (sdk mfSDK) SendMessage(chanID, msg, token string) error {
	endpoint := fmt.Sprintf("channels/%s/messages", chanID)
	url := createURL(sdk.url, sdk.httpAdapterPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(msg))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidArgs
		case http.StatusForbidden:
			return ErrUnauthorized
		default:
			return ErrFailedPublish
		}
	}

	return nil
}

func (sdk *mfSDK) SetContentType(ct ContentType) error {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return ErrInvalidContentType
	}

	sdk.msgContentType = ct

	return nil
}
